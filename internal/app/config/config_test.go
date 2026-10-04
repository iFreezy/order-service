package config

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestLoggingConfiguration(t *testing.T) {
	previous, previousRoot := log.Logger, Root
	previousTimestamp, previousMessage := zerolog.TimestampFieldName, zerolog.MessageFieldName
	t.Cleanup(func() {
		log.Logger, Root = previous, previousRoot
		zerolog.TimestampFieldName, zerolog.MessageFieldName = previousTimestamp, previousMessage
	})
	t.Setenv("APP_REPOSITORY_POSTGRES_ADDRESS", "127.0.0.1:5432")
	t.Setenv("APP_REPOSITORY_POSTGRES_USERNAME", "order_test")
	t.Setenv("APP_REPOSITORY_POSTGRES_PASSWORD", "order_test")
	t.Setenv("APP_REPOSITORY_POSTGRES_NAME", "order_test")
	t.Setenv("APP_MONITOR_LOG_LEVEL", "error")
	var output bytes.Buffer
	if err := Load(LoadArgs{Output: &output}); err != nil {
		t.Fatal(err)
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(strings.Split(output.String(), "\n")[0]), &first); err != nil {
		t.Fatal(err)
	}
	if first["level"] != "debug" || first["timestamp"] == nil || first["msg"] == nil {
		t.Fatalf("early logger: %s", output.String())
	}
	output.Reset()
	log.Debug().Msg("hidden")
	log.Error().Msg("visible")
	if strings.Contains(output.String(), "hidden") || !strings.Contains(output.String(), "visible") {
		t.Fatalf("configured level ignored: %s", output.String())
	}
	t.Setenv("APP_MONITOR_LOG_LEVEL", "debug")
	output.Reset()
	if err := Load(LoadArgs{Output: &output, EnableSimpleLog: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Logger re-initialized") || strings.HasPrefix(output.String(), "{") {
		t.Fatalf("console logger: %s", output.String())
	}
	t.Setenv("APP_MONITOR_LOG_LEVEL", "invalid")
	if err := Load(LoadArgs{Output: &output}); err == nil {
		t.Fatal("invalid log level must fail initialization")
	}
}
