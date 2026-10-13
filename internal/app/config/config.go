package config

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/iFreezy/order-service/internal/app/config/section"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Repository section.Repository
	Processor  section.Processor
	Monitor    section.Monitor
}

var Root Config

type LoadArgs struct {
	Output          io.Writer `json:"-"`
	EnableSimpleLog bool
}

func Load(args LoadArgs) error {
	zerolog.TimestampFieldName = "timestamp"
	zerolog.MessageFieldName = "msg"
	zerolog.TimeFieldFormat = time.RFC3339
	output := args.Output
	if output == nil {
		output = os.Stdout
	}
	if args.EnableSimpleLog {
		output = zerolog.ConsoleWriter{Out: output, TimeFormat: time.RFC3339}
	}
	log.Logger = createLogger(zerolog.DebugLevel, output)
	log.Debug().Msg("Logger initialized with Debug level")
	if err := godotenv.Load(); err != nil {
		log.Debug().Err(err).Msg("Could not load .env")
	}

	if err := envconfig.Process("APP", &Root); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	level, err := zerolog.ParseLevel(Root.Monitor.LogLevel)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	log.Logger = createLogger(level, output)
	log.Info().Str("log_level", level.String()).Msg("Logger re-initialized with config level")
	return nil
}

func createLogger(level zerolog.Level, output io.Writer) zerolog.Logger {
	return zerolog.New(output).Level(level).With().Timestamp().Logger()
}
