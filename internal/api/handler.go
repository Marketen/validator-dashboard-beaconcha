// Package api provides HTTP handlers for the validator-dashboard API.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Marketen/validator-dashboard-beaconcha/internal/config"
	"github.com/Marketen/validator-dashboard-beaconcha/internal/models"
	"github.com/Marketen/validator-dashboard-beaconcha/internal/service"
)

// Handler provides HTTP handlers for the API.
type Handler struct {
	validatorService *service.ValidatorService
	config           *config.Config
}

// NewHandler creates a new API handler.
func NewHandler(validatorService *service.ValidatorService, cfg *config.Config) *Handler {
	return &Handler{
		validatorService: validatorService,
		config:           cfg,
	}
}

// Router returns the HTTP router with all routes configured.
func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /health", h.handleHealth)

	// Validator endpoint
	mux.HandleFunc("POST /validator", h.handleValidator)

	// Apply middleware
	handler := h.recoveryMiddleware(mux)
	handler = h.loggingMiddleware(handler)
	handler = h.corsMiddleware(handler)
	handler = h.maxBodySizeMiddleware(handler, 1<<20) // 1 MB max body size

	return handler
}

// handleHealth returns API health status.
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.jsonResponse(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// handleValidator handles POST /validator requests.
func (h *Handler) handleValidator(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req models.ValidatorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body: "+err.Error())
		return
	}

	// Validate request
	if err := h.validateValidatorRequest(req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	// Fetch validator data
	response, err := h.validatorService.GetValidatorData(r.Context(), req.Chain, req.ValidatorIds)
	if err != nil {
		slog.Error("failed to fetch validator data", "error", err)
		h.errorResponse(w, http.StatusInternalServerError, "internal_error", "Failed to fetch validator data")
		return
	}

	h.jsonResponse(w, http.StatusOK, response)
}

// validateValidatorRequest validates the incoming validator request.
func (h *Handler) validateValidatorRequest(req models.ValidatorRequest) error {
	if len(req.ValidatorIds) == 0 {
		return &ValidationError{Field: "validatorIds", Message: "must contain at least 1 validator ID"}
	}

	if len(req.ValidatorIds) > h.config.MaxValidatorIDs {
		return &ValidationError{Field: "validatorIds", Message: "must contain at most 100 validator IDs"}
	}

	// Check for duplicates and validate each ID
	seen := make(map[int]bool)
	for _, id := range req.ValidatorIds {
		if id < 0 {
			return &ValidationError{Field: "validatorIds", Message: "validator IDs must be non-negative integers"}
		}
		if seen[id] {
			return &ValidationError{Field: "validatorIds", Message: "validator IDs must be unique"}
		}
		seen[id] = true
	}

	// Validate chain
	if req.Chain == "" {
		return &ValidationError{Field: "chain", Message: "must be provided and be one of: mainnet, hoodi"}
	}
	if req.Chain != "mainnet" && req.Chain != "hoodi" {
		return &ValidationError{Field: "chain", Message: "must be one of: mainnet, hoodi"}
	}

	return nil
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// jsonResponse writes a JSON response.
func (h *Handler) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

// errorResponse writes an error JSON response.
func (h *Handler) errorResponse(w http.ResponseWriter, status int, errorCode, message string) {
	h.jsonResponse(w, status, models.APIError{
		Error:   errorCode,
		Message: message,
		Code:    status,
	})
}

// Middleware functions

// getClientIP extracts the client IP from the request.
func (h *Handler) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// loggingMiddleware logs incoming requests.
func (h *Handler) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration", time.Since(start).String(),
			"ip", h.getClientIP(r),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// recoveryMiddleware recovers from panics.
func (h *Handler) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered", "error", err)
				h.errorResponse(w, http.StatusInternalServerError, "internal_error", "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers.
func (h *Handler) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// maxBodySizeMiddleware limits the request body size.
func (h *Handler) maxBodySizeMiddleware(next http.Handler, maxBytes int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next.ServeHTTP(w, r)
	})
}
