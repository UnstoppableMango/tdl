package yolo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

// OpenAIEfforts are the effort levels the openai provider sends as
// reasoning_effort; the rest of Efforts mean nothing there.
var OpenAIEfforts = []string{"low", "medium", "high"}

// OpenAI is the [Provider] for OpenAI's Chat Completions API, which
// OpenAI-compatible servers (OpenRouter, Ollama, llama.cpp, vLLM, LM
// Studio) also serve.
type OpenAI struct {
	client openai.Client
}

// NewOpenAI returns a provider configured by opts, on top of what the
// SDK reads from OPENAI_BASE_URL and OPENAI_API_KEY.
func NewOpenAI(opts ...option.RequestOption) *OpenAI {
	return &OpenAI{client: openai.NewClient(opts...)}
}

// Send asks for one turn. An empty Effort sends no reasoning_effort, so a
// server that does not know it is not refused.
func (o *OpenAI) Send(ctx context.Context, c *Conversation) (Message, error) {
	tools, err := openAITools(c.Tools)
	if err != nil {
		return Message{}, err
	}
	msgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(c.Messages)+1)
	msgs = append(msgs, openai.SystemMessage(c.System))
	for _, m := range c.Messages {
		msgs = append(msgs, openAIMessages(m)...)
	}
	params := openai.ChatCompletionNewParams{
		Model:    c.Model,
		Messages: msgs,
		Tools:    tools,
	}
	if c.Effort != "" {
		params.ReasoningEffort = shared.ReasoningEffort(c.Effort)
	}

	reply, err := o.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return Message{}, err
	}
	if len(reply.Choices) == 0 {
		return Message{}, errors.New("the reply held no choice")
	}
	choice := reply.Choices[0]
	// No native form: the SDK's own drops a call whose type a local
	// server left out, and a tool message would then answer nothing.
	out := Message{Role: Assistant, Text: choice.Message.Content}
	for _, tc := range choice.Message.ToolCalls {
		f := tc.AsFunction()
		input := json.RawMessage(f.Function.Arguments)
		// A local model may write arguments that are not JSON; passed on
		// as a string, the call is refused back to it rather than
		// breaking the conversation's record.
		if !json.Valid(input) {
			input, _ = json.Marshal(f.Function.Arguments)
		}
		out.Calls = append(out.Calls, Call{ID: f.ID, Name: f.Function.Name, Input: input})
	}
	if choice.Message.Refusal != "" {
		out.Stop = StopRefused
		return out, nil
	}
	if out.Stop, err = openAIStop(choice.FinishReason); err != nil {
		return Message{}, err
	}
	return out, nil
}

// openAIStop is why a turn ended.
func openAIStop(r string) (Stop, error) {
	switch r {
	case "stop":
		return StopEnd, nil
	case "tool_calls", "function_call":
		return StopTool, nil
	case "length":
		return StopLength, nil
	case "content_filter":
		return StopRefused, nil
	case "":
		return "", errors.New("the reply did not say why the turn ended")
	}
	return "", fmt.Errorf("the turn stopped for a reason this backend does not know, %q", r)
}

// openAIMessages is a message as the API takes it, built from its parts.
// A user message carrying results is one tool message per result, since
// the API has no other place for them.
func openAIMessages(m Message) []openai.ChatCompletionMessageParamUnion {
	if m.Role == Assistant {
		var p openai.ChatCompletionAssistantMessageParam
		// The API refuses an assistant message with neither content nor
		// calls.
		if m.Text != "" || len(m.Calls) == 0 {
			p.Content.OfString = openai.String(m.Text)
		}
		for _, c := range m.Calls {
			p.ToolCalls = append(p.ToolCalls, openai.ChatCompletionMessageToolCallUnionParam{
				OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
					ID: c.ID,
					Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
						Name:      c.Name,
						Arguments: string(c.Input),
					},
				},
			})
		}
		return []openai.ChatCompletionMessageParamUnion{{OfAssistant: &p}}
	}
	var out []openai.ChatCompletionMessageParamUnion
	for _, r := range m.Results {
		// A tool message has no error flag, so a refusal says so itself.
		content := r.Content
		if r.Error {
			content = "error: " + content
		}
		out = append(out, openai.ToolMessage(content, r.ID))
	}
	if m.Text != "" {
		out = append(out, openai.UserMessage(m.Text))
	}
	return out
}

func openAITools(tools []Tool) ([]openai.ChatCompletionToolUnionParam, error) {
	out := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, t := range tools {
		var schema shared.FunctionParameters
		if err := json.Unmarshal(t.Schema, &schema); err != nil {
			return nil, fmt.Errorf("tool %s: %w", t.Name, err)
		}
		out = append(out, openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        t.Name,
			Description: openai.String(t.Description),
			Parameters:  schema,
		}))
	}
	return out, nil
}
