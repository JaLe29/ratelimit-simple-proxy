package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/JaLe29/ratelimit-simple-proxy/internal/config"
	"github.com/JaLe29/ratelimit-simple-proxy/internal/proxy"
)

func main() {
	// Načtení konfigurace
	config, err := config.LoadConfig(".")
	if err != nil {
		log.Fatalf("Chyba při načítání konfigurace: %v", err)
	}

	// Výpis konfigurace
	fmt.Println("Načtená konfigurace:")
	fmt.Println("IP Headers:", config.IPHeader.Headers)
	fmt.Println("Rate Limity:")
	for i, rl := range config.RateLimits {
		fmt.Printf("  #%d: %s -> %s, %d request(ů) za %d sekund(u)\n",
			i+1, rl.Source, rl.Destination, rl.Requests, rl.PerSecond)
	}

	// ---

	http.HandleFunc("/", proxy.ProxyHandler)
	log.Println("Starting proxy on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
