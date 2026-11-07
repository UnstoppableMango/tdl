package yolo

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go/v2"
)

func TestAnthropicStop(t *testing.T) {
	for reason, want := range map[anthropic.StopReason]Stop{
		anthropic.StopReasonEndTurn:                    StopEnd,
		anthropic.StopReasonStopSequence:               StopEnd,
		anthropic.StopReasonToolUse:                    StopTool,
		anthropic.StopReasonMaxTokens:                  StopLength,
		anthropic.StopReasonModelContextWindowExceeded: StopLength,
		anthropic.StopReasonRefusal:                    StopRefused,
	} {
		if got, err := anthropicStop(reason); err != nil || got != want {
			t.Errorf("%q: %q, %v; want %q", reason, got, err, want)
		}
	}
	// A stream cut off before its last event, and a reason added after
	// this backend was written, are not a finished turn.
	for _, reason := range []anthropic.StopReason{"", anthropic.StopReasonPauseTurn, "something_new"} {
		if got, err := anthropicStop(reason); err == nil {
			t.Errorf("%q: %q, want an error", reason, got)
		}
	}
}
