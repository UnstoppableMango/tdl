package jsonschema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// object is a JSON object that keeps its keys in the order they were set,
// so a schema reads in declaration order and the output is stable.
type object struct {
	keys []string
	vals map[string]any
}

func newObject() *object { return &object{vals: map[string]any{}} }

// set adds or replaces a key. A value is a string, a bool, an int64, a
// float64, a json.Number, an *object, or a []any of those.
func (o *object) set(key string, v any) *object {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = v
	return o
}

func (o *object) get(key string) (any, bool) {
	v, ok := o.vals[key]
	return v, ok
}

func (o *object) del(key string) {
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

// write renders a value as JSON indented by two spaces.
func write(b *bytes.Buffer, v any, indent string) {
	switch v := v.(type) {
	case *object:
		if len(v.keys) == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteString("{\n")
		for i, k := range v.keys {
			b.WriteString(indent + "  ")
			writeString(b, k)
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
		writeString(b, v)
	case bool, int64, float64, json.Number:
		fmt.Fprint(b, v)
	default:
		panic(fmt.Sprintf("jsonschema: cannot write %T", v))
	}
}

// writeString writes a JSON string without escaping <, >, and &, which
// encoding/json does by default and a pattern would read badly with.
func writeString(b *bytes.Buffer, s string) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	b.WriteString(strings.TrimSuffix(buf.String(), "\n"))
}
