package haskell_test

import (
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
)

func TestGenerics(t *testing.T) {
	m := lowerSource(t, "shop.tdl", `package shop

type Page<T> {
  items: [T]
  next: string?
}

type Collection<f, T> {
  items: f<T>
}

enum Result<T, E> {
  Ok { value: T }
  Err { error: E }
}

type Bag<T> {
  items: {T}
}

alias Pair<K, V> = {K -> V}
alias Applied<f: type -> type, T: type> = f<T>
alias Parenthesized<f: (type -> type) -> type, g> = f<g>

type Uses {
  list: Collection<List, int>
  maybe: Collection<Option, string>
  partial: Collection<Map<string>, int>
  pair: Pair<string, int>
  applied: Applied<List, bool>
}
`, nil)
	resp := generate(t, m)
	quiet(t, resp)
	src := check(t, resp)
	contains(t, src,
		"import Data.Kind (Type)\n",
		"data Page t = Page\n  { items :: [t]\n  , next :: Maybe Text\n  }\n  deriving stock (Eq, Ord, Show, Generic, Functor, Foldable, Traversable)\n",
		"data Collection f t = Collection\n  { items :: f t\n  }\n  deriving stock (Generic)\n\n"+
			"deriving stock instance (Eq (f t)) => Eq (Collection f t)\n"+
			"deriving stock instance (Ord (f t)) => Ord (Collection f t)\n"+
			"deriving stock instance (Show (f t)) => Show (Collection f t)\n",
		"data Result t e\n  = Ok {value :: t}\n  | Err {error :: e}\n  deriving stock (Eq, Ord, Show, Generic, Functor, Foldable, Traversable)\n",
		"data Bag t = Bag\n  { items :: Set t\n  }\n  deriving stock (Eq, Ord, Show, Generic)\n",
		"type Pair k v = Map k v\n",
		"type Applied (f :: Type -> Type) t = f t\n",
		"type Parenthesized (f :: (Type -> Type) -> Type) g = f g\n",
		"  { list :: Collection [] Int64\n  , maybe :: Collection Maybe Text\n  , partial :: Collection (Map Text) Int64\n  , pair :: Pair Text Int64\n  , applied :: Applied [] Bool\n",
	)
	absent(t, src, "UndecidableInstances")
}

// Functor, Foldable, and Traversable are decided across the model: through
// another declaration, a recursive one, and a map's value but not its key.
func TestFunctorInference(t *testing.T) {
	m := lowerSource(t, "shop.tdl", `package shop

type Wrapper<T> {
  page: Page<T>
  tree: Tree<T>?
}

type Holder<T> {
  bag: Bag<T>
}

type Tree<T> {
  value: T
  children: [Tree<T>]
}

type Page<T> {
  items: [T]
}

type Bag<T> {
  items: {T}
}

type ByKey<K, V> {
  entries: {K -> [V]}
}

type ByValue<K, V> {
  entries: {V -> K}
}

type Id<T>: string

enum Tag<T> {
  Draft
  Placed
}

type Tagged<f, T> {
  tag: f<string>
  value: T
}

type Applies<T, f> {
  items: f<T>
}
`, nil)
	resp := generate(t, m)
	quiet(t, resp)
	src := check(t, resp)
	for _, name := range []string{"Wrapper", "Tree", "Page", "ByKey", "Id", "Tag", "Tagged"} {
		if !strings.Contains(paragraph(src, name), "Functor, Foldable, Traversable)") {
			t.Errorf("%s does not derive Functor:\n%s", name, src)
		}
	}
	for _, name := range []string{"Holder", "Bag", "ByValue", "Applies"} {
		if strings.Contains(paragraph(src, name), "Functor") {
			t.Errorf("%s derives Functor:\n%s", name, src)
		}
	}
}

// paragraph is the declaration of name and its deriving clauses.
func paragraph(src, name string) string {
	for _, p := range strings.Split(src, "\n\n") {
		if strings.HasPrefix(p, "data "+name+" ") || strings.HasPrefix(p, "newtype "+name+" ") {
			return p
		}
	}
	return ""
}

// A declaration handing its applied parameter to another needs the context
// written out, and one larger than its instance head needs
// UndecidableInstances.
func TestStandaloneDeriving(t *testing.T) {
	m := lowerSource(t, "shop.tdl", `package shop

type Collection<f, T> {
  items: f<T>
}

type Outer<g, A> {
  inner: Collection<g, A>
  count: int
}

type Wrapped<f, T>: f<T>

type Deep<f, T> {
  items: f<[T]>
}

type Kinded<f: type -> type> {
  ints: f<int>
}

target haskell for shop {
  derive("Control.DeepSeq", "NFData")
}
`, nil)
	resp := generate(t, m)
	quiet(t, resp)
	src := check(t, resp)
	contains(t, src,
		"{-# LANGUAGE UndecidableInstances #-}\n",
		"data Outer g a = Outer\n  { inner :: Collection g a\n  , count :: Int64\n  }\n  deriving stock (Generic)\n\n"+
			"deriving stock instance (Eq (g a)) => Eq (Outer g a)\n",
		"deriving anyclass instance (NFData (g a)) => NFData (Outer g a)\n",
		"newtype Wrapped f t = Wrapped (f t)\n  deriving stock (Generic)\n\n",
		"deriving stock instance (Show (f [t])) => Show (Deep f t)\n",
		"data Kinded (f :: Type -> Type) = Kinded\n",
		"deriving stock instance (Ord (f Int64)) => Ord (Kinded f)\n",
	)
}

func TestParametersThatCannotBeGenerated(t *testing.T) {
	m := lowerSource(t, "shop.tdl", `package shop

class Container<f: type -> type> { }

type Box<f, T> requires Container<f> {
  items: f<T>
}

type Keyword<Data> {
  value: Data
}

type Clash<T, t> {
  left: T
  right: t
}

alias Measured<T, u: unit> = T<u>

type Uses {
  clash: Clash<int, int>
}

type Doc<T>: Entity {
  id: string
  revision: int
  body: T
}

type Draft<T>: Entity {
  id: string
  body: T
}

target haskell for shop {
  Doc => key(id, revision)
  Draft => key(id)
}
`, nil)
	resp := generate(t, m)
	for _, want := range []struct{ code, msg string }{
		{emit.LossGeneric, "type parameter Data of Keyword would be the type variable data, which is a Haskell keyword"},
		{emit.LossGeneric, "type parameters T and t of Clash are both the type variable t"},
		{emit.LossUnit, "type parameter u of Measured is a unit"},
		{"", "Uses names Clash"},
		{emit.LossClass, "Box requires Container f of its parameters"},
		{emit.LossKey, "Doc is parameterized, and a key record of several fields is not generated"},
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
	absent(t, src, "Keyword", "Clash", "Measured", "Uses", "DocKey")
	contains(t, src,
		"data Box f t = Box\n",
		"data Doc t = Doc\n",
		"draftKey :: Draft t -> Text\ndraftKey d = d.id\n",
	)
}
