package nix_test

import "testing"

const bags = `package shop

type Bag {
  tags: {string}
  byStatus: {Status -> int}
  byName: {string -> float64}
  byDay: {date -> bool}
}

type Bad {
  byCount: {int -> string}
}

enum Status { Draft Placed }
`

func TestCollections(t *testing.T) {
	resp, out := gen(t, bags)
	contains(t, out,
		"tags = mkOption { type = set types.str; };",
		"byStatus = mkOption { type = keyed Status types.int; };",
		"byName = mkOption { type = types.attrsOf types.number; };",
		`byDay = mkOption { type = keyed (types.strMatching "[0-9]{4}-[0-9]{2}-[0-9]{2}") types.bool; };`,
	)
	warns(t, resp, [2]string{"lossy.collection", "attribute name"})
}

func TestCollectionsEvaluate(t *testing.T) {
	_, out := gen(t, bags)
	got := evaluate(t, out, `let
  good = { tags = [ "a" ]; byStatus.Draft = 1; byName.x = 1.5; byDay."2026-01-02" = true; };
in [
  (eval m.Bag good)
  (ok m.Bag (good // { tags = [ "a" "a" ]; }))
  (ok m.Bag (good // { byStatus.Nope = 1; }))
  (ok m.Bag (good // { byDay.soon = true; }))
]`)
	want := `["{\"byDay\":{\"2026-01-02\":true},\"byName\":{\"x\":1.5},\"byStatus\":{\"Draft\":1},\"tags\":[\"a\"]}",false,false,false]`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
