package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// IPHeaderConfig představuje konfiguraci pro čtení IP adresy z headerů
type IPHeaderConfig struct {
	Headers []string `mapstructure:"headers"` // může být string nebo pole stringů
}

// RateLimitConfig definuje rate limity pro source->destination
type RateLimitConfig struct {
	Source      string `mapstructure:"source"`
	Destination string `mapstructure:"destination"`
	Requests    int    `mapstructure:"requests"`  // počet požadavků
	PerSecond   int    `mapstructure:"perSecond"` // časové okno v sekundách
}

// Config reprezentuje celou konfiguraci aplikace
type Config struct {
	IPHeader   IPHeaderConfig    `mapstructure:"ipHeader"`
	RateLimits []RateLimitConfig `mapstructure:"rateLimits"`
}

// LoadConfig načte konfiguraci pomocí Viper
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Nastavení výchozích hodnot
	v.SetDefault("ipHeader.headers", []string{"X-Forwarded-For", "X-Real-IP"})

	// Konfigurace Viper
	v.SetConfigName("config") // název souboru bez přípony
	v.SetConfigType("yaml")   // typ konfiguračního souboru
	v.AddConfigPath(configPath)
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	// Automatické načtení proměnných prostředí s prefixem "APP_"
	v.SetEnvPrefix("APP")
	v.AutomaticEnv()

	// Čtení konfigurace
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("chyba při načtení konfigurace: %w", err)
	}

	// Unmarshalling konfigurace do struktury
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("chyba při unmarshallingu konfigurace: %w", err)
	}

	// Validace konfigurace
	if len(config.IPHeader.Headers) == 0 {
		return nil, fmt.Errorf("není definován žádný header pro IP")
	}

	for i, rl := range config.RateLimits {
		if rl.Source == "" {
			return nil, fmt.Errorf("u rate limitu #%d chybí source", i+1)
		}
		if rl.Destination == "" {
			return nil, fmt.Errorf("u rate limitu #%d chybí destination", i+1)
		}
		if rl.Requests <= 0 {
			return nil, fmt.Errorf("u rate limitu #%d je neplatný počet requestů: %d", i+1, rl.Requests)
		}
		if rl.PerSecond <= 0 {
			return nil, fmt.Errorf("u rate limitu #%d je neplatná hodnota perSecond: %d", i+1, rl.PerSecond)
		}
	}

	return &config, nil
}
