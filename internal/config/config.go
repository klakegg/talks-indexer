package config

// Config holds all application configuration loaded from environment variables
type Config struct {
	ApplicationConfig
	Http          HttpConfig          `envPrefix:"HTTP_"`
	Moresleep     MoresleepConfig     `envPrefix:"MORESLEEP_"`
	Elasticsearch ElasticsearchConfig `envPrefix:"ELASTICSEARCH_"`
	Index         IndexConfig
	OIDC          OIDCConfig `envPrefix:"OIDC_"`
}
