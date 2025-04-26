package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// IPHeaderConfig představuje konfiguraci pro čtení IP adresy z headerů
type IPHeaderConfig struct {
	Headers []string `yaml:"headers"` // může být string nebo pole stringů
}

// RateLimitConfig definuje rate limity pro source->destination
type RateLimitConfig struct {
	Destination string `yaml:"destination"`
	Requests    int    `yaml:"requests"`  // počet požadavků
	PerSecond   int    `yaml:"perSecond"` // časové okno v sekundách
}

// Config reprezentuje celou konfiguraci aplikace
type Config struct {
	IPHeader   IPHeaderConfig             `yaml:"ipHeader"`
	RateLimits map[string]RateLimitConfig `yaml:"rateLimits"`
}

// LoadConfig načte konfiguraci z YAML souboru
func LoadConfig(configPath string) (*Config, error) {
	// Výchozí konfigurace
	config := &Config{
		IPHeader: IPHeaderConfig{
			Headers: []string{"X-Forwarded-For", "X-Real-IP"},
		},
		RateLimits: make(map[string]RateLimitConfig),
	}

	// Hledání konfiguračního souboru
	var configFile string
	for _, path := range []string{configPath, ".", "./config"} {
		file := path + "/config.yaml"
		if _, err := os.Stat(file); err == nil {
			configFile = file
			break
		}
		file = path + "/config.yml"
		if _, err := os.Stat(file); err == nil {
			configFile = file
			break
		}
	}

	// Pokud soubor existuje, načti ho
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("chyba při čtení konfiguračního souboru: %w", err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("chyba při parsování konfigurace: %w", err)
		}

		fmt.Printf("Načtená konfigurace ze souboru: %s\n", configFile)
	} else {
		fmt.Println("Konfigurační soubor nenalezen, používám výchozí hodnoty")
	}

	// Validace konfigurace
	if len(config.IPHeader.Headers) == 0 {
		return nil, fmt.Errorf("není definován žádný header pro IP")
	}

	// Validace rate limitů v mapě
	for key, rl := range config.RateLimits {
		if rl.Destination == "" {
			return nil, fmt.Errorf("u rate limitu '%s' chybí destination", key)
		}
		if rl.Requests <= 0 {
			return nil, fmt.Errorf("u rate limitu '%s' je neplatný počet requestů: %d", key, rl.Requests)
		}
		if rl.PerSecond <= 0 {
			return nil, fmt.Errorf("u rate limitu '%s' je neplatná hodnota perSecond: %d", key, rl.PerSecond)
		}
	}

	// Debug výpis
	fmt.Println("Načtené rate limity:")
	for k, rl := range config.RateLimits {
		fmt.Printf("Klíč: %s, Destination: %s, Requests: %d, PerSecond: %d\n",
			k, rl.Destination, rl.Requests, rl.PerSecond)
	}

	return config, nil
}
