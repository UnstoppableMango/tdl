package yolo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Provider is a language model service: it takes a conversation and
// returns the model's next turn. Anthropic is the only one; the interface
// keeps a second one a new type rather than a redesign.
type Provider interface {
	Send(ctx context.Context, c *Conversation) (Message, error)
}

// Conversation is everything a provider sends: the model's name, the
// system prompt, the tools the model may call, and the messages so far.
type Conversation struct {
	Model    string
	System   string
	Tools    []Tool
	Messages []Message
}

// Tool is a tool the model may call. Schema is a JSON Schema object with
// properties and required.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"`
}

// Role is who wrote a message.
type Role string

const (
	User      Role = "user"
	Assistant Role = "assistant"
)

// Stop is why the model ended a turn.
type Stop string

const (
	// StopEnd is a turn the model finished.
	StopEnd Stop = "end"
	// StopTool is a turn waiting for its calls' results.
	StopTool Stop = "tool"
	// StopLength is a turn cut off at the token limit.
	StopLength Stop = "length"
	// StopRefused is a turn the provider declined to finish.
	StopRefused Stop = "refused"
)

// Message is one turn. A user message carries text or the results of the
// previous turn's calls; an assistant message carries text, calls, and why
// it stopped.
type Message struct {
	Role    Role     `json:"role"`
	Text    string   `json:"text,omitempty"`
	Calls   []Call   `json:"calls,omitempty"`
	Results []Result `json:"results,omitempty"`
	Stop    Stop     `json:"stop,omitempty"`

	// native is the provider's own form of a reply, sent back unchanged so
	// that what the provider needs to see again (thinking, say) survives.
	// It is never recorded.
	native any
}

// Call is the model calling a tool.
type Call struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// Result answers a call. Error says the call was refused.
type Result struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Error   bool   `json:"error,omitempty"`
}

// ReplayEnv names a directory holding a recorded conversation. When it is
// set, the backend replays that instead of calling a provider, so tests
// and checks run with no network and no key.
const ReplayEnv = "TDL_YOLO_REPLAY"

// KeyEnv is where the Anthropic provider reads its key.
const KeyEnv = "ANTHROPIC_API_KEY"

// errNoKey reports a missing key, before anything is sent.
var errNoKey = errors.New("the yolo backend calls Anthropic's API and needs " + KeyEnv + " set")

// FromEnv is the provider the environment names: a replay of the
// directory ReplayEnv names, or Anthropic with the key KeyEnv holds.
func FromEnv() (Provider, error) {
	if dir := os.Getenv(ReplayEnv); dir != "" {
		r, err := LoadReplay(dir)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ReplayEnv, err)
		}
		return r, nil
	}
	key := os.Getenv(KeyEnv)
	if key == "" {
		return nil, errNoKey
	}
	return NewAnthropic(key), nil
}
