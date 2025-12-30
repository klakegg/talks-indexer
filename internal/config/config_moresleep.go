package config

// MoresleepConfig holds moresleep API client configuration
type MoresleepConfig struct {
	URL      string `env:"URL" envDefault:"http://localhost:8082"`
	User     string `env:"USER"`
	Password string `env:"PASSWORD"`
}

// HasCredentials returns true if authentication credentials are configured
func (c *MoresleepConfig) HasCredentials() bool {
	return c.User != "" && c.Password != ""
}
