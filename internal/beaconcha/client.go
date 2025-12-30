package beaconcha

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client wraps calls to beaconcha and enforces a 1 req/sec rate limit
type Client struct {
	baseURL string
	client  *http.Client
	reqCh   chan request
	once    sync.Once
	apiKey  string
	chain   string
}

func NewClient(baseURL string) *Client {
	c := &Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
		reqCh:   make(chan request, 100),
		chain:   "mainnet",
	}
	c.startWorker()
	return c
}

// Simple representation of the upstream response we need
type ValidatorOverview struct {
	Slashed                   bool    `json:"slashed"`
	Status                    string  `json:"status"`
	WithdrawalCredentialsType string  `json:"withdrawal_credentials_type"`
	WithdrawalCredentials     string  `json:"withdrawal_credentials"`
	ActivationEpoch           int64   `json:"activation_epoch"`
	ExitEpoch                 int64   `json:"exit_epoch"`
	CurrentBalance            uint64  `json:"current_balance"`
	CurrentBalanceRaw         string  `json:"current_balance_raw"`
	Online                    bool    `json:"online"`
	BeaconScore               float64 `json:"beacon_score"`
}

// HumanAmount represents both raw and human-friendly formatted amounts
type HumanAmount struct {
	Raw  string `json:"raw"`
	ETH  string `json:"eth"`  // decimal representation in ETH
	Gwei string `json:"gwei"` // decimal representation in Gwei
}

type request struct {
	method string
	path   string
	body   interface{}
	resp   chan result
}

type result struct {
	raw []byte
	err error
}

func (c *Client) startWorker() {
	c.once.Do(func() {
		// rateCh receives one token per second to strictly enforce 1 req/sec
		rateCh := make(chan struct{}, 1)
		go func() {
			t := time.NewTicker(1 * time.Second)
			defer t.Stop()
			for range t.C {
				select {
				case rateCh <- struct{}{}:
				default:
				}
			}
		}()

		go func() {
			var cooldownUntil time.Time
			const cooldownMargin = 500 * time.Millisecond
			for req := range c.reqCh {
				// if we're in a global cooldown (set after a 429), wait until it expires
				if !cooldownUntil.IsZero() {
					now := time.Now()
					if now.Before(cooldownUntil) {
						sleep := cooldownUntil.Sub(now)
						fmt.Printf("beaconcha.cooldown wait=%s until=%s\n", sleep, cooldownUntil.UTC().Format(time.RFC3339Nano))
						time.Sleep(sleep)
					}
				}

				res := result{}

				// prepare body bytes
				var rawBody []byte
				if req.body != nil {
					rb, jerr := json.Marshal(req.body)
					if jerr != nil {
						res.err = jerr
						req.resp <- res
						continue
					}
					rawBody = rb
				}

				// normalize URL to avoid duplicate segments
				p := req.path
				if !strings.HasPrefix(p, "/") {
					p = "/" + p
				}
				var url string
				if strings.HasSuffix(c.baseURL, "/api/v2") && strings.HasPrefix(p, "/api/v2") {
					url = strings.TrimSuffix(c.baseURL, "/api/v2") + p
				} else {
					url = strings.TrimRight(c.baseURL, "/") + p
				}

				// truncated body for logging
				bodyStr := ""
				if len(rawBody) > 0 {
					bs := string(rawBody)
					if len(bs) > 1024 {
						bodyStr = bs[:1024] + "..."
					} else {
						bodyStr = bs
					}
				}
				fmt.Printf("beaconcha.request %s %s %s body=%s\n", time.Now().UTC().Format(time.RFC3339Nano), req.method, url, bodyStr)

				var resp *http.Response
				var attempt int
				for {
					// wait for rate token (1 request per second)
					<-rateCh
					fmt.Printf("beaconcha.token %s\n", time.Now().UTC().Format(time.RFC3339Nano))

					// create a fresh http.Request for each attempt so the body is sent each time
					var httpReq *http.Request
					var err error
					if len(rawBody) > 0 {
						httpReq, err = http.NewRequest(req.method, url, bytes.NewReader(rawBody))
					} else {
						httpReq, err = http.NewRequest(req.method, url, nil)
					}
					if err != nil {
						res.err = err
						req.resp <- res
						break
					}
					httpReq.Header.Set("Content-Type", "application/json")
					httpReq.Header.Set("User-Agent", "validator-dashboard/1.0")
					if c.apiKey != "" {
						httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
					}

					// log masked authorization presence
					authMask := "none"
					if httpReq.Header.Get("Authorization") != "" {
						a := httpReq.Header.Get("Authorization")
						if len(a) > 10 {
							authMask = a[:10] + "..."
						} else {
							authMask = "set"
						}
					}
					fmt.Printf("beaconcha.request.headers %s Authorization=%s\n", time.Now().UTC().Format(time.RFC3339Nano), authMask)

					attempt++
					resp, err = c.client.Do(httpReq)
					if err != nil {
						res.err = err
						req.resp <- res
						break
					}
					if resp.StatusCode == http.StatusTooManyRequests && attempt <= 3 {
						// read Retry-After header if present
						ra := resp.Header.Get("Retry-After")
						resp.Body.Close()
						wait := 1 * time.Second
						if ra != "" {
							if secsInt, err := strconv.Atoi(ra); err == nil {
								wait = time.Duration(secsInt) * time.Second
							} else if d, perr := time.ParseDuration(ra); perr == nil {
								wait = d
							}
						}
						if wait < time.Second {
							wait = time.Second
						}
						cooldownUntil = time.Now().Add(wait + cooldownMargin)
						fmt.Printf("beaconcha.retry %s status=429 attempt=%d wait=%s cooldown_until=%s\n", time.Now().UTC().Format(time.RFC3339Nano), attempt, wait, cooldownUntil.UTC().Format(time.RFC3339Nano))
						time.Sleep(wait + cooldownMargin)
						continue
					}
					break
				}
				if resp == nil {
					continue
				}
				b, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					res.err = err
					req.resp <- res
					continue
				}
				if resp.StatusCode != http.StatusOK {
					rb := string(b)
					if len(rb) > 1024 {
						rb = rb[:1024] + "..."
					}
					fmt.Printf("beaconcha.response %s status=%d body=%s\n", time.Now().UTC().Format(time.RFC3339Nano), resp.StatusCode, rb)
					res.err = fmt.Errorf("upstream status %d: %s", resp.StatusCode, rb)
					req.resp <- res
					continue
				}
				res.raw = b
				req.resp <- res
			}
		}()
	})
}

// SetAPIKey sets an Authorization header value for outgoing requests
func (c *Client) SetAPIKey(key string) {
	c.apiKey = key
}

// SetChain sets the chain used in request bodies (e.g., "mainnet" or "hoodi").
func (c *Client) SetChain(chain string) {
	if chain == "" {
		chain = "mainnet"
	}
	c.chain = chain
}

// BatchFetchOverview posts to /ethereum/validators and returns a map[id]ValidatorOverview
func (c *Client) BatchFetchOverview(ids []int) (map[int]ValidatorOverview, error) {
	const chunk = 100
	out := make(map[int]ValidatorOverview)
	for i := 0; i < len(ids); i += chunk {
		end := i + chunk
		if end > len(ids) {
			end = len(ids)
		}
		body := map[string]interface{}{
			"validator": map[string]interface{}{"validator_identifiers": ids[i:end]},
			"chain":     c.chain,
			"page_size": 10,
		}
		raw, err := c.queueRawRequest("POST", "/api/v2/ethereum/validators", body)
		if err != nil {
			return out, err
		}
		var envelope struct {
			Data []json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return out, err
		}
		for _, d := range envelope.Data {
			var item struct {
				Validator struct {
					Index int `json:"index"`
				} `json:"validator"`
				Slashed               bool   `json:"slashed"`
				Status                string `json:"status"`
				WithdrawalCredentials struct {
					Type    string `json:"type"`
					Address string `json:"address"`
				} `json:"withdrawal_credentials"`
				LifeCycleEpochs struct {
					Activation int64 `json:"activation"`
					Exit       int64 `json:"exit"`
				} `json:"life_cycle_epochs"`
				Balances struct {
					Current string `json:"current"`
				} `json:"balances"`
				Online      bool    `json:"online"`
				BeaconScore float64 `json:"beacon_score"`
			}
			if err := json.Unmarshal(d, &item); err != nil {
				continue
			}
			out[item.Validator.Index] = ValidatorOverview{
				Slashed:                   item.Slashed,
				Status:                    item.Status,
				WithdrawalCredentialsType: item.WithdrawalCredentials.Type,
				WithdrawalCredentials:     item.WithdrawalCredentials.Address,
				ActivationEpoch:           item.LifeCycleEpochs.Activation,
				ExitEpoch:                 item.LifeCycleEpochs.Exit,
				CurrentBalance:            0,
				CurrentBalanceRaw:         item.Balances.Current,
				Online:                    item.Online,
				BeaconScore:               item.BeaconScore,
			}
		}
	}
	return out, nil
}

// BatchFetchRewards calls /validators/rewards-aggregate and returns aggregated totals for the provided validators.
func (c *Client) BatchFetchRewards(ids []int, evaluationWindow string) (map[string]string, error) {
	bodyAgg := map[string]interface{}{
		"validator": map[string]interface{}{"validator_identifiers": ids},
		"range":     map[string]interface{}{"evaluation_window": evaluationWindow},
		"chain":     c.chain,
	}
	rawAgg, aggErr := c.queueRawRequest("POST", "/api/v2/ethereum/validators/rewards-aggregate", bodyAgg)
	if aggErr != nil {
		return nil, aggErr
	}
	var envelope struct {
		Data struct {
			Total        string `json:"total"`
			TotalReward  string `json:"total_reward"`
			TotalPenalty string `json:"total_penalty"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rawAgg, &envelope); err != nil {
		return nil, err
	}
	out := map[string]string{
		"total":         envelope.Data.Total,
		"total_reward":  envelope.Data.TotalReward,
		"total_penalty": envelope.Data.TotalPenalty,
	}
	return out, nil
}

// BatchFetchPerformance calls /validators/performance-aggregate and returns aggregated metrics for the provided validators.
func (c *Client) BatchFetchPerformance(ids []int, evaluationWindow string) (map[string]interface{}, error) {
	bodyAgg := map[string]interface{}{
		"validator": map[string]interface{}{"validator_identifiers": ids},
		"range":     map[string]interface{}{"evaluation_window": evaluationWindow},
		"chain":     c.chain,
	}
	rawAgg, aggErr := c.queueRawRequest("POST", "/api/v2/ethereum/validators/performance-aggregate", bodyAgg)
	if aggErr != nil {
		return nil, aggErr
	}
	var envAgg struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(rawAgg, &envAgg); err != nil {
		return nil, err
	}
	return envAgg.Data, nil
}

// FetchValidator posts to /api/v2/ethereum/validators with a single id and parses data[0]
func (c *Client) FetchValidator(id int) (ValidatorOverview, error) {
	if id <= 0 {
		return ValidatorOverview{}, errors.New("invalid id")
	}
	body := map[string]interface{}{
		"validator": map[string]interface{}{"validator_identifiers": []int{id}},
		"chain":     c.chain,
		"page_size": 10,
	}
	raw, err := c.queueRawRequest("POST", "/api/v2/ethereum/validators", body)
	if err != nil {
		return ValidatorOverview{}, err
	}
	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ValidatorOverview{}, err
	}
	if len(envelope.Data) == 0 {
		return ValidatorOverview{}, errors.New("no data in response")
	}
	var item struct {
		Slashed               bool   `json:"slashed"`
		Status                string `json:"status"`
		WithdrawalCredentials struct {
			Type    string `json:"type"`
			Address string `json:"address"`
		} `json:"withdrawal_credentials"`
		LifeCycleEpochs struct {
			Activation int64 `json:"activation"`
			Exit       int64 `json:"exit"`
		} `json:"life_cycle_epochs"`
		Balances struct {
			Current string `json:"current"`
		} `json:"balances"`
		Online      bool    `json:"online"`
		BeaconScore float64 `json:"beacon_score"`
	}
	if err := json.Unmarshal(envelope.Data[0], &item); err != nil {
		return ValidatorOverview{}, err
	}
	out := ValidatorOverview{
		Slashed:                   item.Slashed,
		Status:                    item.Status,
		WithdrawalCredentialsType: item.WithdrawalCredentials.Type,
		WithdrawalCredentials:     item.WithdrawalCredentials.Address,
		ActivationEpoch:           item.LifeCycleEpochs.Activation,
		ExitEpoch:                 item.LifeCycleEpochs.Exit,
		CurrentBalance:            0,
		CurrentBalanceRaw:         item.Balances.Current,
		Online:                    item.Online,
		BeaconScore:               item.BeaconScore,
	}
	return out, nil
}

// ParseBigToHuman parses a decimal integer string (wei or gwei) into Gwei and ETH strings.
func ParseBigToHuman(s string) (gweiStr, ethStr string) {
	bi := new(big.Int)
	if _, ok := bi.SetString(s, 10); !ok {
		return "0", "0"
	}
	gweiDiv := new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil)
	ethDiv := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	f := new(big.Float).SetInt(bi)
	gweiF := new(big.Float).Quo(f, new(big.Float).SetInt(gweiDiv))
	ethF := new(big.Float).Quo(f, new(big.Float).SetInt(ethDiv))
	gweiStr = gweiF.Text('f', 6)
	ethStr = ethF.Text('f', 9)
	return
}

// getHeadEpoch attempts to fetch current head epoch from known endpoints.
func (c *Client) getHeadEpoch() (int64, error) {
	paths := []string{"/api/v2/ethereum/chain/head", "/api/v2/ethereum/status"}
	for _, p := range paths {
		raw, err := c.queueRawRequest("GET", p, nil)
		if err != nil {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}
		if data, ok := obj["data"].(map[string]interface{}); ok {
			if head, ok2 := data["head_epoch"].(float64); ok2 {
				return int64(head), nil
			}
			if epoch, ok2 := data["epoch"].(float64); ok2 {
				return int64(epoch), nil
			}
		}
	}
	return 0, errors.New("could not determine head epoch")
}

// queueRawRequest sends a request via the worker and returns raw response bytes
func (c *Client) queueRawRequest(method, path string, body interface{}) ([]byte, error) {
	req := request{method: method, path: path, body: body, resp: make(chan result, 1)}
	select {
	case c.reqCh <- req:
	default:
		return nil, errors.New("beaconcha client queue full")
	}
	select {
	case res := <-req.resp:
		return res.raw, res.err
	case <-time.After(30 * time.Second):
		return nil, errors.New("beaconcha request timeout")
	}
}
