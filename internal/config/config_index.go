package config

// IndexConfig holds index name configuration
type IndexConfig struct {
	Private string `env:"PRIVATE_INDEX" envDefault:"javazone_private"`
	Public  string `env:"PUBLIC_INDEX" envDefault:"javazone_public"`
}
