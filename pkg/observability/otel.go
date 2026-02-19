package observability

import (
	"context"

	"github.com/paul-cloud-game-backend/paul-cloud-game-backend/pkg/config"
	"github.com/rs/zerolog"
)

// InitOTEL provides optional OpenTelemetry initialization scaffolding.
// It is a no-op unless ENABLE_OTEL=true.
func InitOTEL(ctx context.Context, cfg config.Config, logger zerolog.Logger) (func(context.Context) error, error) {
	if !cfg.EnableOTEL {
		logger.Info().Msg("otel disabled")
		return func(context.Context) error { return nil }, nil
	}

	logger.Info().
		Str("otlp_endpoint", cfg.OTELEndpoint).
		Msg("otel enabled (scaffolding mode): exporter/provider wiring is not configured yet")

	return func(context.Context) error {
		logger.Info().Msg("otel shutdown complete")
		return nil
	}, nil
}
