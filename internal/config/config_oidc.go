package config

// OIDCConfig holds OIDC authentication configuration (only used in production mode)
type OIDCConfig struct {
	IssuerURL    string `env:"ISSUER_URL"`
	ClientID     string `env:"CLIENT_ID"`
	ClientSecret string `env:"CLIENT_SECRET"`
	RedirectURL  string `env:"REDIRECT_URL"`
}

// IsConfigured returns true if OIDC is fully configured
func (c *OIDCConfig) IsConfigured() bool {
	return c.IssuerURL != "" &&
		c.ClientID != "" &&
		c.ClientSecret != ""
}
