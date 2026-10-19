package config

import "os"

type Config struct {
	DatabaseURL string
	Port        string
	JWTSecret   string
}

// Load reads configuration from the real environment. PORT defaults to
// "8080"; DATABASE_URL and JWT_SECRET are read directly from the env.
func Load() (*Config, error) {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	return &cfg, nil
}
