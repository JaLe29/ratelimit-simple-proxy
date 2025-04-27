package config

// Local types
type rateLimitConfig struct {
	Destination  string   `yaml:"destination"`
	Requests     int      `yaml:"requests"`
	PerSecond    int      `yaml:"perSecond"`
	IPPermaBlock []string `yaml:"ipPermaBlock"`
}

type config struct {
	IPHeader   IPHeaderConfig             `yaml:"ipHeader"`
	RateLimits map[string]rateLimitConfig `yaml:"rateLimits"`
}

// Global types
type IPHeaderConfig struct {
	Headers []string `yaml:"headers"`
}

type RateLimitConfig struct {
	Destination  string          `yaml:"destination"`
	Requests     int             `yaml:"requests"`
	PerSecond    int             `yaml:"perSecond"`
	IPPermaBlock map[string]bool `yaml:"ipPermaBlock"`
}

type Config struct {
	IPHeader   IPHeaderConfig             `yaml:"ipHeader"`
	RateLimits map[string]RateLimitConfig `yaml:"rateLimits"`
}
