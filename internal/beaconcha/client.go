// Package beaconcha provides a client for the Beaconcha v2 API.
package beaconcha

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Marketen/validator-dashboard-beaconcha/internal/models"
	"github.com/Marketen/validator-dashboard-beaconcha/internal/ratelimiter"
)

// Client is the Beaconcha API client with built-in rate limiting.
type Client struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	rateLimiter *ratelimiter.GlobalRateLimiter
}

// NewClient creates a new Beaconcha API client.
func NewClient(baseURL, apiKey string, rateLimiter *ratelimiter.GlobalRateLimiter, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		rateLimiter: rateLimiter,
	}
}

// GetValidators fetches validator overview data for the given indices.
// Uses POST /api/v2/ethereum/validators with cursor-based pagination.
func (c *Client) GetValidators(ctx context.Context, chain string, validatorIds []int) ([]models.BeaconchainValidatorData, error) {
	if len(validatorIds) == 0 {
		return nil, nil
	}

	var allData []models.BeaconchainValidatorData
	cursor := ""

	for {
		reqBody := models.BeaconchainValidatorsRequest{
			Chain: chain,
			Validator: models.BeaconchainValidatorSelector{
				ValidatorIdentifiers: validatorIds,
			},
			PageSize: 10, // Max allowed by Beaconcha API
			Cursor:   cursor,
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}

		url := fmt.Sprintf("%s/api/v2/ethereum/validators", c.baseURL)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		c.addHeaders(req)
		req.Header.Set("Content-Type", "application/json")

		slog.Debug("beaconcha request", "method", "POST", "endpoint", "validators", "cursor", cursor)

		resp, body, err := c.doRequestWithRetry(ctx, req, bodyBytes, 3)
		if err != nil {
			return nil, fmt.Errorf("fetch validators: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			slog.Error("beaconcha error response", "status", resp.StatusCode, "body", string(body))
			return nil, fmt.Errorf("beaconcha returned status %d: %s", resp.StatusCode, string(body))
		}

		var response models.BeaconchainValidatorsResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}

		allData = append(allData, response.Data...)

		// Check if there are more pages
		if response.Paging == nil || response.Paging.NextCursor == "" {
			break
		}
		cursor = response.Paging.NextCursor
	}

	return allData, nil
}

// GetRewardsAggregate fetches aggregated rewards for validators.
// Uses POST /api/v2/ethereum/validators/rewards-aggregate
func (c *Client) GetRewardsAggregate(ctx context.Context, chain string, validatorIds []int, evalRange string) (*models.BeaconchainRewardsAggregateResponse, error) {
	if len(validatorIds) == 0 {
		return nil, nil
	}

	reqBody := models.BeaconchainRewardsAggregateRequest{
		Chain: chain,
		Validator: models.BeaconchainValidatorSelector{
			ValidatorIdentifiers: validatorIds,
		},
		Range: models.BeaconchainTimeRangeSelector{
			EvaluationWindow: evalRange,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v2/ethereum/validators/rewards-aggregate", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.addHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	slog.Debug("beaconcha request", "method", "POST", "endpoint", "rewards-aggregate")

	resp, body, err := c.doRequestWithRetry(ctx, req, bodyBytes, 3)
	if err != nil {
		return nil, fmt.Errorf("fetch rewards: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("beaconcha error response", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("beaconcha returned status %d: %s", resp.StatusCode, string(body))
	}

	var response models.BeaconchainRewardsAggregateResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &response, nil
}

// GetPerformanceAggregate fetches aggregated performance metrics for validators.
// Uses POST /api/v2/ethereum/validators/performance-aggregate
func (c *Client) GetPerformanceAggregate(ctx context.Context, chain string, validatorIds []int, evalRange string) (*models.BeaconchainPerformanceAggregateResponse, error) {
	if len(validatorIds) == 0 {
		return nil, nil
	}

	reqBody := models.BeaconchainPerformanceAggregateRequest{
		Chain: chain,
		Validator: models.BeaconchainValidatorSelector{
			ValidatorIdentifiers: validatorIds,
		},
		Range: models.BeaconchainTimeRangeSelector{
			EvaluationWindow: evalRange,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v2/ethereum/validators/performance-aggregate", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.addHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	slog.Debug("beaconcha request", "method", "POST", "endpoint", "performance-aggregate")

	resp, body, err := c.doRequestWithRetry(ctx, req, bodyBytes, 3)
	if err != nil {
		return nil, fmt.Errorf("fetch performance: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("beaconcha error response", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("beaconcha returned status %d: %s", resp.StatusCode, string(body))
	}

	var response models.BeaconchainPerformanceAggregateResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &response, nil
}

// addHeaders adds required headers to the request.
func (c *Client) addHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}

// doRequestWithRetry performs an HTTP request with retry logic for rate limit errors.
// It handles 429 responses by waiting for the reset duration and retrying.
func (c *Client) doRequestWithRetry(ctx context.Context, req *http.Request, bodyBytes []byte, maxRetries int) (*http.Response, []byte, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Wait for rate limiter before each attempt
		if err := c.rateLimiter.WaitAdaptive(ctx); err != nil {
			return nil, nil, fmt.Errorf("rate limiter: %w", err)
		}

		// Clone the request for retry (body needs to be reset)
		reqClone := req.Clone(ctx)
		if bodyBytes != nil {
			reqClone.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := c.httpClient.Do(reqClone)
		if err != nil {
			lastErr = fmt.Errorf("http request: %w", err)
			continue
		}

		// Update rate limiter with response headers
		c.rateLimiter.UpdateFromHeaders(ratelimiter.ParseRateLimitHeaders(resp))

		// Handle rate limit (429)
		if resp.StatusCode == http.StatusTooManyRequests {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			// Get reset time from headers, default to exponential backoff
			info := ratelimiter.ParseRateLimitHeaders(resp)
			waitTime := time.Duration(2<<attempt) * time.Second // 2, 4, 8, 16...
			if info != nil && info.Reset > 0 {
				waitTime = info.Reset + 100*time.Millisecond // Add small buffer
			}

			slog.Warn("rate limited by beaconcha, waiting before retry",
				"attempt", attempt+1,
				"maxRetries", maxRetries,
				"waitTime", waitTime,
				"status", resp.StatusCode)

			select {
			case <-time.After(waitTime):
				lastErr = fmt.Errorf("rate limited (429): %s", string(body))
				continue
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
		}

		// Read body for successful or other error responses
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("read response: %w", err)
		}

		return resp, body, nil
	}

	return nil, nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
