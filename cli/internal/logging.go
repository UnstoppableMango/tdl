package cli

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/logging"
)

func FromCommand(cmd *cobra.Command) *slog.Logger {
	return logging.FromContext(cmd.Context())
}

func SetLogger(cmd *cobra.Command, logger *slog.Logger) {
	cmd.SetContext(logging.WithLogger(
		cmd.Context(),
		logger,
	))
}
