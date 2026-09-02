package ir

import "testing"

// The unit renderers are the part of a dump the conformance goldens cannot
// reach. A golden shows what lowering produces, and lowering never produces
// an unresolved reference, a dimensionless quantity, or a base with no name.
// Those branches exist for a model built by hand, which `ir` being public
// API means anyone may do.

func dims(pairs ...any) *Unit {
	u := &Unit{}
	for i := 0; i < len(pairs); i += 2 {
		u.Dims = append(u.Dims, &Dimension{
			Base:     &ID{Index: int32(i / 2), Name: pairs[i].(string)},
			Exponent: int32(pairs[i+1].(int)),
		})
	}
	return u
}

func TestDimsText(t *testing.T) {
	cases := []struct {
		name string
		unit *Unit
		want string
	}{
		{"no unit", nil, "?"},
		{"dimensionless", &Unit{}, "1"},
		{"one base", dims("kg", 1), "kg"},
		{"product", dims("kg", 1, "m", 1), "kg*m"},
		{"exponent", dims("m", 3), "m^3"},
		{"quotient", dims("kg", 1, "m", -1, "s", -2), "kg/m/s^2"},
		{"no numerator", dims("s", -1), "1/s"},
		{
			// An ID carries a name so there is something to print. `?^2`
			// reads as a gap where a bare `^2` reads as a broken renderer.
			"base with no name",
			&Unit{Dims: []*Dimension{{Base: &ID{Index: 0}, Exponent: 2}}},
			"?^2",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := dimsText(c.unit); got != c.want {
				t.Errorf("dimsText = %q, want %q", got, c.want)
			}
		})
	}
}

func TestModelUnit(t *testing.T) {
	m := &Model{Units: []*Unit{dims("kg", 1)}}

	if got := m.Unit(&ID{Index: 0}); got == nil {
		t.Error("Unit(0) = nil, want the entry")
	}
	if got := m.Unit(&ID{Index: Unresolved}); got != nil {
		t.Errorf("Unit(unresolved) = %v, want nil", got)
	}
	// An index past the table is a model that was built wrong rather than a
	// lookup that should panic.
	if got := m.Unit(&ID{Index: 7}); got != nil {
		t.Errorf("Unit(7) = %v, want nil", got)
	}
}

func TestUnitRef(t *testing.T) {
	d := &dumper{model: &Model{Units: []*Unit{dims("kg", 1, "s", -2)}}}

	if got := d.unitRef(&ID{Index: 0}); got != "units[0] kg/s^2" {
		t.Errorf("unitRef = %q", got)
	}
	if got := d.unitRef(&ID{Index: Unresolved}); got != "?" {
		t.Errorf("unitRef(unresolved) = %q, want ?", got)
	}
}
