package cli

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

type loggerKey struct{}

func WithLogger(ctx context.Context) context.Context {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	logger := slog.New(handler)

	return context.WithValue(ctx, loggerKey{}, logger)
}

func GetLogger(cmd *cobra.Command) *slog.Logger {
	v := cmd.Context().Value(loggerKey{})
	if log, ok := v.(*slog.Logger); ok {
		return log
	}

	log := slog.Default()
	log.Debug("using the default logger")

	return log
}
