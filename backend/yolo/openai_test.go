package yolo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openai/openai-go/v3/option"
)

func TestOpenAIStop(t *testing.T) {
	for reason, want := range map[string]Stop{
		"stop":           StopEnd,
		"tool_calls":     StopTool,
		"function_call":  StopTool,
		"length":         StopLength,
		"content_filter": StopRefused,
	} {
		if got, err := openAIStop(reason); err != nil || got != want {
			t.Errorf("%q: %q, %v; want %q", reason, got, err, want)
		}
	}
	for _, reason := range []string{"", "something_new"} {
		if got, err := openAIStop(reason); err == nil {
			t.Errorf("%q: %q, want an error", reason, got)
		}
	}
}

// chatServer answers each request with the next reply, and keeps what was
// asked of it.
type chatServer struct {
	*httptest.Server
	paths    []string
	requests []map[string]any
}

func newChatServer(t *testing.T, replies ...string) *chatServer {
	s := &chatServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("request %s: %v", body, err)
		}
		s.paths = append(s.paths, r.URL.Path)
		s.requests = append(s.requests, req)
		n := len(s.requests) - 1
		if n >= len(replies) {
			http.Error(w, "no more replies", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, replies[n])
	}))
	t.Cleanup(s.Close)
	return s
}

func completion(message, finish string) string {
	return `{"id":"c","object":"chat.completion","created":1,"model":"m","choices":[{"index":0,"finish_reason":"` +
		finish + `","message":` + message + `}]}`
}

// A call goes out as a function tool, comes back as a call, and its
// result goes back as a tool message after the assistant's turn.
func TestOpenAIToolRoundTrip(t *testing.T) {
	// The call carries no type, as some local servers write it.
	srv := newChatServer(t,
		completion(`{"role":"assistant","content":null,"tool_calls":[{"id":"call_1","function":{"name":"write_file","arguments":"{\"path\":\"a.kt\",\"content\":\"x\"}"}}]}`, "tool_calls"),
		completion(`{"role":"assistant","content":"Done."}`, "stop"),
	)
	o := NewOpenAI(option.WithBaseURL(srv.URL+"/v1/"), option.WithAPIKey("key"), option.WithMaxRetries(0))
	conv := &Conversation{Model: "m", System: "sys", Tools: tools, Messages: []Message{{Role: User, Text: "hi"}}}

	reply, err := o.Send(context.Background(), conv)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Stop != StopTool || len(reply.Calls) != 1 || reply.Calls[0].ID != "call_1" ||
		reply.Calls[0].Name != "write_file" || string(reply.Calls[0].Input) != `{"path":"a.kt","content":"x"}` {
		t.Fatalf("reply %+v", reply)
	}
	conv.Messages = append(conv.Messages, reply, Message{Role: User, Results: []Result{
		{ID: "call_1", Content: "wrote a.kt"},
	}})
	reply, err = o.Send(context.Background(), conv)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Stop != StopEnd || reply.Text != "Done." || len(reply.Calls) != 0 {
		t.Errorf("reply %+v", reply)
	}

	for _, p := range srv.paths {
		if p != "/v1/chat/completions" {
			t.Errorf("path %q, want the base URL's", p)
		}
	}
	first := srv.requests[0]
	if first["model"] != "m" {
		t.Errorf("model %v", first["model"])
	}
	var names []string
	for _, tool := range first["tools"].([]any) {
		fn := tool.(map[string]any)["function"].(map[string]any)
		names = append(names, fn["name"].(string))
		if fn["parameters"].(map[string]any)["type"] != "object" {
			t.Errorf("%s's parameters %v", fn["name"], fn["parameters"])
		}
	}
	if strings.Join(names, ",") != "write_file,report_loss" {
		t.Errorf("tools %v", names)
	}

	msgs := srv.requests[1]["messages"].([]any)
	roles := make([]string, len(msgs))
	for i, m := range msgs {
		roles[i] = m.(map[string]any)["role"].(string)
	}
	if strings.Join(roles, ",") != "system,user,assistant,tool" {
		t.Fatalf("roles %v", roles)
	}
	calls := msgs[2].(map[string]any)["tool_calls"].([]any)
	if len(calls) != 1 || calls[0].(map[string]any)["id"] != "call_1" {
		t.Errorf("the assistant's turn went back as %v", msgs[2])
	}
	if tool := msgs[3].(map[string]any); tool["tool_call_id"] != "call_1" || tool["content"] != "wrote a.kt" {
		t.Errorf("the result went back as %v", tool)
	}
}

// reasoning_effort is sent only when the conversation names an effort.
func TestOpenAIEffort(t *testing.T) {
	for effort, want := range map[string]any{"": nil, "low": "low"} {
		srv := newChatServer(t, completion(`{"role":"assistant","content":"ok"}`, "stop"))
		o := NewOpenAI(option.WithBaseURL(srv.URL), option.WithAPIKey("key"), option.WithMaxRetries(0))
		if _, err := o.Send(context.Background(), &Conversation{Model: "m", Effort: effort, Messages: []Message{{Role: User, Text: "hi"}}}); err != nil {
			t.Fatal(err)
		}
		if got := srv.requests[0]["reasoning_effort"]; got != want {
			t.Errorf("effort %q sent reasoning_effort %v, want %v", effort, got, want)
		}
	}
}

// A conversation built from its parts, as a replayed one is, says what a
// live one does, and a refused call says so in its tool message.
func TestOpenAIMessagesFromParts(t *testing.T) {
	srv := newChatServer(t, completion(`{"role":"assistant","content":"ok"}`, "stop"))
	o := NewOpenAI(option.WithBaseURL(srv.URL), option.WithAPIKey("key"), option.WithMaxRetries(0))
	conv := &Conversation{Model: "m", Messages: []Message{
		{Role: User, Text: "hi"},
		{Role: Assistant, Text: "writing", Calls: []Call{{ID: "c1", Name: "write_file", Input: json.RawMessage(`{"path":"/x"}`)}}, Stop: StopTool},
		{Role: User, Results: []Result{{ID: "c1", Content: "no content", Error: true}}},
		{Role: Assistant, Stop: StopEnd},
	}}
	if _, err := o.Send(context.Background(), conv); err != nil {
		t.Fatal(err)
	}
	msgs := srv.requests[0]["messages"].([]any)
	asst := msgs[2].(map[string]any)
	fn := asst["tool_calls"].([]any)[0].(map[string]any)["function"].(map[string]any)
	if asst["content"] != "writing" || fn["name"] != "write_file" || fn["arguments"] != `{"path":"/x"}` {
		t.Errorf("assistant %v", asst)
	}
	if tool := msgs[3].(map[string]any); tool["content"] != "error: no content" {
		t.Errorf("tool %v", tool)
	}
	if empty, ok := msgs[4].(map[string]any)["content"]; !ok || empty != "" {
		t.Errorf("an empty assistant turn went as %v", msgs[4])
	}
}

// Arguments that are not JSON reach the backend as a string, which it
// refuses back to the model.
func TestOpenAIArgumentsNotJSON(t *testing.T) {
	srv := newChatServer(t, completion(`{"role":"assistant","content":null,"tool_calls":[{"id":"c","type":"function","function":{"name":"write_file","arguments":"{path: a"}}]}`, "tool_calls"))
	o := NewOpenAI(option.WithBaseURL(srv.URL), option.WithAPIKey("key"), option.WithMaxRetries(0))
	reply, err := o.Send(context.Background(), &Conversation{Model: "m", Messages: []Message{{Role: User, Text: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(reply.Calls[0].Input); got != `"{path: a"` {
		t.Errorf("input %s", got)
	}
}
