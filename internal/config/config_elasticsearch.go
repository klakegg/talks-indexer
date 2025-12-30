package config

// ElasticsearchConfig holds Elasticsearch client configuration
type ElasticsearchConfig struct {
	URL      string `env:"URL" envDefault:"http://localhost:9200"`
	User     string `env:"USER"`
	Password string `env:"PASSWORD"`
}

// HasCredentials returns true if authentication credentials are configured
func (c *ElasticsearchConfig) HasCredentials() bool {
	return c.User != "" && c.Password != ""
}
