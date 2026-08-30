// Package plugin is the wire protocol a TDL backend speaks, and the SDK
// for writing one.
//
// # Writing a backend
//
// A backend implements [Backend]: [Backend.Describe] says what it is and
// what directives it understands, and [Backend.Generate] turns a request
// into files. A plugin's main is [Serve] and nothing else:
//
//	func main() {
//		if err := plugin.Serve(myBackend{}); err != nil {
//			fmt.Fprintln(os.Stderr, err)
//			os.Exit(1)
//		}
//	}
//
// Build it as tdl-gen-<name> and put it on PATH. A target block naming
// that backend will find it.
//
// # What a request contains
//
// Three things about the model surprise people, and all three come from
// how the compiler resolves rather than from this protocol.
//
// The prelude is in it. A model whose source declares two things arrives
// with twenty-one declarations, nineteen of them string, List, Option,
// and the rest of the standard prelude, merged in untagged. That is what
// lets a replacement prelude change what a collection is without any
// backend learning about it, and it means a backend emitting one file per
// declaration will emit nineteen nobody asked for. Filter by the filename
// in each declaration's position.
//
// Directives are tagged, not filtered. A node carries the directives of
// every target block in the model, each naming the block it came from, so
// a backend keeps its own with [Directives]. Reading them unfiltered means
// acting on another backend's instructions.
//
// A directive expanded from a class names the class it came from, so a
// backend can say why a rule applies rather than only that it does.
//
// Run `tdl ir --format json` over a model to see all of this before
// writing code against it.
//
// # Returning files
//
// A backend returns contents, and tdl writes them. That is what lets tdl
// enforce path confinement, --verify, and --clean rather than asking every
// backend to honour them. Paths are relative to [Request.Out]; an absolute
// one, or one climbing out with "..", is refused and nothing is written.
//
// A problem with the model belongs in [Response] diagnostics, where it
// reaches the user with a position attached. Returning an error from
// [Backend.Generate] means the backend could not produce a response at
// all.
//
// # What a plugin will not see
//
// Units, because ir defers them and a model using one does not lower. A
// dependency's target blocks, because merging them needs the dependency
// lowered and nothing does that yet. And class-scoped directives on types
// that satisfy a class only through a conditional instance: a directive on
// Auditable reaches Audited and not the Page[Audited] that satisfies
// Auditable through an instance.
//
// # The wire
//
// Messages are protobuf, framed with a varint length prefix, in both
// directions over one connection. tdl sends a [Handshake] first and a
// plugin answers with a [HandshakeReply], accepting or refusing with the
// version it needed. Refusing is the point: a plugin that silently ignored
// fields it was compiled before would emit subtly wrong code with no
// diagnostic anywhere.
//
// [Conn] is the codec, for anyone implementing the protocol in another
// language or embedding it somewhere [Serve] does not fit.
package plugin
