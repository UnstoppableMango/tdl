package logging

import (
	"context"
	"log/slog"
	"os"
)

type loggerKey int

var key loggerKey

func NewLogger() *slog.Logger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	return slog.New(handler)
}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, key, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	val := ctx.Value(key)
	if log, ok := val.(*slog.Logger); ok {
		return log
	}

	return slog.Default()
}
