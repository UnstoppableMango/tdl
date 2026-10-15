package haskell_test

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/backend/internal/irtest"
)

func TestClasses(t *testing.T) {
	m := lowerSource(t, "shop.tdl", `package shop

/// Has a creation time.
class Timestamped {
  createdAt: instant
}

class Auditable: Timestamped {
  updatedAt: instant
}

class Paged {
  type Cursor
  pageSize: int
}

class Container<f: type -> type> { }

class Projection<from, to> | from -> to { }

class Archived { }

class Keyed<T> requires Archived<T> { }

type Order: Auditable, Archived, Keyed {
  createdAt: instant
  updatedAt: instant
}

type OrderSummary { total: int }

instance Projection<Order, OrderSummary>

instance Container<List>

instance Paged for OrderList {
  type Cursor = OrderCursor
}

type OrderList {
  pageSize: int
}

type OrderCursor {
  token: string
}

type Page<T> {
  items: [T]
}

instance <T> Archived<Page<T>> requires Archived<T>

enum Status: Archived {
  Draft
  Placed
}
`, nil)
	resp := generate(t, m)
	quiet(t, resp)
	src := check(t, resp)
	contains(t, src,
		"{-# LANGUAGE DataKinds #-}\n",
		"{-# LANGUAGE FunctionalDependencies #-}\n",
		"{-# LANGUAGE TypeFamilies #-}\n",
		"import GHC.Records (HasField)\n",
		"-- | Has a creation time.\nclass (HasField \"createdAt\" a UTCTime) => Timestamped a\n",
		"class (Timestamped a, HasField \"updatedAt\" a UTCTime) => Auditable a\n",
		"class (HasField \"pageSize\" a Int64) => Paged a where\n  type Cursor a\n",
		"class Container (f :: Type -> Type)\n",
		"class Projection from to | from -> to\n",
		"class (Archived t) => Keyed t\n",
		"instance Auditable Order\n",
		"instance Timestamped Order\n",
		"instance Archived Order\n",
		"instance Keyed Order\n",
		"instance Projection Order OrderSummary\n",
		"instance Container []\n",
		"instance Paged OrderList where\n  type Cursor OrderList = OrderCursor\n",
		"instance (Archived t) => Archived (Page t)\n",
		"instance Archived Status\n",
	)
	absent(t, src, "UndecidableInstances", "UndecidableSuperClasses")
	if strings.Count(src, "instance Timestamped Order") != 1 {
		t.Errorf("Timestamped Order is declared more or less than once:\n%s", src)
	}
}

// A class reaching itself through what it requires generates under
// UndecidableSuperClasses, and a class requiring one implies its instance
// for a conditional instance too, under the same context.
func TestClassesHaskellExpresses(t *testing.T) {
	m := lowerSource(t, "shop.tdl", `package shop

class Ping: Pong { }
class Pong: Ping { }

class Stored { }
class Archived: Stored { }

type Page<T> {
  items: [T]
}

instance <T> Archived<Page<T>> requires Archived<T>

type Order: Ping { id: string }

type Envelope<T> requires Archived<T> {
  body: T
}

type Doc<T>: Entity requires Archived<T> {
  id: string
  body: T
}

type Listed<T>: Entity requires Archived<[T]> {
  id: string
  body: T
}

target haskell for shop {
  Doc => key(id)
  Listed => key(id)
}
`, nil)
	resp := generate(t, m)
	src := check(t, resp)
	contains(t, src,
		"{-# LANGUAGE UndecidableSuperClasses #-}\n",
		"class (Pong a) => Ping a\n",
		"instance Ping Order\n",
		"instance Pong Order\n",
		"instance (Archived t) => Archived (Page t)\n",
		"instance (Archived t) => Stored (Page t)\n",
		"docKey :: (Archived t) => Doc t -> Text\n",
		"listedKey :: (Archived [t]) => Listed t -> Text\n",
	)
	if !warned(resp, "Envelope requires Archived t of its parameters, which a Haskell type cannot carry") {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
}

func TestClassesThatCannotBeGenerated(t *testing.T) {
	m := lowerSource(t, "shop.tdl", `package shop

class Auditable requires Entity {
  createdAt: instant
}

class Paged {
  type Cursor
}

class Labelled {
  label: string
}

class Tagged<T> requires Labelled<T> { }

type Order: Entity, Auditable, Paged, Tagged {
  id: string
  createdAt: instant
}

type Item: Labelled {
  label: string
}

type Box<T> requires Entity<T> {
  value: T
}

target haskell for shop {
  Item.label => name("title")
}
`, nil)
	resp := generate(t, m)
	for _, want := range []struct{ code, msg string }{
		{emit.LossClass, "Auditable is generated without a constraint: Entity is the prelude's"},
		{emit.LossClass, "Box is generated without a constraint: Entity is the prelude's"},
		{emit.LossClass, "Order's conformance to Paged binds no type for Paged.Cursor"},
		{emit.LossClass, "Order's conformance to Tagged needs the instance Labelled Order"},
		{emit.LossClass, "Item's conformance to Labelled needs Item.label labelled label"},
	} {
		found := false
		for _, d := range resp.GetDiagnostics() {
			found = found || (d.GetCode() == want.code && strings.Contains(d.GetMessage(), want.msg))
		}
		if !found {
			t.Errorf("no %q warning %q: %+v", want.code, want.msg, resp.GetDiagnostics())
		}
	}
	src := check(t, resp)
	contains(t, src,
		"class (HasField \"createdAt\" a UTCTime) => Auditable a\n",
		"instance Auditable Order\n",
	)
	absent(t, src, "instance Paged", "instance Tagged", "instance Labelled", "Entity")
}

// An instance for a dependency's type imports its fields, which HasField
// needs in scope.
func TestInstanceForADependency(t *testing.T) {
	d := deps{"shipping.tdl": `package shop.shipping

type Address {
  line1: string
  createdAt: instant
}

target haskell for shop.shipping {
  module("Shop.Shipping")
}
`}
	root := lowerSource(t, "shop.tdl", `package shop

import "shipping.tdl" as shipping

class Timestamped {
  createdAt: instant
}

class Auditable: Timestamped { }

instance Auditable<shipping.Address>
instance Auditable for shipping.Address
`, d)
	resp := generate(t, root)
	quiet(t, resp)
	src := string(resp.GetFiles()[0].GetContent())
	contains(t, src,
		"import Shop.Shipping (Address (..))\n",
		"instance Auditable Address\n",
		"instance Timestamped Address\n",
	)
	if strings.Count(src, "instance Auditable Address") != 1 {
		t.Errorf("the duplicate instance is written:\n%s", src)
	}
	dep := generate(t, lowerSource(t, "shipping.tdl", d["shipping.tdl"], nil))
	compileTogether(t, resp, dep)
}

// Every class in the conformance corpus generates and compiles, an
// associated type binding included. What warns is a constraint on a type's
// parameters, and the instance for a type whose package has no haskell
// target block.
func TestConformanceClasses(t *testing.T) {
	for _, dir := range []string{"classes", "instances", "comments", "mixin_include", "targets"} {
		t.Run(dir, func(t *testing.T) {
			m := irtest.Conformance(t, "../../testdata/conformance/"+dir)
			resp := generate(t, m)
			for _, d := range resp.GetDiagnostics() {
				ok := d.GetCode() == emit.LossConstraint ||
					d.GetCode() == emit.LossClass && strings.Contains(d.GetMessage(), "of its parameters") ||
					strings.Contains(d.GetMessage(), "shop.shipping.Address is declared in another package")
				if !ok {
					t.Errorf("diagnostic: %+v", d)
				}
			}
			src := check(t, resp)
			if dir == "instances" {
				contains(t, src, "  type Cursor OrderList = OrderCursor\n", "instance (Archived t) => Archived (Page t)\n")
			}
		})
	}
}

// An instance for a prelude type is one for the Haskell type it maps to,
// written once when two map to one; a parameterized class's fields are
// required of its parameter; a conformance supplies a class's first
// argument; and an associated type keeps its kind.
func TestInstanceShapes(t *testing.T) {
	m := lowerSource(t, "shop.tdl", `package shop

class Archived { }
class Sized<T> { size: int }
class Projection<from, to> | from -> to { }

/// Holds things.
deprecated("use Archived")
class Holder {
  /// The element.
  type Elem: type -> type
  type Key
}

instance Archived<string>
instance Archived<uuid>
instance <T> Archived<Option<T>> requires Archived<T>
instance <T> Archived<[T]> requires Archived<T>
instance <K, V> Archived<{K -> V}> requires Archived<K>, Archived<V>
instance Sized<Bag>
instance Holder for Bag {
  type Elem = List
  type Key = string
}

type Bag { size: int }
type Email: string
instance Archived for Email
type Sum { x: int }
type Order: Projection<Sum> { x: int }
`, nil)
	resp := generate(t, m)
	quiet(t, resp)
	src := check(t, resp)
	contains(t, src,
		"class (HasField \"size\" t Int64) => Sized t\n",
		"-- | Holds things.\nclass Holder a where\n  -- | The element.\n  type Elem a :: Type -> Type\n  type Key a\n\n{-# DEPRECATED Holder \"use Archived\" #-}\n",
		"instance Projection Order Sum\n",
		"instance Archived Text\n",
		"instance (Archived t) => Archived (Maybe t)\n",
		"instance (Archived t) => Archived [t]\n",
		"instance (Archived k, Archived v) => Archived (Map k v)\n",
		"instance Sized Bag\n",
		"instance Archived Email\n",
		"instance Holder Bag where\n  type Elem Bag = []\n  type Key Bag = Text\n",
	)
	if strings.Count(src, "instance Archived Text") != 1 {
		t.Errorf("Archived Text is declared more or less than once:\n%s", src)
	}
}

// An instance for an alias and one for its target are one instance, as
// GHC sees them, and an instance whose class requires an instance the
// backend cannot name is dropped.
func TestInstancesGHCWouldRefuse(t *testing.T) {
	m := lowerSource(t, "shop.tdl", `package shop

class Archived { }

class Listed<T> requires Archived<[T]> { }

alias Tag = string

instance Archived<Tag>
instance Archived<string>

type Order: Listed { id: string }
`, nil)
	resp := generate(t, m)
	if !warned(resp, "Order's conformance to Listed needs an instance this module cannot name") {
		t.Errorf("diagnostics = %+v", resp.GetDiagnostics())
	}
	src := check(t, resp)
	contains(t, src, "instance Archived Tag\n", "class (Archived [t]) => Listed t\n")
	absent(t, src, "instance Archived Text", "instance Listed")
}
