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
// Uses POST /api/v2/ethereum/validators
func (c *Client) GetValidators(ctx context.Context, validatorIds []int) ([]models.BeaconchainValidatorData, error) {
	if len(validatorIds) == 0 {
		return nil, nil
	}

	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	reqBody := models.BeaconchainValidatorsRequest{
		Validator: models.BeaconchainValidatorSelector{
			ValidatorIdentifiers: validatorIds,
		},
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

	slog.Debug("beaconcha request", "method", "POST", "endpoint", "validators")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("beaconcha error response", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("beaconcha returned status %d: %s", resp.StatusCode, string(body))
	}

	var response models.BeaconchainValidatorsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return response.Data, nil
}

// GetRewardsAggregate fetches aggregated rewards for validators.
// Uses POST /api/v2/ethereum/validators/rewards-aggregate
func (c *Client) GetRewardsAggregate(ctx context.Context, validatorIds []int) (*models.BeaconchainRewardsAggregateResponse, error) {
	if len(validatorIds) == 0 {
		return nil, nil
	}

	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	reqBody := models.BeaconchainRewardsAggregateRequest{
		Validator: models.BeaconchainValidatorSelector{
			ValidatorIdentifiers: validatorIds,
		},
		Range: models.BeaconchainTimeRangeSelector{
			EvaluationWindow: "all_time",
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
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
func (c *Client) GetPerformanceAggregate(ctx context.Context, validatorIds []int) (*models.BeaconchainPerformanceAggregateResponse, error) {
	if len(validatorIds) == 0 {
		return nil, nil
	}

	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	reqBody := models.BeaconchainPerformanceAggregateRequest{
		Validator: models.BeaconchainValidatorSelector{
			ValidatorIdentifiers: validatorIds,
		},
		Range: models.BeaconchainTimeRangeSelector{
			EvaluationWindow: "all_time",
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
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
