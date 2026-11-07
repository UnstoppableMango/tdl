package yolo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

// DefaultModel is the model a target block without a model directive
// uses.
const DefaultModel = "claude-opus-5-5"

// maxTokens bounds one turn. A turn writing a large file needs room, and
// streaming keeps a long turn clear of the HTTP timeout.
const maxTokens = 64000

// Anthropic is the [Provider] for Anthropic's Messages API.
type Anthropic struct {
	client anthropic.Client
}

// NewAnthropic returns a provider authenticating with key.
func NewAnthropic(key string, opts ...option.RequestOption) *Anthropic {
	return &Anthropic{client: anthropic.NewClient(append([]option.RequestOption{option.WithAPIKey(key)}, opts...)...)}
}

// Send streams one turn and returns it whole.
func (a *Anthropic) Send(ctx context.Context, c *Conversation) (Message, error) {
	tools, err := anthropicTools(c.Tools)
	if err != nil {
		return Message{}, err
	}
	msgs := make([]anthropic.MessageParam, 0, len(c.Messages))
	for _, m := range c.Messages {
		msgs = append(msgs, anthropicMessage(m))
	}
	params := anthropic.MessageNewParams{
		Model:     c.Model,
		MaxTokens: maxTokens,
		System:    []anthropic.TextBlockParam{{Text: c.System}},
		Tools:     tools,
		Messages:  msgs,
		// Mapping a model is the kind of work effort pays for.
		OutputConfig: anthropic.OutputConfigParam{Effort: anthropic.OutputConfigEffortHigh},
		// Every turn resends the system prompt, the spec, and the model.
		CacheControl: anthropic.NewCacheControlEphemeralParam(),
	}

	stream := a.client.Messages.NewStreaming(ctx, params)
	defer stream.Close()
	var reply anthropic.Message
	stopped := false
	for stream.Next() {
		event := stream.Current()
		if err := reply.Accumulate(event); err != nil {
			return Message{}, err
		}
		stopped = stopped || event.Type == "message_stop"
	}
	if err := stream.Err(); err != nil {
		return Message{}, err
	}
	// The stop reason arrives before message_stop, so a stream cut off
	// between the two would otherwise pass for a finished turn.
	if !stopped {
		return Message{}, errors.New("the stream ended before the turn did")
	}

	out := Message{Role: Assistant, native: reply.ToParam()}
	var text []string
	for _, block := range reply.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			text = append(text, b.Text)
		case anthropic.ToolUseBlock:
			out.Calls = append(out.Calls, Call{ID: b.ID, Name: b.Name, Input: b.Input})
		}
	}
	out.Text = strings.Join(text, "\n")
	if out.Stop, err = anthropicStop(reply.StopReason); err != nil {
		return Message{}, err
	}
	return out, nil
}

// anthropicStop is why a turn ended. A stream that closed before saying
// so ended early, and a reason this provider does not know is not a
// turn it can trust to be whole.
func anthropicStop(r anthropic.StopReason) (Stop, error) {
	switch r {
	case anthropic.StopReasonEndTurn, anthropic.StopReasonStopSequence:
		return StopEnd, nil
	case anthropic.StopReasonToolUse:
		return StopTool, nil
	case anthropic.StopReasonMaxTokens, anthropic.StopReasonModelContextWindowExceeded:
		return StopLength, nil
	case anthropic.StopReasonRefusal:
		return StopRefused, nil
	case "":
		return "", errors.New("the stream ended before the turn did")
	}
	return "", fmt.Errorf("the turn stopped for a reason this backend does not know, %q", r)
}

// anthropicMessage is a message as the API takes it: a reply this
// provider returned as it came, and anything else built from its parts.
func anthropicMessage(m Message) anthropic.MessageParam {
	if p, ok := m.native.(anthropic.MessageParam); ok {
		return p
	}
	var blocks []anthropic.ContentBlockParamUnion
	if m.Text != "" {
		blocks = append(blocks, anthropic.NewTextBlock(m.Text))
	}
	for _, r := range m.Results {
		blocks = append(blocks, anthropic.NewToolResultBlock(r.ID, r.Content, r.Error))
	}
	for _, c := range m.Calls {
		blocks = append(blocks, anthropic.NewToolUseBlock(c.ID, c.Input, c.Name))
	}
	if m.Role == Assistant {
		return anthropic.NewAssistantMessage(blocks...)
	}
	return anthropic.NewUserMessage(blocks...)
}

func anthropicTools(tools []Tool) ([]anthropic.ToolUnionParam, error) {
	out := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		var schema struct {
			Properties map[string]any `json:"properties"`
			Required   []string       `json:"required"`
		}
		if err := json.Unmarshal(t.Schema, &schema); err != nil {
			return nil, fmt.Errorf("tool %s: %w", t.Name, err)
		}
		out = append(out, anthropic.ToolUnionParam{OfTool: &anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: schema.Properties, Required: schema.Required},
		}})
	}
	return out, nil
}
