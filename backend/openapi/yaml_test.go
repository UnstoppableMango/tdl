package openapi

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/unstoppablemango/tdl/backend/internal/jsonschema"
)

// Shapes the generated documents do not reach today, such as empty and
// nested sequences, still read back as the JSON they stand for.
func TestYAMLReadsBackAsJSON(t *testing.T) {
	doc := jsonschema.NewObject().
		Set("empty", []any{}).
		Set("none", jsonschema.NewObject()).
		Set("items", []any{
			jsonschema.NewObject(),
			[]any{},
			[]any{"a", []any{int64(1), true}},
			jsonschema.NewObject().Set("x", "y").Set("n", json.Number("1.5")),
			"no",
			float64(2),
		})

	var y, j bytes.Buffer
	writeYAML(&y, doc)
	jsonschema.WriteJSON(&j, doc)

	var fromYAML, fromJSON any
	if err := yaml.Unmarshal(y.Bytes(), &fromYAML); err != nil {
		t.Fatalf("not YAML: %v\n%s", err, y.String())
	}
	if err := json.Unmarshal(j.Bytes(), &fromJSON); err != nil {
		t.Fatal(err)
	}
	// Round the YAML side through JSON so numbers compare alike.
	b, err := json.Marshal(fromYAML)
	if err != nil {
		t.Fatal(err)
	}
	var normalized any
	if err := json.Unmarshal(b, &normalized); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(normalized, fromJSON) {
		t.Errorf("YAML reads back as %v, want %v\n%s", normalized, fromJSON, y.String())
	}
}

func TestYAMLRefusesAnUnknownValue(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("no panic")
		}
	}()
	scalar(struct{}{})
}
