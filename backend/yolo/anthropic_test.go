package yolo

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
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

// A stream that closes after its stop reason but before message_stop is
// cut off, not finished; one that reaches message_stop is a turn.
func TestAnthropicStreamMustStop(t *testing.T) {
	events := []string{
		`event: message_start
data: {"type":"message_start","message":{"id":"m","type":"message","role":"assistant","model":"m","content":[],"stop_reason":null,"usage":{"input_tokens":1,"output_tokens":0}}}`,
		`event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Done."}}`,
		`event: content_block_stop
data: {"type":"content_block_stop","index":0}`,
		`event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":1}}`,
	}
	stop := `event: message_stop
data: {"type":"message_stop"}`
	for name, tc := range map[string]struct {
		events []string
		ok     bool
	}{
		"cut off":  {events, false},
		"finished": {append(slices.Clone(events), stop), true},
	} {
		t.Run(name, func(t *testing.T) {
			var body []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ = io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(w, strings.Join(tc.events, "\n\n")+"\n\n")
			}))
			defer srv.Close()
			a := NewAnthropic("key", option.WithBaseURL(srv.URL), option.WithMaxRetries(0))
			reply, err := a.Send(context.Background(), &Conversation{Model: "m", Effort: "low", Messages: []Message{{Role: User, Text: "hi"}}})
			if !strings.Contains(string(body), `"effort":"low"`) {
				t.Errorf("the request does not ask for the effort: %s", body)
			}
			if tc.ok && (err != nil || reply.Stop != StopEnd || reply.Text != "Done.") {
				t.Errorf("reply %+v, %v; want a finished turn", reply, err)
			}
			if !tc.ok && err == nil {
				t.Errorf("reply %+v, want an error", reply)
			}
		})
	}
}
