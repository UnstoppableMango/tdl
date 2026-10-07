package jsonschema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Object is a JSON object that keeps its keys in the order they were set,
// so a schema reads in declaration order and the output is stable.
type Object struct {
	keys []string
	vals map[string]any
}

// NewObject returns an empty object.
func NewObject() *Object { return &Object{vals: map[string]any{}} }

// Set adds or replaces a key. A value is a string, a bool, an int64, a
// float64, a json.Number, an *Object, or a []any of those.
func (o *Object) Set(key string, v any) *Object {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = v
	return o
}

// Get returns a key's value.
func (o *Object) Get(key string) (any, bool) {
	v, ok := o.vals[key]
	return v, ok
}

// Keys returns the keys in the order they were set.
func (o *Object) Keys() []string { return o.keys }

// Len is the number of keys.
func (o *Object) Len() int { return len(o.keys) }

func (o *Object) del(key string) {
	if _, ok := o.vals[key]; !ok {
		return
	}
	delete(o.vals, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			return
		}
	}
}

// WriteJSON renders a value as JSON indented by two spaces.
func WriteJSON(b *bytes.Buffer, v any) { write(b, v, "") }

func write(b *bytes.Buffer, v any, indent string) {
	switch v := v.(type) {
	case *Object:
		if len(v.keys) == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteString("{\n")
		for i, k := range v.keys {
			b.WriteString(indent + "  ")
			WriteString(b, k)
			b.WriteString(": ")
			write(b, v.vals[k], indent+"  ")
			if i < len(v.keys)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString(indent + "}")
	case []any:
		if len(v) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteString("[\n")
		for i, e := range v {
			b.WriteString(indent + "  ")
			write(b, e, indent+"  ")
			if i < len(v)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString(indent + "]")
	case string:
		WriteString(b, v)
	case bool, int64, float64, json.Number:
		fmt.Fprint(b, v)
	default:
		panic(fmt.Sprintf("jsonschema: cannot write %T", v))
	}
}

// WriteString writes a JSON string without escaping <, >, and &, which
// encoding/json does by default and a pattern would read badly with.
func WriteString(b *bytes.Buffer, s string) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	b.WriteString(strings.TrimSuffix(buf.String(), "\n"))
}
