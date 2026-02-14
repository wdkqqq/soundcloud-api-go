package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"soundcloud-api/internal/config"
	"soundcloud-api/internal/middleware"
	"soundcloud-api/internal/scclient"
	"soundcloud-api/internal/utils"
	"soundcloud-api/pkg/types"
)

type Handlers struct {
	Cfg         *config.Config
	ScClient    *scclient.SoundCloudClient
	RateLimiter *middleware.RateLimiter
	Logger      *log.Logger
}

func New(cfg *config.Config, scClient *scclient.SoundCloudClient, rateLimiter *middleware.RateLimiter) *Handlers {
	logger := initLogger(cfg.LogFile)
	return &Handlers{
		Cfg:         cfg,
		ScClient:    scClient,
		RateLimiter: rateLimiter,
		Logger:      logger,
	}
}

func initLogger(logFile string) *log.Logger {
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("can't open log file: %v", err)
	}
	mw := io.MultiWriter(os.Stdout, f)
	return log.New(mw, "", log.LstdFlags|log.Lshortfile)
}

func (h *Handlers) logDebug(format string, v ...interface{}) {
	if h.Cfg.Debug {
		h.Logger.Printf("[DEBUG] "+format, v...)
	}
}

func (h *Handlers) logInfo(format string, v ...interface{}) {
	h.Logger.Printf("[INFO] "+format, v...)
}

func (h *Handlers) logError(format string, v ...interface{}) {
	h.Logger.Printf("[ERROR] "+format, v...)
}

func (h *Handlers) logRequest(r *http.Request, trackURL string) {
	if h.Cfg.Debug {
		h.logDebug("Request: %s %s from %s", r.Method, r.URL.Path, utils.GetClientID(r))
		h.logDebug("Track URL: %s", trackURL)
		h.logDebug("Headers: %v", r.Header)
	} else {
		h.logInfo("Processing %s request for track from %s", r.Method, utils.GetClientID(r))
	}
}

func (h *Handlers) logResponse(result map[string]interface{}) {
	if h.Cfg.Debug {
		redacted := utils.DeepCopyMap(result)
		if _, ok := redacted["stream_url"]; ok {
			redacted["stream_url"] = "[REDACTED]"
		}
		b, _ := json.MarshalIndent(redacted, "", "  ")
		h.logDebug("Response: %s", string(b))
	} else {
		streamURL, hasStream := result["stream_url"]
		errorMsg, hasError := result["error"]

		if hasStream && streamURL != nil && (!hasError || errorMsg == nil) {
			h.logInfo("Success: stream URL obtained")
		} else {
			errorCode, _ := result["error_code"].(string)
			h.logInfo("Failed: %s", errorCode)
		}
	}
}

func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.Cfg.RequestTimeout)
	defer cancel()

	h.logDebug("Health check request from %s", utils.GetClientID(r))

	valid, errMsg := h.ScClient.ValidateToken(ctx)
	status := "healthy"
	statusCode := http.StatusOK
	if !valid {
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
		h.logError("Token validation failed: %s", errMsg)
	} else {
		h.logDebug("Token validation successful")
	}

	health := &types.HealthResponse{
		Status:    status,
		Service:   "soundcloud-api",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0.0",
	}

	if !valid {
		health.TokenError = errMsg
	}

	utils.WriteJSON(w, statusCode, health)
	h.logDebug("Health check response: %s", status)
}

func (h *Handlers) PostStreamHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		h.logDebug("Invalid content type: %s", r.Header.Get("Content-Type"))
		utils.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":      "Content-Type must be application/json",
			"error_code": "INVALID_CONTENT_TYPE",
		})
		return
	}

	var sr types.StreamRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&sr); err != nil {
		h.logDebug("JSON decode error: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":      "Invalid JSON body",
			"error_code": "INVALID_JSON",
		})
		return
	}

	h.processStreamRequest(w, r, sr.TrackURL)
}

func (h *Handlers) GetStreamHandler(w http.ResponseWriter, r *http.Request) {
	trackURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if trackURL == "" {
		h.logDebug("Missing URL parameter")
		utils.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":      "Missing 'url' parameter",
			"error_code": "MISSING_URL_PARAM",
		})
		return
	}

	h.processStreamRequest(w, r, trackURL)
}

func (h *Handlers) processStreamRequest(w http.ResponseWriter, r *http.Request, trackURL string) {
	trackURL = strings.TrimSpace(trackURL)
	isValid, errMsg := utils.ValidateSoundCloudURL(trackURL, h.Cfg.MaxTrackURLLen)
	if !isValid {
		h.logDebug("URL validation failed: %s", errMsg)
		utils.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":      errMsg,
			"error_code": "INVALID_URL",
		})
		return
	}

	h.logRequest(r, trackURL)

	ctx, cancel := context.WithTimeout(r.Context(), h.Cfg.RequestTimeout)
	defer cancel()

	result, err := h.ScClient.GetStreamURL(ctx, trackURL)
	if err != nil {
		h.logError("Unexpected error getting stream URL: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error":      "Internal server error",
			"error_code": "INTERNAL_ERROR",
		})
		return
	}

	h.logResponse(result)

	if result["stream_url"] != nil && result["error"] == nil {
		utils.WriteJSON(w, http.StatusOK, result)
		return
	}

	utils.WriteJSON(w, http.StatusBadRequest, result)
}

func (h *Handlers) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	h.logDebug("Not found: %s %s", r.Method, r.URL.Path)
	utils.WriteJSON(w, http.StatusNotFound, map[string]interface{}{
		"error":      "Endpoint not found",
		"error_code": "NOT_FOUND",
	})
}

func (h *Handlers) GetRateLimitMiddleware() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.RateLimitMiddleware(h.RateLimiter, next)
	}
}
