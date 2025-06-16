package main

import (
	"log"
	"net/http"

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

	proxy := proxy.NewProxy(config, metric.NewMetric())

	http.HandleFunc("/rlsp/system/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

	log.Println("Starting proxy on :8080")

	for key, value := range config.RateLimits {
		log.Printf("Rate limit for %s: %v\n", key, value)
	}

	log.Printf("Auth domain: %s\n", config.GoogleAuth.AuthDomain)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
