package smithy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// prelude is the namespace of Smithy's prelude shapes.
const prelude = "smithy.api"

// shape is one Smithy shape, as Generate writes it and import reads it
// from the JSON AST, so that generating can read its own output back.
type shape struct {
	// id is the absolute shape ID, such as shop#Order.
	id   string
	kind string

	traits traits

	// members is a structure's, a union's, or an enum's, in order.
	members []*member
	// member is a list's.
	member *member
	// key and value are a map's.
	key, value *member
}

func (s *shape) name() string {
	_, name, _ := strings.Cut(s.id, "#")
	return name
}

func (s *shape) namespace() string {
	ns, _, _ := strings.Cut(s.id, "#")
	return ns
}

// targets is every shape ID the shape's members target.
func (s *shape) targets() []string {
	var out []string
	for _, m := range s.members {
		out = append(out, m.target)
	}
	for _, m := range []*member{s.member, s.key, s.value} {
		if m != nil {
			out = append(out, m.target)
		}
	}
	return out
}

// member is a member of a shape: a field, an enum member, a union
// member, or a collection's element, key, or value.
type member struct {
	name   string
	target string
	traits traits
}

// traits is the traits generating writes and import reads. Any other
// trait is named in other.
type traits struct {
	doc        *string
	deprecated *deprecation
	required   bool
	sparse     bool
	unique     bool
	enumValue  *string
	// source is the tdl#source trait: the declaration as TDL.
	source *string
	// trait is set on a shape that defines a trait.
	trait bool
	other []string
}

type deprecation struct {
	message string
}

// Trait IDs.
const (
	traitDoc        = "smithy.api#documentation"
	traitDeprecated = "smithy.api#deprecated"
	traitRequired   = "smithy.api#required"
	traitSparse     = "smithy.api#sparse"
	traitUnique     = "smithy.api#uniqueItems"
	traitEnumValue  = "smithy.api#enumValue"
	traitTrait      = "smithy.api#trait"
	traitSource     = "tdl#source"
)

// astModel is a Smithy JSON AST model, its shapes in the order the AST
// lists them.
type astModel struct {
	metadata map[string]json.RawMessage
	shapes   []*shape
	// raw is each shape's JSON, by ID.
	raw map[string]json.RawMessage
}

// parseAST reads a Smithy JSON AST model.
func parseAST(data []byte) (*astModel, error) {
	var raw struct {
		Metadata map[string]json.RawMessage `json:"metadata"`
		Shapes   json.RawMessage            `json:"shapes"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	m := &astModel{metadata: raw.Metadata, raw: map[string]json.RawMessage{}}
	err := eachKey(raw.Shapes, func(id string, value json.RawMessage) error {
		m.raw[id] = value
		s, err := parseShape(id, value)
		if err != nil {
			return fmt.Errorf("%s: %v", id, err)
		}
		m.shapes = append(m.shapes, s)
		return nil
	})
	return m, err
}

// eachKey calls fn with each key of a JSON object and its value, in the
// order they are written.
func eachKey(data json.RawMessage, fn func(string, json.RawMessage) error) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return fmt.Errorf("want a JSON object")
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return err
		}
		if err := fn(tok.(string), value); err != nil {
			return err
		}
	}
	return nil
}

type astMember struct {
	Target string                     `json:"target"`
	Traits map[string]json.RawMessage `json:"traits"`
}

func parseShape(id string, data json.RawMessage) (*shape, error) {
	var raw struct {
		Type    string                     `json:"type"`
		Traits  map[string]json.RawMessage `json:"traits"`
		Members json.RawMessage            `json:"members"`
		Member  *astMember                 `json:"member"`
		Key     *astMember                 `json:"key"`
		Value   *astMember                 `json:"value"`
		Mixins  json.RawMessage            `json:"mixins"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	s := &shape{id: id, kind: raw.Type}
	var err error
	if s.traits, err = parseTraits(raw.Traits); err != nil {
		return nil, err
	}
	if len(raw.Mixins) > 0 && string(raw.Mixins) != "[]" {
		s.traits.other = append(s.traits.other, "mixins")
	}
	err = eachKey(raw.Members, func(name string, value json.RawMessage) error {
		var m astMember
		if err := json.Unmarshal(value, &m); err != nil {
			return err
		}
		parsed, err := parseMember(name, &m)
		s.members = append(s.members, parsed)
		return err
	})
	if err != nil {
		return nil, err
	}
	for _, m := range []struct {
		raw *astMember
		out **member
		n   string
	}{{raw.Member, &s.member, "member"}, {raw.Key, &s.key, "key"}, {raw.Value, &s.value, "value"}} {
		if m.raw != nil {
			if *m.out, err = parseMember(m.n, m.raw); err != nil {
				return nil, err
			}
		}
	}
	return s, nil
}

func parseMember(name string, m *astMember) (*member, error) {
	t, err := parseTraits(m.Traits)
	return &member{name: name, target: m.Target, traits: t}, err
}

func parseTraits(raw map[string]json.RawMessage) (traits, error) {
	var t traits
	str := func(data json.RawMessage) (*string, error) {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
		return &s, nil
	}
	var err error
	for id, value := range raw {
		switch id {
		case traitDoc:
			t.doc, err = str(value)
		case traitEnumValue:
			// An intEnum's value is a number, which is not read.
			if t.enumValue, err = str(value); err != nil {
				t.other, err = append(t.other, id), nil
			}
		case traitSource:
			t.source, err = str(value)
		case traitDeprecated:
			var d struct {
				Message string `json:"message"`
			}
			err = json.Unmarshal(value, &d)
			t.deprecated = &deprecation{d.Message}
		case traitRequired:
			t.required = true
		case traitSparse:
			t.sparse = true
		case traitUnique:
			t.unique = true
		case traitTrait:
			t.trait = true
		default:
			t.other = append(t.other, id)
		}
		if err != nil {
			return t, fmt.Errorf("the trait %s: %v", id, err)
		}
	}
	return t, nil
}
