package cli

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

type loggerKey struct{}

func NewLogger() *slog.Logger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	return slog.New(handler)
}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	val := ctx.Value(loggerKey{})
	if log, ok := val.(*slog.Logger); ok {
		return log
	}

	return slog.Default()
}

func FromCommand(cmd *cobra.Command) *slog.Logger {
	return FromContext(cmd.Context())
}
