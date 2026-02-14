package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"soundcloud-api/internal/config"
	"soundcloud-api/internal/handlers"
	"soundcloud-api/internal/middleware"
	"soundcloud-api/internal/scclient"
)

func main() {
	if err := config.LoadEnvFile(".env"); err != nil {
		log.Printf("warning: failed to load .env: %v", err)
	}

	cfg := config.Load()

	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)
	defer rateLimiter.Stop()

	scClient := scclient.New(cfg.AuthToken, cfg.ClientID, cfg.RequestTimeout)
	handler := handlers.New(cfg, scClient, rateLimiter)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	valid, errMsg := scClient.ValidateToken(ctx)
	if valid {
		handler.Logger.Println("SoundCloud auth_token is valid!")
	} else {
		handler.Logger.Printf("SoundCloud auth_token is invalid: %s", errMsg)
	}

	mux := http.NewServeMux()
	rateLimitMiddleware := handler.GetRateLimitMiddleware()

	mux.HandleFunc("/health", rateLimitMiddleware(handler.HealthHandler))
	mux.HandleFunc("/soundcloud/stream-url", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			rateLimitMiddleware(handler.PostStreamHandler)(w, r)
		case http.MethodGet:
			rateLimitMiddleware(handler.GetStreamHandler)(w, r)
		default:
			handler.NotFoundHandler(w, r)
		}
	})
	mux.HandleFunc("/", handler.NotFoundHandler)

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	requestedPort := cfg.Port
	listener, err := net.Listen("tcp", ":"+requestedPort)
	if err != nil {
		if !errors.Is(err, syscall.EADDRINUSE) {
			handler.Logger.Fatalf("server failed to bind :%s: %v", requestedPort, err)
		}

		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			handler.Logger.Fatalf("server failed to bind fallback port: %v", err)
		}
	}

	actualPort := requestedPort
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		actualPort = strconv.Itoa(tcpAddr.Port)
	}

	if actualPort != requestedPort {
		handler.Logger.Printf("Port :%s is busy, using :%s", requestedPort, actualPort)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		handler.Logger.Printf("Server starting on :%s", actualPort)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			handler.Logger.Fatalf("server failed: %v", err)
		}
	}()

	<-ctx.Done()
	handler.Logger.Println("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		handler.Logger.Printf("HTTP server shutdown error: %v", err)
	} else {
		handler.Logger.Println("Server shutdown completed")
	}
}
