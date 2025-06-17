package config

import (
	"fmt"
	"os"

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
		IpBlackList: []string{},
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

		if rl.CacheMaxTtlSeconds < 0 {
			return nil, fmt.Errorf("rate limit '%s' has invalid cacheMaxTtlSeconds value: %d", key, rl.CacheMaxTtlSeconds)
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
		fmt.Printf("Key: %s, Destination: %s, Requests: %d, PerSecond: %d, CacheMaxTtlSeconds: %d\n",
			k, rl.Destination, rl.Requests, rl.PerSecond, rl.CacheMaxTtlSeconds)
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

	// Copy global blacklist to all rate limits
	for _, value := range globalConfig.RateLimits {
		for _, valueBl := range config.IpBlackList {
			value.IpBlackList[valueBl] = true
		}
	}

	return globalConfig, nil
}
