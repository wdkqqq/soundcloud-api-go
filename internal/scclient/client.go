package scclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"soundcloud-api/internal/utils"
)

type SoundCloudClient struct {
	httpClient *http.Client
	authToken  string
	clientID   string
}

func New(authToken, clientID string, timeout time.Duration) *SoundCloudClient {
	client := &http.Client{
		Timeout: timeout,
	}
	return &SoundCloudClient{
		httpClient: client,
		authToken:  authToken,
		clientID:   clientID,
	}
}

func (s *SoundCloudClient) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", "Go-http-client/1.1")
	req.Header.Set("Accept", "application/json")
	if s.authToken != "" {
		req.Header.Set("Authorization", "OAuth "+s.authToken)
	}
	return s.httpClient.Do(req)
}

func (s *SoundCloudClient) ValidateToken(ctx context.Context) (bool, string) {
	u := "https://api-v2.soundcloud.com/me"
	req, _ := http.NewRequest("GET", u, nil)
	resp, err := s.doRequest(ctx, req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true, ""
	}
	body, _ := io.ReadAll(resp.Body)
	return false, "status code " + strconv.Itoa(resp.StatusCode) + ": " + string(body)
}

func (s *SoundCloudClient) ResolveTrack(ctx context.Context, trackURL string) (map[string]interface{}, error) {
	resolveURL := "https://api-v2.soundcloud.com/resolve"
	req, _ := http.NewRequest("GET", resolveURL, nil)
	q := req.URL.Query()
	q.Set("url", trackURL)
	q.Set("client_id", s.clientID)
	req.URL.RawQuery = q.Encode()

	resp, err := s.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, errors.New("resolve failed: " + strconv.Itoa(resp.StatusCode))
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func (s *SoundCloudClient) GetStreamURL(ctx context.Context, trackURL string) (map[string]interface{}, error) {
	trackInfo, err := s.ResolveTrack(ctx, trackURL)
	if err != nil || trackInfo == nil {
		return map[string]interface{}{
			"error":      "Track not found or unavailable",
			"stream_url": nil,
			"error_code": "TRACK_NOT_FOUND",
		}, nil
	}

	if policyRaw, ok := trackInfo["policy"]; ok {
		if policyStr, ok := policyRaw.(string); ok && strings.ToUpper(policyStr) == "BLOCK" {
			return map[string]interface{}{
				"error":      "Track is blocked in your region",
				"stream_url": nil,
				"error_code": "GEO_BLOCKED",
			}, nil
		}
	}

	media, _ := trackInfo["media"].(map[string]interface{})
	transcodings, _ := media["transcodings"].([]interface{})
	var progressiveURL string
	for _, t := range transcodings {
		if tm, ok := t.(map[string]interface{}); ok {
			format, _ := tm["format"].(map[string]interface{})
			if proto, _ := format["protocol"].(string); proto == "progressive" {
				if u, _ := tm["url"].(string); u != "" {
					progressiveURL = u
					break
				}
			}
		}
	}

	if progressiveURL == "" {
		return map[string]interface{}{
			"error":      "Progressive stream not available for this track",
			"stream_url": nil,
			"error_code": "NO_PROGRESSIVE_STREAM",
		}, nil
	}

	u, err := url.Parse(progressiveURL)
	if err != nil {
		return map[string]interface{}{
			"error":      "Internal error building stream URL",
			"stream_url": nil,
			"error_code": "INTERNAL_ERROR",
		}, nil
	}

	q := u.Query()
	q.Set("client_id", s.clientID)
	u.RawQuery = q.Encode()

	req, _ := http.NewRequest("GET", u.String(), nil)
	resp, err := s.doRequest(ctx, req)
	if err != nil {
		return map[string]interface{}{
			"error":      "Network error: " + err.Error(),
			"stream_url": nil,
			"error_code": "NETWORK_ERROR",
		}, nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return map[string]interface{}{
			"error":      "Stream API error: " + strconv.Itoa(resp.StatusCode),
			"stream_url": nil,
			"error_code": "API_ERROR_" + strconv.Itoa(resp.StatusCode),
		}, nil
	}

	var streamResp map[string]interface{}
	if err := json.Unmarshal(body, &streamResp); err != nil {
		return map[string]interface{}{
			"error":      "Internal server error",
			"stream_url": nil,
			"error_code": "INTERNAL_ERROR",
		}, nil
	}

	finalURL, _ := streamResp["url"].(string)
	if finalURL == "" {
		return map[string]interface{}{
			"error":      "No stream URL in response",
			"stream_url": nil,
			"error_code": "NO_STREAM_URL",
		}, nil
	}

	trackInfoTitle, _ := trackInfo["title"].(string)
	userObj, _ := trackInfo["user"].(map[string]interface{})
	username, _ := userObj["username"].(string)

	return map[string]interface{}{
		"stream_url": finalURL,
		"error":      nil,
		"error_code": nil,
		"track_info": map[string]interface{}{
			"title":         utils.IfString(trackInfoTitle, "Unknown"),
			"artist":        utils.IfString(username, "Unknown"),
			"duration":      trackInfo["duration"],
			"permalink_url": trackInfo["permalink_url"],
			"artwork_url":   trackInfo["artwork_url"],
			"genre":         trackInfo["genre"],
			"release_date":  trackInfo["release_date"],
		},
		"cache_info": map[string]interface{}{
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"ttl_seconds": 3600,
		},
	}, nil
}
