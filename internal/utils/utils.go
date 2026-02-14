package utils

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
)

func GetClientID(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func ValidateSoundCloudURL(raw string, maxLen int) (bool, string) {
	if strings.TrimSpace(raw) == "" {
		return false, "URL is required"
	}
	if len(raw) > maxLen {
		return false, "URL too long (max " + strconv.Itoa(maxLen) + " characters)"
	}
	if !strings.HasPrefix(raw, "https://soundcloud.com/") {
		return false, "Invalid SoundCloud URL format"
	}

	parts := strings.Split(strings.Trim(raw, "/"), "/")
	if len(parts) < 4 || parts[2] == "" {
		return false, "Invalid SoundCloud track URL format"
	}

	return true, ""
}

func IfString(s string, otherwise string) string {
	if strings.TrimSpace(s) == "" {
		return otherwise
	}
	return s
}

func DeepCopyMap(m map[string]interface{}) map[string]interface{} {
	b, err := json.Marshal(m)
	if err != nil {
		return m
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return m
	}
	return out
}
