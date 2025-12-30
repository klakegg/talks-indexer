package config

import "fmt"

// HttpConfig holds HTTP server configuration
type HttpConfig struct {
	Host string `env:"HOST" envDefault:"0.0.0.0"`
	Port int    `env:"PORT" envDefault:"8080"`
}

// Addr returns the address string for the HTTP server
func (c *HttpConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
