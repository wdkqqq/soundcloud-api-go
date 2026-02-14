package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadEnvFile_MissingFileDoesNotFail(t *testing.T) {
	err := LoadEnvFile(filepath.Join(t.TempDir(), ".env"))
	if err != nil {
		t.Fatalf("expected nil error for missing .env, got %v", err)
	}
}

func TestLoadEnvFile_LoadsValuesAndKeepsExistingEnv(t *testing.T) {
	t.Setenv("EXISTING_KEY", "from-env")

	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	content := `
# comment
AUTH_TOKEN=token-123
CLIENT_ID='client-456'
export RATE_LIMIT_REQUESTS=250
EXISTING_KEY=from-file
INVALID_LINE
`

	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp .env: %v", err)
	}

	if err := LoadEnvFile(envPath); err != nil {
		t.Fatalf("LoadEnvFile returned error: %v", err)
	}

	if got := os.Getenv("AUTH_TOKEN"); got != "token-123" {
		t.Fatalf("AUTH_TOKEN = %q, want %q", got, "token-123")
	}

	if got := os.Getenv("CLIENT_ID"); got != "client-456" {
		t.Fatalf("CLIENT_ID = %q, want %q", got, "client-456")
	}

	if got := os.Getenv("RATE_LIMIT_REQUESTS"); got != "250" {
		t.Fatalf("RATE_LIMIT_REQUESTS = %q, want %q", got, "250")
	}

	if got := os.Getenv("EXISTING_KEY"); got != "from-env" {
		t.Fatalf("EXISTING_KEY = %q, want %q", got, "from-env")
	}
}

func TestLoad_UsesEnvValues(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "auth-from-env")
	t.Setenv("CLIENT_ID", "client-from-env")
	t.Setenv("RATE_LIMIT_REQUESTS", "42")
	t.Setenv("RATE_LIMIT_WINDOW", "2m")
	t.Setenv("REQUEST_TIMEOUT", "15s")
	t.Setenv("MAX_TRACK_URL_LEN", "1024")
	t.Setenv("LOG_FILE", "api.log")
	t.Setenv("PORT", "7001")
	t.Setenv("DEBUG", "true")

	cfg := Load()

	if cfg.AuthToken != "auth-from-env" {
		t.Fatalf("AuthToken = %q, want %q", cfg.AuthToken, "auth-from-env")
	}

	if cfg.ClientID != "client-from-env" {
		t.Fatalf("ClientID = %q, want %q", cfg.ClientID, "client-from-env")
	}

	if cfg.RateLimitRequests != 42 {
		t.Fatalf("RateLimitRequests = %d, want %d", cfg.RateLimitRequests, 42)
	}

	if cfg.RateLimitWindow != 2*time.Minute {
		t.Fatalf("RateLimitWindow = %s, want %s", cfg.RateLimitWindow, 2*time.Minute)
	}

	if cfg.RequestTimeout != 15*time.Second {
		t.Fatalf("RequestTimeout = %s, want %s", cfg.RequestTimeout, 15*time.Second)
	}

	if cfg.MaxTrackURLLen != 1024 {
		t.Fatalf("MaxTrackURLLen = %d, want %d", cfg.MaxTrackURLLen, 1024)
	}

	if cfg.LogFile != "api.log" {
		t.Fatalf("LogFile = %q, want %q", cfg.LogFile, "api.log")
	}

	if cfg.Port != "7001" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "7001")
	}

	if !cfg.Debug {
		t.Fatal("Debug = false, want true")
	}
}

func TestLoad_UsesDefaultsForInvalidOrEmptyValues(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "")
	t.Setenv("CLIENT_ID", "")
	t.Setenv("RATE_LIMIT_REQUESTS", "invalid")
	t.Setenv("RATE_LIMIT_WINDOW", "invalid")
	t.Setenv("REQUEST_TIMEOUT", "invalid")
	t.Setenv("MAX_TRACK_URL_LEN", "invalid")
	t.Setenv("LOG_FILE", "")
	t.Setenv("PORT", "")
	t.Setenv("DEBUG", "invalid")

	cfg := Load()

	if cfg.AuthToken != "" {
		t.Fatalf("AuthToken = %q, want empty string", cfg.AuthToken)
	}

	if cfg.ClientID != "" {
		t.Fatalf("ClientID = %q, want empty string", cfg.ClientID)
	}

	if cfg.RateLimitRequests != 100 {
		t.Fatalf("RateLimitRequests = %d, want %d", cfg.RateLimitRequests, 100)
	}

	if cfg.RateLimitWindow != 3600*time.Second {
		t.Fatalf("RateLimitWindow = %s, want %s", cfg.RateLimitWindow, 3600*time.Second)
	}

	if cfg.RequestTimeout != 30*time.Second {
		t.Fatalf("RequestTimeout = %s, want %s", cfg.RequestTimeout, 30*time.Second)
	}

	if cfg.MaxTrackURLLen != 500 {
		t.Fatalf("MaxTrackURLLen = %d, want %d", cfg.MaxTrackURLLen, 500)
	}

	if cfg.LogFile != "SC_API.log" {
		t.Fatalf("LogFile = %q, want %q", cfg.LogFile, "SC_API.log")
	}

	if cfg.Port != "5000" {
		t.Fatalf("Port = %q, want %q", cfg.Port, "5000")
	}

	if cfg.Debug {
		t.Fatal("Debug = true, want false")
	}
}
