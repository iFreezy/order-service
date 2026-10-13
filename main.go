package main

import (
	"os"

	"github.com/iFreezy/order-service/cmd"
	"github.com/rs/zerolog/log"
)

func main() {
	if err := cmd.NewApp().Run(os.Args); err != nil {
		log.Error().Err(err).Msg("Application failed")
		os.Exit(1)
	}
}
