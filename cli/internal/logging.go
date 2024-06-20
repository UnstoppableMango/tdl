package cli

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

type loggerKey struct{}

func WithLogger(ctx context.Context) context.Context {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	return context.WithValue(ctx, &loggerKey{}, logger)
}

func GetLogger(cmd *cobra.Command) slog.Logger {
	return cmd.Context().Value(&loggerKey{}).(slog.Logger)
}
