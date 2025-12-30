package validator

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Marketen/validator-dashboard-beaconcha/internal/beaconcha"
	"github.com/Marketen/validator-dashboard-beaconcha/internal/cache"
)

// Request types
type ValidatorRequest struct {
	ValidatorId []int `json:"validatorId"`
	// Range selects the evaluation window for rewards/performance. Allowed: "24h","7d","30d","90d","all_time".
	Range string `json:"range,omitempty"`
}

// Response types (simplified for initial version)
type ValidatorOverview struct {
	ID                        int       `json:"id"`
	Slashed                   bool      `json:"slashed"`
	Status                    string    `json:"status"`
	WithdrawalCredentialsType string    `json:"withdrawal_credentials_type"`
	WithdrawalCredentials     string    `json:"withdrawal_credentials"`
	ActivationEpoch           int64     `json:"activation_epoch"`
	ExitEpoch                 int64     `json:"exit_epoch"`
	CurrentBalanceRaw         string    `json:"current_balance_raw"`
	CurrentBalanceHuman       string    `json:"current_balance_human"`
	Online                    bool      `json:"online"`
	FetchedAt                 time.Time `json:"fetched_at"`
}

type ValidatorRewards struct {
	TotalRaw   string `json:"total_raw"`
	TotalHuman string `json:"total_human"`
}

type ValidatorPerformance struct {
	BeaconScore float64 `json:"beacon_score"`
}

type ValidatorResponse struct {
	Overview    ValidatorOverview    `json:"overview"`
	Rewards     ValidatorRewards     `json:"rewards"`
	Performance ValidatorPerformance `json:"performance"`
}

type ValidatorBatchResponse struct {
	Successes  map[int]ValidatorResponse `json:"successes"`
	Errors     map[int]string            `json:"errors"`
	Aggregates map[string]interface{}    `json:"aggregates,omitempty"`
}

var bcClient = beaconcha.NewClient("https://beaconcha.in/api/v2")
var memCache = cache.NewMemoryCache(20 * time.Minute)

func init() {
	if k := getEnv("BEACONCHA_API_KEY", ""); k != "" {
		bcClient.SetAPIKey(k)
	}
}

// SetChain forwards chain selection to the internal beaconcha client
func SetChain(chain string) {
	if chain == "" {
		return
	}
	bcClient.SetChain(chain)
}

// getEnv helper to avoid importing os repeatedly
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req ValidatorRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Validate IDs: 1-100 unique, max 100
	if len(req.ValidatorId) == 0 || len(req.ValidatorId) > 100 {
		http.Error(w, "validatorId must contain 1-100 ids", http.StatusBadRequest)
		return
	}
	// dedupe and sort
	seen := make(map[int]struct{})
	ids := make([]int, 0, len(req.ValidatorId))
	for _, id := range req.ValidatorId {
		if id < 1 || id > 1000000000 { // allow large ids but basic sanity
			http.Error(w, "validator ids out of range", http.StatusBadRequest)
			return
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	sort.Ints(ids)

	// check cache first and collect missing ids for a single batch call
	missing := make([]int, 0, len(ids))
	responses := make([]ValidatorResponse, 0, len(ids))
	successMap := make(map[int]*ValidatorResponse)
	errsMap := make(map[int]string)

	for _, id := range ids {
		cacheKey := "validator:" + fmt.Sprintf("%d", id)
		if v, ok := memCache.Get(cacheKey); ok {
			if vr, ok := v.(ValidatorResponse); ok {
				responses = append(responses, vr)
				successMap[id] = &vr
				continue
			}
		}
		missing = append(missing, id)
	}

	var firstErr error
	// If there are missing ids, fetch them in a single batch (handler guarantees <=100 ids)
	if len(missing) > 0 {
		overviews, err := bcClient.BatchFetchOverview(missing)
		if err != nil {
			// mark all missing ids as errors
			for _, id := range missing {
				errsMap[id] = err.Error()
				if firstErr == nil {
					firstErr = err
				}
			}
		} else {
			for _, id := range missing {
				if ov, ok := overviews[id]; ok {
					resp := ValidatorResponse{
						Overview: ValidatorOverview{
							ID:                        id,
							Slashed:                   ov.Slashed,
							Status:                    ov.Status,
							WithdrawalCredentialsType: ov.WithdrawalCredentialsType,
							WithdrawalCredentials:     ov.WithdrawalCredentials,
							ActivationEpoch:           ov.ActivationEpoch,
							ExitEpoch:                 ov.ExitEpoch,
							CurrentBalanceRaw:         ov.CurrentBalanceRaw,
							Online:                    ov.Online,
							FetchedAt:                 time.Now().UTC(),
						},
						Rewards: ValidatorRewards{},
						Performance: ValidatorPerformance{
							BeaconScore: ov.BeaconScore,
						},
					}
					// cache
					memCache.Set("validator:"+fmt.Sprintf("%d", id), resp)
					responses = append(responses, resp)
					successMap[id] = &resp
				} else {
					errsMap[id] = "no data from upstream"
				}
			}
		}
	}

	if len(responses) == 0 {
		log.Printf("all upstream calls failed: %v", firstErr)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}

	// Determine evaluation window for rewards/performance
	// Normalize range input (allow several aliases). Default to all_time.
	evalWindow := "all_time"
	if req.Range != "" {
		r := strings.ToLower(strings.TrimSpace(req.Range))
		switch r {
		case "24h", "7d", "30d", "90d":
			evalWindow = r
		case "all", "all_time", "alltime", "all-time":
			evalWindow = "all_time"
		default:
			http.Error(w, "invalid range; allowed: 24h,7d,30d,90d,all_time", http.StatusBadRequest)
			return
		}
	}

	// Fetch rewards and performance for the successful ids and merge
	idsToFetch := make([]int, 0, len(successMap))
	for id := range successMap {
		idsToFetch = append(idsToFetch, id)
	}
	var aggRewards map[string]string
	var aggPerf map[string]interface{}
	if len(idsToFetch) > 0 {
		// Call aggregate endpoints for totals and performance for the whole set
		var rerr error
		aggRewards, rerr = bcClient.BatchFetchRewards(idsToFetch, evalWindow)
		if rerr != nil {
			log.Printf("rewards fetch error: %v", rerr)
		}
		var perr error
		aggPerf, perr = bcClient.BatchFetchPerformance(idsToFetch, evalWindow)
		if perr != nil {
			log.Printf("performance fetch error: %v", perr)
		}
		// Attach aggregates to response (not per-validator)
		outAggregates := make(map[string]interface{})
		if aggRewards != nil {
			outAggregates["rewards"] = aggRewards
		}
		if aggPerf != nil {
			outAggregates["performance"] = aggPerf
		}
		// update cache entries with available aggregate-derived fields where appropriate
		for _, id := range idsToFetch {
			vr := successMap[id]
			// if aggregate rewards contains total_reward, attach proportional placeholder in per-id (best-effort not implemented)
			memCache.Set("validator:"+fmt.Sprintf("%d", id), *vr)
		}
	}

	// If multiple responses, return array; if single, return object for convenience
	// fill human-readable balances
	for _, vr := range responses {
		// CurrentBalanceRaw stored in cached/overview when available
		// try to parse from bcClient.ParseBigToHuman via the cached overview raw
		// but we have CurrentBalance as uint64 placeholder; instead, if cache had raw, use it
		// If successMap contains the vr, set human string
		if s, ok := successMap[vr.Overview.ID]; ok {
			if s.Overview.CurrentBalanceRaw != "" {
				_, eth := beaconcha.ParseBigToHuman(s.Overview.CurrentBalanceRaw)
				s.Overview.CurrentBalanceHuman = eth
				// also update responses slice
				for i := range responses {
					if responses[i].Overview.ID == s.Overview.ID {
						responses[i].Overview.CurrentBalanceHuman = eth
					}
				}
			}
		}
	}

	out := ValidatorBatchResponse{Successes: make(map[int]ValidatorResponse), Errors: errsMap, Aggregates: nil}
	for _, r := range responses {
		out.Successes[r.Overview.ID] = r
	}
	if len(idsToFetch) > 0 {
		// attach aggregates if we have them
		if aggRewards != nil || aggPerf != nil {
			out.Aggregates = map[string]interface{}{}
			if aggRewards != nil {
				out.Aggregates["rewards"] = aggRewards
			}
			if aggPerf != nil {
				out.Aggregates["performance"] = aggPerf
			}
		}
	}
	writeJSON(w, out)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
