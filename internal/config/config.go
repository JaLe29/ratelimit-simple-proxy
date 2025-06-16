package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfig načte konfiguraci z YAML souboru
func LoadConfig(configPath string) (*Config, error) {
	// Výchozí konfigurace
	config := &config{
		IPHeader: IPHeaderConfig{
			Headers: []string{"X-Forwarded-For", "X-Real-IP"},
		},
		RateLimits:  make(map[string]rateLimitConfig),
		IpBlackList: []string{},
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

	// Validace Google Auth
	if config.GoogleAuth != nil && config.GoogleAuth.Enabled {
		if config.GoogleAuth.ClientID == "" {
			return nil, fmt.Errorf("Google Auth je povoleno, ale chybí clientId")
		}
		if config.GoogleAuth.ClientSecret == "" {
			return nil, fmt.Errorf("Google Auth je povoleno, ale chybí clientSecret")
		}
		if config.GoogleAuth.RedirectURL == "" {
			return nil, fmt.Errorf("Google Auth je povoleno, ale chybí redirectUrl")
		}
	}

	for key, rl := range config.RateLimits {
		if rl.Destination == "" {
			return nil, fmt.Errorf("u rate limitu '%s' chybí destination", key)
		}
		if rl.Requests < -1 {
			return nil, fmt.Errorf("u rate limitu '%s' je neplatný počet requestů: %d", key, rl.Requests)
		}
		if rl.PerSecond < -1 {
			return nil, fmt.Errorf("u rate limitu '%s' je neplatná hodnota perSecond: %d", key, rl.PerSecond)
		}

		if rl.Requests == -1 && rl.PerSecond != -1 || rl.Requests != -1 && rl.PerSecond == -1 {
			return nil, fmt.Errorf("u rate limitu '%s' je neplatný počet requestů a perSecond: %d, %d", key, rl.Requests, rl.PerSecond)
		}

		if rl.CacheMaxTtlSeconds < 0 {
			return nil, fmt.Errorf("u rate limitu '%s' je neplatná hodnota cacheMaxTtlSeconds: %d", key, rl.CacheMaxTtlSeconds)
		}

		// Validace allowedEmails pro Google Auth
		if config.GoogleAuth != nil && config.GoogleAuth.Enabled && len(rl.AllowedEmails) > 0 {
			if len(rl.AllowedEmails) == 0 {
				return nil, fmt.Errorf("u rate limitu '%s' je povoleno Google Auth, ale chybí seznam povolených emailů (allowedEmails)", key)
			}
		}
	}

	// Debug výpis
	fmt.Println("Načtené rate limity:")
	for k, rl := range config.RateLimits {
		fmt.Printf("Klíč: %s, Destination: %s, Requests: %d, PerSecond: %d, CacheMaxTtlSeconds: %d\n",
			k, rl.Destination, rl.Requests, rl.PerSecond, rl.CacheMaxTtlSeconds)
		if len(rl.AllowedEmails) > 0 {
			fmt.Printf("  Allowed Emails: %v\n", rl.AllowedEmails)
		}
	}

	// create global config with better structure
	globalConfig := &Config{
		IPHeader:   config.IPHeader,
		GoogleAuth: config.GoogleAuth,
		RateLimits: make(map[string]RateLimitConfig),
	}

	for key, value := range config.RateLimits {
		globalConfig.RateLimits[key] = RateLimitConfig{
			Destination:        value.Destination,
			Requests:           value.Requests,
			PerSecond:          value.PerSecond,
			IpBlackList:        make(map[string]bool),
			CacheMaxTtlSeconds: value.CacheMaxTtlSeconds,
			AllowedEmails:      value.AllowedEmails,
		}

		for _, ip := range value.IpBlackList {
			globalConfig.RateLimits[key].IpBlackList[ip] = true
		}
	}

	// copy global blacklist to all rate limits
	for _, value := range globalConfig.RateLimits {
		for _, valueBl := range config.IpBlackList {
			value.IpBlackList[valueBl] = true
		}
	}

	return globalConfig, nil
}
