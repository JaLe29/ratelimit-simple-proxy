package config

// Local types
type rateLimitConfig struct {
	Destination        string      `yaml:"destination"`
	Requests           int         `yaml:"requests"`
	PerSecond          int         `yaml:"perSecond"`
	IpBlackList        []string    `yaml:"ipBlackList"`
	CacheMaxTtlSeconds int         `yaml:"cacheMaxTtlSeconds"`
	GoogleAuth         *GoogleAuth `yaml:"googleAuth,omitempty"`
}

type config struct {
	IPHeader           IPHeaderConfig             `yaml:"ipHeader"`
	RateLimits         map[string]rateLimitConfig `yaml:"rateLimits"`
	IpBlackList        []string                   `yaml:"ipBlackList"`
	CacheMaxTtlSeconds int                        `yaml:"cacheMaxTtlSeconds"`
}

// Global types
type IPHeaderConfig struct {
	Headers []string `yaml:"headers"`
}

type RateLimitConfig struct {
	Destination        string          `yaml:"destination"`
	Requests           int             `yaml:"requests"`
	PerSecond          int             `yaml:"perSecond"`
	IpBlackList        map[string]bool `yaml:"ipBlackList"`
	CacheMaxTtlSeconds int             `yaml:"cacheMaxTtlSeconds"`
	GoogleAuth         *GoogleAuth     `yaml:"googleAuth,omitempty"`
}

type GoogleAuth struct {
	Enabled       bool     `yaml:"enabled"`
	ClientID      string   `yaml:"clientId"`
	ClientSecret  string   `yaml:"clientSecret"`
	RedirectURL   string   `yaml:"redirectUrl"`
	AllowedEmails []string `yaml:"allowedEmails"`
}

type Config struct {
	IPHeader   IPHeaderConfig             `yaml:"ipHeader"`
	RateLimits map[string]RateLimitConfig `yaml:"rateLimits"`
}
