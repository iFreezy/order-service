package main

import (
	"context"
	"log"

	"github.com/iFreezy/order-service/internal/app/config"
	rhealth "github.com/iFreezy/order-service/internal/app/handler/http/health"
	rprocessor "github.com/iFreezy/order-service/internal/app/processor/http"
	rcpostgres "github.com/iFreezy/order-service/internal/app/repository/conn/postgres"
)

func main() {
	config.Load()

	cfg := config.Root

	client, err := rcpostgres.NewClient(context.Background(), cfg.Repository.Postgres)
	if err != nil {
		log.Fatalf("Failed to connect to postgres: %v", err)
	}

	defer func() {
		if err = client.Close(); err != nil {
			log.Printf("close postgres connection: %v", err)
		}
	}()

	log.Printf("Connected to database: %s", cfg.Repository.Postgres.Name)

	healthHandler := rhealth.NewHandler()

	httpProcessor := rprocessor.NewHTTP(healthHandler, cfg.Processor.WebServer)

	if err = httpProcessor.Serve(); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}
