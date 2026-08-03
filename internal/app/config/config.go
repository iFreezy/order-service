package config

import (
	"log"

	"github.com/iFreezy/order-service/internal/app/config/section"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Repository section.Repository
	Processor  section.Processor
	Monitor    section.Monitor
}

var Root Config

// Load reads the .env file (when present) and fills Root from the APP_* env
// variables. A missing .env is not fatal, an invalid configuration is.
func Load() {
	if err := godotenv.Load(); err != nil {
		log.Printf("load .env: %v", err)
	}

	if err := envconfig.Process("APP", &Root); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
}
