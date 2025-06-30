package config

import "time"

// ServerConfig represents server-specific configuration
type ServerConfig struct {
	ReadTimeout    time.Duration `yaml:"readTimeout"`
	WriteTimeout   time.Duration `yaml:"writeTimeout"`
	IdleTimeout    time.Duration `yaml:"idleTimeout"`
	MaxHeaderBytes int           `yaml:"maxHeaderBytes"`
}

// TransportConfig represents HTTP transport configuration
type TransportConfig struct {
	MaxIdleConns        int           `yaml:"maxIdleConns"`
	MaxIdleConnsPerHost int           `yaml:"maxIdleConnsPerHost"`
	IdleConnTimeout     time.Duration `yaml:"idleConnTimeout"`
	TLSHandshakeTimeout time.Duration `yaml:"tlsHandshakeTimeout"`
	DisableCompression  bool          `yaml:"disableCompression"`
}

// Local types
type rateLimitConfig struct {
	Destination   string      `yaml:"destination"`
	Requests      int         `yaml:"requests"`
	PerSecond     int         `yaml:"perSecond"`
	IPBlackList   []string    `yaml:"ipBlackList"`
	AllowedEmails []string    `yaml:"allowedEmails"`
	Auth          *DomainAuth `yaml:"auth"`
}

// DomainAuth represents authentication configuration for a specific domain
type DomainAuth struct {
	Domain      string `yaml:"domain"`      // Auth domain for this specific domain
	RedirectURL string `yaml:"redirectUrl"` // Redirect URL for this specific domain
}

type config struct {
	IPHeader    IPHeaderConfig             `yaml:"ipHeader"`
	GoogleAuth  *GoogleAuth                `yaml:"googleAuth"`
	RateLimits  map[string]rateLimitConfig `yaml:"rateLimits"`
	IPBlackList []string                   `yaml:"ipBlackList"`
	Server      ServerConfig               `yaml:"server"`
	Transport   TransportConfig            `yaml:"transport"`
}

// Global types
type IPHeaderConfig struct {
	Headers []string `yaml:"headers"`
}

type RateLimitConfig struct {
	Destination   string          `yaml:"destination"`
	Requests      int             `yaml:"requests"`
	PerSecond     int             `yaml:"perSecond"`
	IPBlackList   map[string]bool `yaml:"ipBlackList"`
	AllowedEmails []string        `yaml:"allowedEmails"`
	Auth          *DomainAuth     `yaml:"auth"`
}

type GoogleAuth struct {
	Enabled          bool     `yaml:"enabled"`
	ClientID         string   `yaml:"clientId"`
	ClientSecret     string   `yaml:"clientSecret"`
	RedirectURL      string   `yaml:"redirectUrl"`
	ProtectedDomains []string `yaml:"protectedDomains"`
	AuthDomain       string   `yaml:"authDomain"`
	SharedDomains    []string `yaml:"sharedDomains"` // List of domains that share cookies
}

type DomainGroup struct {
	Domain       string   `yaml:"domain"`       // Main domain for cookie (e.g. jale.cz)
	Subdomains   []string `yaml:"subdomains"`   // List of subdomains that share cookies
	OtherDomains []string `yaml:"otherDomains"` // Other domains that share cookies (e.g. auto.cz)
}

type Config struct {
	IPHeader   IPHeaderConfig             `yaml:"ipHeader"`
	GoogleAuth *GoogleAuth                `yaml:"googleAuth"`
	RateLimits map[string]RateLimitConfig `yaml:"rateLimits"`
	Server     ServerConfig               `yaml:"server"`
	Transport  TransportConfig            `yaml:"transport"`
}
