package config

// Local types
type rateLimitConfig struct {
	Destination        string   `yaml:"destination"`
	Requests           int      `yaml:"requests"`
	PerSecond          int      `yaml:"perSecond"`
	IPBlackList        []string `yaml:"ipBlackList"`
	CacheMaxTTLSeconds int      `yaml:"cacheMaxTtlSeconds"`
	AllowedEmails      []string `yaml:"allowedEmails"`
}

type config struct {
	IPHeader           IPHeaderConfig             `yaml:"ipHeader"`
	GoogleAuth         *GoogleAuth                `yaml:"googleAuth"`
	RateLimits         map[string]rateLimitConfig `yaml:"rateLimits"`
	IPBlackList        []string                   `yaml:"ipBlackList"`
	CacheMaxTTLSeconds int                        `yaml:"cacheMaxTtlSeconds"`
}

// Global types
type IPHeaderConfig struct {
	Headers []string `yaml:"headers"`
}

type RateLimitConfig struct {
	Destination        string          `yaml:"destination"`
	Requests           int             `yaml:"requests"`
	PerSecond          int             `yaml:"perSecond"`
	IPBlackList        map[string]bool `yaml:"ipBlackList"`
	CacheMaxTTLSeconds int             `yaml:"cacheMaxTtlSeconds"`
	AllowedEmails      []string        `yaml:"allowedEmails"`
	InjectControlPanel bool            `yaml:"injectControlPanel"` // Inject control panel into this domain's pages
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
}
