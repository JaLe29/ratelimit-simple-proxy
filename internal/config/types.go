package config

// Local types
type rateLimitConfig struct {
	Destination        string   `yaml:"destination"`
	Requests           int      `yaml:"requests"`
	PerSecond          int      `yaml:"perSecond"`
	IpBlackList        []string `yaml:"ipBlackList"`
	CacheMaxTtlSeconds int      `yaml:"cacheMaxTtlSeconds"`
	AllowedEmails      []string `yaml:"allowedEmails"`
}

type config struct {
	IPHeader           IPHeaderConfig             `yaml:"ipHeader"`
	GoogleAuth         *GoogleAuth                `yaml:"googleAuth"`
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
	AllowedEmails      []string        `yaml:"allowedEmails"`
}

type GoogleAuth struct {
	Enabled          bool     `yaml:"enabled"`
	ClientID         string   `yaml:"clientId"`
	ClientSecret     string   `yaml:"clientSecret"`
	RedirectURL      string   `yaml:"redirectUrl"`
	ProtectedDomains []string `yaml:"protectedDomains"`
	AuthDomain       string   `yaml:"authDomain"`
	SharedDomains    []string `yaml:"sharedDomains"` // Seznam domén, které sdílejí cookie
}

type DomainGroup struct {
	Domain       string   `yaml:"domain"`       // Hlavní doména pro cookie (např. jale.cz)
	Subdomains   []string `yaml:"subdomains"`   // Seznam subdomén, které sdílejí cookie
	OtherDomains []string `yaml:"otherDomains"` // Jiné domény, které sdílejí cookie (např. auto.cz)
}

type Config struct {
	IPHeader   IPHeaderConfig             `yaml:"ipHeader"`
	GoogleAuth *GoogleAuth                `yaml:"googleAuth"`
	RateLimits map[string]RateLimitConfig `yaml:"rateLimits"`
}
