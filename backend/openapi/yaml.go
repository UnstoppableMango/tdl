package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/jsonschema"
)

// writeYAML renders a document in block style. A string is written plain
// when YAML would read it back as the same string, and double-quoted with
// JSON's escapes, which YAML shares, otherwise.
func writeYAML(b *bytes.Buffer, doc *jsonschema.Object) {
	mapping(b, doc, "")
}

// mapping writes each key of a non-empty object on its own line.
func mapping(b *bytes.Buffer, o *jsonschema.Object, indent string) {
	for _, k := range o.Keys() {
		b.WriteString(indent)
		entry(b, o, k, indent)
	}
}

// entry writes `key: value` from the key's column.
func entry(b *bytes.Buffer, o *jsonschema.Object, k, indent string) {
	v, _ := o.Get(k)
	b.WriteString(scalar(k) + ":")
	switch v := v.(type) {
	case *jsonschema.Object:
		if v.Len() == 0 {
			b.WriteString(" {}\n")
			return
		}
		b.WriteString("\n")
		mapping(b, v, indent+"  ")
	case []any:
		if len(v) == 0 {
			b.WriteString(" []\n")
			return
		}
		b.WriteString("\n")
		for _, e := range v {
			b.WriteString(indent + "  - ")
			item(b, e, indent+"    ")
		}
	default:
		b.WriteString(" " + scalar(v) + "\n")
	}
}

// item writes a sequence entry from just after its `- `, where an object's
// first key sits and the rest line up beneath it.
func item(b *bytes.Buffer, v any, indent string) {
	switch v := v.(type) {
	case *jsonschema.Object:
		if v.Len() == 0 {
			b.WriteString("{}\n")
			return
		}
		for i, k := range v.Keys() {
			if i > 0 {
				b.WriteString(indent)
			}
			entry(b, v, k, indent)
		}
	case []any:
		if len(v) == 0 {
			b.WriteString("[]\n")
			return
		}
		for i, e := range v {
			if i > 0 {
				b.WriteString(indent)
			}
			b.WriteString("- ")
			item(b, e, indent+"  ")
		}
	default:
		b.WriteString(scalar(v) + "\n")
	}
}

// plain is a string YAML reads back unchanged without quotes, short of
// the words in [keywords].
var plain = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_.$/-]*$`)

// keywords are what YAML 1.1 or 1.2 reads as a bool or a null.
var keywords = map[string]bool{
	"y": true, "n": true, "yes": true, "no": true, "true": true, "false": true,
	"on": true, "off": true, "null": true,
}

func scalar(v any) string {
	switch v := v.(type) {
	case string:
		if plain.MatchString(v) && !keywords[strings.ToLower(v)] {
			return v
		}
		var buf bytes.Buffer
		jsonschema.WriteString(&buf, v)
		return buf.String()
	case bool, int64, float64, json.Number:
		return fmt.Sprint(v)
	}
	panic(fmt.Sprintf("openapi: cannot write %T", v))
}
