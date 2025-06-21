package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/config"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/metric"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/proxy"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	config, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	proxy, err := proxy.NewProxy(config, metric.NewMetric())
	if err != nil {
		log.Fatalf("Error creating proxy: %v", err)
	}

	// Create HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: createHandler(config, proxy),
	}

	// Start server in a goroutine
	go func() {
		log.Println("Starting proxy on :8080")
		for key, value := range config.RateLimits {
			log.Printf("Rate limit for %s: %v\n", key, value)
		}
		log.Printf("Auth domain: %s\n", config.GoogleAuth.AuthDomain)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	// Kill will send syscall.SIGTERM signal to the process
	// Kill -2 will send syscall.SIGINT signal to the process
	// Kill -9 will send syscall.SIGKILL signal to the process (can't be caught, so don't need to add it)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 30 seconds to finish
	// the request it is currently handling
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown proxy first
	if err := proxy.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down proxy: %v", err)
	}

	// Then shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited gracefully")
}

func createHandler(config *config.Config, proxy *proxy.Proxy) http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/rlsp/system/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Main proxy handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Host == config.GoogleAuth.AuthDomain {
			if r.URL.Path == "/auth/callback" {
				proxy.ProxyHandler(w, r)
				return
			}
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		proxy.ProxyHandler(w, r)
	})

	return mux
}
