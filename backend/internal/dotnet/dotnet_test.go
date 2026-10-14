package dotnet_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/dotnet"
)

func TestPrimitive(t *testing.T) {
	for name, want := range map[string]dotnet.Type{
		// The spec puts no width on int.
		"int":      {Namespace: "System", Name: "Int64"},
		"int32":    {Namespace: "System", Name: "Int32"},
		"uint64":   {Namespace: "System", Name: "UInt64"},
		"float32":  {Namespace: "System", Name: "Single"},
		"bytes":    {Namespace: "System", Name: "Byte", Array: true},
		"decimal":  {Namespace: "System", Name: "Decimal"},
		"uuid":     {Namespace: "System", Name: "Guid"},
		"instant":  {Namespace: "System", Name: "DateTimeOffset"},
		"date":     {Namespace: "System", Name: "DateOnly"},
		"duration": {Namespace: "System", Name: "TimeSpan"},
	} {
		if got, ok := dotnet.Primitive(name); !ok || got != want {
			t.Errorf("Primitive(%q) = %+v, %v; want %+v", name, got, ok, want)
		}
	}
	if _, ok := dotnet.Primitive("money"); ok {
		t.Error("an unknown primitive maps to a type")
	}
}

func TestNamespace(t *testing.T) {
	for pkg, want := range map[string]string{
		"shop":         "Shop",
		"shop.billing": "Shop.Billing",
		"acme.v1":      "Acme.V1",
		"ACME.api":     "ACME.Api",
	} {
		if got := dotnet.Namespace(pkg); got != want {
			t.Errorf("Namespace(%q) = %q, want %q", pkg, got, want)
		}
	}
}

func TestParse(t *testing.T) {
	got, err := dotnet.Parse([]string{"System.Runtime.Serialization", "DataContract"})
	if want := (dotnet.Qualified{Namespace: "System.Runtime.Serialization", Name: "DataContract"}); err != nil || got != want {
		t.Errorf("Parse = %+v, %v; want %+v", got, err, want)
	}
	for _, args := range [][]string{
		{"NodaTime"},
		{"NodaTime", "Instant", "Extra"},
		{"", "Instant"},
		{"Noda..Time", "Instant"},
		{"Noda Time", "Instant"},
		{"1Noda", "Instant"},
		{"NodaTime", "Instant<T>"},
		{"NodaTime", "Noda.Instant"},
		{"NodaTime", ""},
	} {
		if _, err := dotnet.Parse(args); err == nil {
			t.Errorf("Parse(%q) accepts it", args)
		}
	}
}
