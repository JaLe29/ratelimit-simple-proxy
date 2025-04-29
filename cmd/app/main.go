package main

import (
	"log"
	"net/http"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/config"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/proxy"
)

func main() {
	config, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	proxy := proxy.NewProxy(config)

	http.HandleFunc("/rlsp/system/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/", proxy.ProxyHandler)

	log.Println("Starting proxy on :8080")

	for key, value := range config.RateLimits {
		log.Printf("Rate limit for %s: %v\n", key, value)
	}

	log.Fatal(http.ListenAndServe(":8080", nil))

}
