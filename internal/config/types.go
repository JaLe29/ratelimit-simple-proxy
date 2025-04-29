package config

// Local types
type rateLimitConfig struct {
	Destination        string   `yaml:"destination"`
	Requests           int      `yaml:"requests"`
	PerSecond          int      `yaml:"perSecond"`
	IpBlackList        []string `yaml:"ipBlackList"`
	CacheMaxTtlSeconds int      `yaml:"cacheMaxTtlSeconds"`
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
}

type Config struct {
	IPHeader   IPHeaderConfig             `yaml:"ipHeader"`
	RateLimits map[string]RateLimitConfig `yaml:"rateLimits"`
}
