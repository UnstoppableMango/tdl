package util

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func GitRoot(ctx context.Context) (string, error) {
	revParse, err := exec.CommandContext(ctx,
		"git", "rev-parse", "--show-toplevel",
	).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git root: %w", err)
	}

	return strings.TrimSpace(string(revParse)), nil
}
