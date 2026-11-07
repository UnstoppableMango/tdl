package yolo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/unstoppablemango/tdl/docs"
)

// RecordingFile is the file in a case directory holding its conversation.
const RecordingFile = "conversation.json"

// specMarker stands in for docs/spec.md in a recorded system prompt, so
// editing the spec does not make every recording stale.
const specMarker = "{{docs/spec.md}}"

// Recording is one conversation: what was sent first, then each turn's new
// messages and the model's reply.
type Recording struct {
	Model  string `json:"model"`
	System string `json:"system"`
	Tools  []Tool `json:"tools"`
	Turns  []Turn `json:"turns"`
}

// Turn is the messages the backend added since the last reply, and the
// reply they got.
type Turn struct {
	Sent  []Message `json:"sent"`
	Reply Message   `json:"reply"`
}

// Replay is the [Provider] that answers from a recording, and fails when
// the backend sends something the recording did not.
type Replay struct {
	rec  *Recording
	next int
}

// LoadReplay reads dir's recording.
func LoadReplay(dir string) (*Replay, error) {
	data, err := os.ReadFile(filepath.Join(dir, RecordingFile))
	if err != nil {
		return nil, err
	}
	var rec Recording
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Join(dir, RecordingFile), err)
	}
	return &Replay{rec: &rec}, nil
}

// Send returns the next recorded reply, once the conversation matches what
// was recorded up to it.
func (r *Replay) Send(_ context.Context, c *Conversation) (Message, error) {
	const stale = "; re-record with go test ./backend/yolo -record"
	switch {
	case c.Model != r.rec.Model:
		return Message{}, fmt.Errorf("the model is %q, the recording %q%s", c.Model, r.rec.Model, stale)
	case recordedSystem(c.System) != r.rec.System:
		return Message{}, fmt.Errorf("the system prompt differs from the recording%s", stale)
	case !reflect.DeepEqual(normalTools(c.Tools), normalTools(r.rec.Tools)):
		return Message{}, fmt.Errorf("the tools differ from the recording%s", stale)
	case r.next >= len(r.rec.Turns):
		return Message{}, fmt.Errorf("the recording ends after %d turns%s", len(r.rec.Turns), stale)
	}

	// Everything before this turn must still be what was recorded, since a
	// live provider would see it again.
	var history []Message
	for _, t := range r.rec.Turns[:r.next] {
		history = append(history, t.Sent...)
		history = append(history, t.Reply)
	}
	if len(c.Messages) < len(history) || !sameMessages(c.Messages[:len(history)], history) {
		return Message{}, fmt.Errorf("turn %d changed the conversation before it%s", r.next+1, stale)
	}
	sent := c.Messages[len(history):]
	turn := r.rec.Turns[r.next]
	if !sameMessages(sent, turn.Sent) {
		got, _ := json.MarshalIndent(sent, "", "  ")
		return Message{}, fmt.Errorf("turn %d sent something other than the recording%s:\n%s", r.next+1, stale, got)
	}
	r.next++
	return turn.Reply, nil
}

// sentSince is how many messages the turns so far account for: each
// turn's sent messages and its reply.
func sentSince(turns []Turn) int {
	n := 0
	for _, t := range turns {
		n += len(t.Sent) + 1
	}
	return n
}

// Recorder is a [Provider] that passes each turn to another and keeps the
// conversation, for [Recorder.Save] to write as a recording.
type Recorder struct {
	Provider Provider
	rec      Recording
}

// Send passes the turn on and records it.
func (r *Recorder) Send(ctx context.Context, c *Conversation) (Message, error) {
	seen := sentSince(r.rec.Turns)
	if len(c.Messages) < seen {
		return Message{}, fmt.Errorf("the conversation holds %d messages, fewer than the %d recorded before it", len(c.Messages), seen)
	}
	reply, err := r.Provider.Send(ctx, c)
	if err != nil {
		return reply, err
	}
	r.rec.Model, r.rec.System, r.rec.Tools = c.Model, recordedSystem(c.System), c.Tools
	sent := make([]Message, 0, len(c.Messages))
	for _, m := range c.Messages[seen:] {
		m.native = nil
		sent = append(sent, m)
	}
	recorded := reply
	recorded.native = nil
	// The history holds a reply as the backend's own; record it so.
	recorded.Role = Assistant
	r.rec.Turns = append(r.rec.Turns, Turn{Sent: sent, Reply: recorded})
	return reply, nil
}

// Save writes the recording to dir, laid out as jsonfmt lays it out.
func (r *Recorder) Save(dir string) error {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")
	if err := enc.Encode(r.rec); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, RecordingFile), b.Bytes(), 0o644)
}

func recordedSystem(s string) string {
	return strings.ReplaceAll(s, docs.Spec, specMarker)
}

// normalTools compares tools by their schema's meaning, not its spacing.
func normalTools(tools []Tool) []any {
	out := make([]any, len(tools))
	for i, t := range tools {
		var schema any
		_ = json.Unmarshal(t.Schema, &schema)
		out[i] = []any{t.Name, t.Description, schema}
	}
	return out
}

// sameMessages compares messages as JSON, so a call's input compares by
// meaning and the native form is ignored.
func sameMessages(a, b []Message) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !sameJSON(a[i], b[i]) {
			return false
		}
	}
	return true
}

func sameJSON(a, b any) bool {
	ja, err1 := json.Marshal(a)
	jb, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	var va, vb any
	if json.Unmarshal(ja, &va) != nil || json.Unmarshal(jb, &vb) != nil {
		return bytes.Equal(ja, jb)
	}
	return reflect.DeepEqual(va, vb)
}
