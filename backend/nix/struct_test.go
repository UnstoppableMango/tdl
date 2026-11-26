package nix_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const cart = `package shop

/// What a cart holds.
type Cart {
  /// The total.
  total: int32
  count: uint64
  note: string?
  coupon: string | null
  both: string? | null
  skus: [string]
  sparse: [int?]
  id: uuid
  deprecated("gone") fax: string
  status: Status
  took: duration
  include Stamped
}

mixin Stamped {
  at: instant
}

enum Status { Draft Placed }

target nix for shop {
  Cart.coupon => wire("code")
}
`

func TestStruct(t *testing.T) {
	resp, out := gen(t, cart)
	contains(t, out,
		"# What a cart holds.\n  Cart = struct {",
		`total = mkOption { type = types.ints.s32; description = "The total."; };`,
		"count = mkOption { type = types.ints.unsigned; };",
		"note = optional { type = types.str; };",
		"code = mkOption { type = nullable types.str; };",
		"both = optional { type = types.str; };",
		"skus = mkOption { type = types.listOf types.str; };",
		"sparse = mkOption { type = types.listOf (types.nullOr types.int); };",
		`id = mkOption { type = types.strMatching "[0-9a-fA-F]{8}-`,
		`description = "Deprecated: gone";`,
		"status = mkOption { type = Status; };",
		`at = mkOption { type = types.strMatching "[0-9]{4}-`,
		"Stamped = struct {",
	)
	warns(t, resp, [2]string{"lossy.primitive", "uint64"}, [2]string{"lossy.optional", "both"})
}

func TestStructSkips(t *testing.T) {
	resp, out := gen(t, `package shop
unit kg
type Id: string
type Uses { id: Id }
type Kept { x: int = 1 }
type Parcel { weight: decimal<kg> where { min(0) } }
`)
	if strings.Contains(out, "Uses") {
		t.Errorf("output has Uses:\n%s", out)
	}
	warns(t, resp, [2]string{"", "Uses names Id"}, [2]string{"lossy.default", "Kept.x"},
		[2]string{"lossy.unit", "Parcel.weight"}, [2]string{"lossy.constraint", "Parcel.weight"})
}

// evaluate evaluates checks, a Nix list, against the file out holds as m,
// with eval of a type and a value toJSON of what lib.evalModules merges
// and ok whether that succeeds, and returns the list as JSON.
func evaluate(t *testing.T, out, checks string) string {
	t.Helper()
	if testing.Short() {
		t.Skip("short")
	}
	if _, err := exec.LookPath("nix-instantiate"); err != nil {
		t.Skip("nix-instantiate is not on PATH")
	}
	if err := exec.Command("nix-instantiate", "--eval", "-E", "<nixpkgs/lib>").Run(); err != nil {
		t.Skip("<nixpkgs> is not on NIX_PATH")
	}
	path := filepath.Join(t.TempDir(), "model.nix")
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	expr := `let
  lib = import <nixpkgs/lib>;
  m = import ` + path + ` { inherit lib; };
  eval = type: v: builtins.toJSON (lib.evalModules { modules = [ { options.x = lib.mkOption { inherit type; }; config.x = v; } ]; }).config.x;
  ok = type: v: (builtins.tryEval (eval type v)).success;
in ` + checks
	b, err := exec.Command("nix-instantiate", "--eval", "--strict", "--json", "-E", expr).CombinedOutput()
	if err != nil {
		t.Fatalf("nix-instantiate: %v\n%s", err, b)
	}
	return strings.TrimSpace(string(b))
}

func TestStructEvaluates(t *testing.T) {
	_, out := gen(t, cart)
	got := evaluate(t, out, `let
  good = { total = 1; count = 2; code = null; both = "b"; skus = [ "a" ]; sparse = [ 1 null ]; id = "00000000-0000-0000-0000-000000000000"; fax = "f"; status = "Draft"; took = "PT1H"; at = "2026-01-02T03:04:05Z"; };
in [
  (eval m.Cart good)
  (ok m.Cart (good // { total = 3000000000; }))
  (ok m.Cart (good // { extra = 1; }))
  (ok m.Cart (good // { status = "Nope"; }))
  (ok m.Cart (good // { id = "nope"; }))
  (ok m.Cart (removeAttrs good [ "code" ]))
  (ok m.Cart (removeAttrs good [ "total" ]))
  (eval m.Cart (good // { note = "n"; }) != eval m.Cart good)
  (ok m.Cart (good // { at = "2026-13-02T03:04:05Z"; }))
  (ok m.Cart (good // { took = "PT"; }))
  (ok m.Cart (good // { took = "P"; }))
]`)
	want := `["{\"at\":\"2026-01-02T03:04:05Z\",\"both\":\"b\",\"code\":null,\"count\":2,\"fax\":\"f\",\"id\":\"00000000-0000-0000-0000-000000000000\",\"skus\":[\"a\"],\"sparse\":[1,null],\"status\":\"Draft\",\"took\":\"PT1H\",\"total\":1}",false,false,false,false,false,false,true,false,false,false]`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestRecursiveEvaluates(t *testing.T) {
	_, out := gen(t, "package shop\ntype Node {\n  children: [Node]\n  next: Node?\n}\n")
	contains(t, out, "children = mkOption { type = types.listOf (opaque Node); };")
	got := evaluate(t, out, `[
  (eval m.Node { children = [ { children = [ ]; } ]; })
  (builtins.length (lib.optionAttrSetToDocList (lib.evalModules { modules = [ { options.x = lib.mkOption { type = m.Node; }; } ]; }).options) > 0)
]`)
	if want := `["{\"children\":[{\"children\":[]}]}",true]`; got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
