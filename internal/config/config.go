package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads configuration from YAML file
func LoadConfig(configPath string) (*Config, error) {
	// Default configuration
	config := &config{
		IPHeader: IPHeaderConfig{
			Headers: []string{"X-Forwarded-For", "X-Real-IP"},
		},
		RateLimits:  make(map[string]rateLimitConfig),
		IPBlackList: []string{},
	}

	// Search for configuration file
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

	// If file exists, load it
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("error reading configuration file: %w", err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("error parsing configuration: %w", err)
		}

		fmt.Printf("Loaded configuration from file: %s\n", configFile)
	} else {
		fmt.Println("Configuration file not found, using default values")
	}

	// Validate configuration
	if len(config.IPHeader.Headers) == 0 {
		return nil, fmt.Errorf("no IP header defined")
	}

	// Validate Google Auth
	if config.GoogleAuth != nil && config.GoogleAuth.Enabled {
		if config.GoogleAuth.ClientID == "" {
			return nil, fmt.Errorf("Google Auth is enabled but clientId is missing")
		}
		if config.GoogleAuth.ClientSecret == "" {
			return nil, fmt.Errorf("Google Auth is enabled but clientSecret is missing")
		}
		if config.GoogleAuth.RedirectURL == "" {
			return nil, fmt.Errorf("Google Auth is enabled but redirectUrl is missing")
		}
	}

	for key, rl := range config.RateLimits {
		if rl.Destination == "" {
			return nil, fmt.Errorf("rate limit '%s' is missing destination", key)
		}
		if rl.Requests < -1 {
			return nil, fmt.Errorf("rate limit '%s' has invalid number of requests: %d", key, rl.Requests)
		}
		if rl.PerSecond < -1 {
			return nil, fmt.Errorf("rate limit '%s' has invalid perSecond value: %d", key, rl.PerSecond)
		}

		if rl.Requests == -1 && rl.PerSecond != -1 || rl.Requests != -1 && rl.PerSecond == -1 {
			return nil, fmt.Errorf("rate limit '%s' has invalid requests and perSecond values: %d, %d", key, rl.Requests, rl.PerSecond)
		}

		// Validate allowedEmails for Google Auth
		if config.GoogleAuth != nil && config.GoogleAuth.Enabled && len(rl.AllowedEmails) > 0 {
			if len(rl.AllowedEmails) == 0 {
				return nil, fmt.Errorf("rate limit '%s' has Google Auth enabled but missing allowed emails list (allowedEmails)", key)
			}
		}
	}

	// Debug output
	fmt.Println("Loaded rate limits:")
	for k, rl := range config.RateLimits {
		fmt.Printf("Key: %s, Destination: %s, Requests: %d, PerSecond: %d\n",
			k, rl.Destination, rl.Requests, rl.PerSecond)
		if len(rl.AllowedEmails) > 0 {
			fmt.Printf("  Allowed Emails: %v\n", rl.AllowedEmails)
		}
	}

	// Create global config with better structure
	globalConfig := &Config{
		IPHeader:   config.IPHeader,
		GoogleAuth: config.GoogleAuth,
		RateLimits: make(map[string]RateLimitConfig),
	}

	for key, value := range config.RateLimits {
		rateLimitConfig := RateLimitConfig{
			Destination:   value.Destination,
			Requests:      value.Requests,
			PerSecond:     value.PerSecond,
			IPBlackList:   make(map[string]bool),
			AllowedEmails: value.AllowedEmails,
			Auth:          value.Auth,
		}

		for _, ip := range value.IPBlackList {
			rateLimitConfig.IPBlackList[ip] = true
		}

		// Add the original domain
		globalConfig.RateLimits[key] = rateLimitConfig

		// Auto-generate www variants for domains
		if key != config.GoogleAuth.AuthDomain { // Skip auth domain
			var alternativeDomain string
			if strings.HasPrefix(key, "www.") {
				// If domain starts with www., also add version without www.
				alternativeDomain = strings.TrimPrefix(key, "www.")
			} else if !strings.Contains(key, ":") { // Only add www. for domains without port
				// If domain doesn't start with www., also add www. version
				alternativeDomain = "www." + key
			}

			if alternativeDomain != "" && alternativeDomain != key {
				// Create a copy of the rate limit config for the alternative domain
				alternativeConfig := RateLimitConfig{
					Destination:   value.Destination,
					Requests:      value.Requests,
					PerSecond:     value.PerSecond,
					IPBlackList:   make(map[string]bool),
					AllowedEmails: value.AllowedEmails,
					Auth:          value.Auth,
				}

				for _, ip := range value.IPBlackList {
					alternativeConfig.IPBlackList[ip] = true
				}

				globalConfig.RateLimits[alternativeDomain] = alternativeConfig
				fmt.Printf("Auto-generated domain variant: %s -> %s\n", key, alternativeDomain)
			}
		}
	}

	// Copy global blacklist to all rate limits
	for key, value := range globalConfig.RateLimits {
		for _, valueBl := range config.IPBlackList {
			value.IPBlackList[valueBl] = true
		}
		globalConfig.RateLimits[key] = value
	}

	return globalConfig, nil
}
