package logging

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// New returns a zerolog logger pre-configured with app and service metadata.
func New(appName, serviceName, env string) zerolog.Logger {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	logger := zerolog.New(output).
		With().
		Timestamp().
		Str("app", appName).
		Str("service", serviceName).
		Str("env", env).
		Logger()

	return logger
}
