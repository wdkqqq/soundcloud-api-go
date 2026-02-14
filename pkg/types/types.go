package types

import "time"

type StreamRequest struct {
	TrackURL string `json:"track_url"`
}

type RateInfo struct {
	Count     int       `json:"count"`
	ResetTime time.Time `json:"reset_time"`
}

type HealthResponse struct {
	Status     string `json:"status"`
	Service    string `json:"service"`
	Timestamp  string `json:"timestamp"`
	Version    string `json:"version"`
	TokenError string `json:"token_error,omitempty"`
}

type StreamResponse struct {
	StreamURL interface{}            `json:"stream_url"`
	Error     interface{}            `json:"error"`
	ErrorCode interface{}            `json:"error_code"`
	TrackInfo map[string]interface{} `json:"track_info"`
	CacheInfo map[string]interface{} `json:"cache_info"`
}

type RateLimitResponse struct {
	Error   string                 `json:"error"`
	Details map[string]interface{} `json:"details"`
}
