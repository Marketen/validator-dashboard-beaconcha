package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Marketen/validator-dashboard-beaconcha/internal/config"
	"github.com/Marketen/validator-dashboard-beaconcha/internal/models"
)

func TestValidateValidatorRequest_Valid(t *testing.T) {
	h := &Handler{
		config: &config.Config{MaxValidatorIDs: 100},
	}

	tests := []struct {
		name    string
		request models.ValidatorRequest
	}{
		{
			name:    "single validator",
			request: models.ValidatorRequest{ValidatorIds: []int{1}, Chain: "mainnet"},
		},
		{
			name:    "multiple validators",
			request: models.ValidatorRequest{ValidatorIds: []int{1, 2, 3, 4, 5}, Chain: "mainnet"},
		},
		{
			name:    "max validators",
			request: models.ValidatorRequest{ValidatorIds: makeRange(1, 100), Chain: "mainnet"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := h.validateValidatorRequest(tt.request); err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateValidatorRequest_Invalid(t *testing.T) {
	h := &Handler{
		config: &config.Config{MaxValidatorIDs: 100},
	}

	tests := []struct {
		name    string
		request models.ValidatorRequest
		errMsg  string
	}{
		{
			name:    "empty array",
			request: models.ValidatorRequest{ValidatorIds: []int{}, Chain: "mainnet"},
			errMsg:  "at least 1",
		},
		{
			name:    "too many validators",
			request: models.ValidatorRequest{ValidatorIds: makeRange(1, 101), Chain: "mainnet"},
			errMsg:  "at most 100",
		},
		{
			name:    "duplicate validators",
			request: models.ValidatorRequest{ValidatorIds: []int{1, 2, 1}, Chain: "mainnet"},
			errMsg:  "unique",
		},
		{
			name:    "negative validator ID",
			request: models.ValidatorRequest{ValidatorIds: []int{1, -1}, Chain: "mainnet"},
			errMsg:  "non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := h.validateValidatorRequest(tt.request)
			if err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !containsString(err.Error(), tt.errMsg) {
				t.Errorf("error should contain '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestHandler_Health(t *testing.T) {
	h := &Handler{
		config: &config.Config{
			MaxValidatorIDs:     100,
			IPRateLimitRequests: 60,
			IPRateLimitWindow:   60,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", response["status"])
	}
}

func TestHandler_Validator_InvalidJSON(t *testing.T) {
	h := &Handler{
		config: &config.Config{
			MaxValidatorIDs:     100,
			IPRateLimitRequests: 60,
			IPRateLimitWindow:   60,
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/validator", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	h.handleValidator(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response models.APIError
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Error != "invalid_request" {
		t.Errorf("expected error 'invalid_request', got '%s'", response.Error)
	}
}

func TestHandler_Validator_ValidationError(t *testing.T) {
	h := &Handler{
		config: &config.Config{
			MaxValidatorIDs:     100,
			IPRateLimitRequests: 60,
			IPRateLimitWindow:   60,
		},
	}

	body := `{"validatorIds": []}`
	req := httptest.NewRequest(http.MethodPost, "/validator", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.handleValidator(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response models.APIError
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Error != "validation_error" {
		t.Errorf("expected error 'validation_error', got '%s'", response.Error)
	}
}

func TestGetClientIP(t *testing.T) {
	h := &Handler{}

	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xRealIP    string
		expectedIP string
	}{
		{
			name:       "remote addr only",
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "x-forwarded-for single",
			remoteAddr: "10.0.0.1:12345",
			xff:        "203.0.113.195",
			expectedIP: "203.0.113.195",
		},
		{
			name:       "x-forwarded-for multiple",
			remoteAddr: "10.0.0.1:12345",
			xff:        "203.0.113.195, 70.41.3.18, 150.172.238.178",
			expectedIP: "203.0.113.195",
		},
		{
			name:       "x-real-ip",
			remoteAddr: "10.0.0.1:12345",
			xRealIP:    "203.0.113.100",
			expectedIP: "203.0.113.100",
		},
		{
			name:       "xff takes precedence over x-real-ip",
			remoteAddr: "10.0.0.1:12345",
			xff:        "203.0.113.195",
			xRealIP:    "203.0.113.100",
			expectedIP: "203.0.113.195",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := h.getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("expected IP '%s', got '%s'", tt.expectedIP, ip)
			}
		})
	}
}

func makeRange(start, end int) []int {
	result := make([]int, end-start+1)
	for i := range result {
		result[i] = start + i
	}
	return result
}

func containsString(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
