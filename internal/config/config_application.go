package config

// Mode represents the application running mode
type Mode string

const (
	ModeProduction  Mode = "production"
	ModeDevelopment Mode = "development"
)

// IsDevelopment returns true if the mode is development
func (m Mode) IsDevelopment() bool {
	return m == ModeDevelopment
}

// IsProduction returns true if the mode is production
func (m Mode) IsProduction() bool {
	return m == ModeProduction
}

// ApplicationConfig holds application-level configuration
type ApplicationConfig struct {
	Mode Mode `env:"MODE" envDefault:"production"`
}
