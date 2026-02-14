package middleware

import (
	"net/http"
	"sync"
	"time"

	"soundcloud-api/internal/utils"
	"soundcloud-api/pkg/types"
)

type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*types.RateInfo
	max      int
	window   time.Duration
	cleanup  time.Duration
	shutdown chan struct{}
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:  make(map[string]*types.RateInfo),
		max:      max,
		window:   window,
		cleanup:  window * 2,
		shutdown: make(chan struct{}),
	}
	go rl.gc()
	return rl
}

func (r *RateLimiter) gc() {
	t := time.NewTicker(r.cleanup)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			r.mu.Lock()
			now := time.Now()
			for k, v := range r.clients {
				if now.After(v.ResetTime) {
					delete(r.clients, k)
				}
			}
			r.mu.Unlock()
		case <-r.shutdown:
			return
		}
	}
}

func (r *RateLimiter) Stop() {
	close(r.shutdown)
}

func (r *RateLimiter) IsRateLimited(clientID string) (bool, *types.RateLimitResponse) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if info, ok := r.clients[clientID]; ok {
		if now.After(info.ResetTime) {
			info.Count = 1
			info.ResetTime = now.Add(r.window)
			return false, nil
		}
		if info.Count >= r.max {
			return true, &types.RateLimitResponse{
				Error: "Rate limit exceeded",
				Details: map[string]interface{}{
					"limit":          r.max,
					"window_seconds": int(r.window.Seconds()),
					"reset_time":     info.ResetTime.Format(time.RFC3339),
				},
			}
		}
		info.Count++
		return false, nil
	}

	r.clients[clientID] = &types.RateInfo{
		Count:     1,
		ResetTime: now.Add(r.window),
	}
	return false, nil
}

func RateLimitMiddleware(rl *RateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID := utils.GetClientID(r)
		limited, details := rl.IsRateLimited(clientID)
		if limited {
			utils.WriteJSON(w, http.StatusTooManyRequests, details)
			return
		}
		next(w, r)
	}
}
