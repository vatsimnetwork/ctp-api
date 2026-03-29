package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	AppPort        string
	DatabaseURL    string
	AuthServiceURL string
	SimulatorURL   string
}

var C *Config

func Load() {
	_ = godotenv.Load()

	C = &Config{
		AppPort:        getEnv("APP_PORT", "8080"),
		DatabaseURL:    requireEnv("DATABASE_URL"),
		AuthServiceURL: requireEnv("AUTH_SERVICE_URL"),
		SimulatorURL:   getEnv("SIMULATOR_URL", ""),
	}

	log.Info().
		Str("port", C.AppPort).
		Msg("config loaded")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatal().Msg(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}
