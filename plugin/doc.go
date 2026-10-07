// Package plugin is the wire protocol a TDL backend speaks, and the SDK
// for writing one.
//
// # Writing a backend
//
// A backend implements [Backend]. A plugin's main is [Serve] and nothing
// else:
//
//	func main() {
//		if err := plugin.Serve(myBackend{}); err != nil {
//			fmt.Fprintln(os.Stderr, err)
//			os.Exit(1)
//		}
//	}
//
// Build it as tdl-gen-<name> and put it on PATH.
//
// # What a request contains
//
// The prelude's declarations (string, List, Option, and the rest) are in
// the model untagged; filter by the filename in a declaration's position.
//
// A node carries the directives of every target block, each naming its
// target; keep your own with [Directives]. A directive expanded from a
// class names that class.
//
// `tdl ir --format json` shows what a model contains.
//
// # Returning files
//
// A backend returns contents, and tdl writes them. Paths are relative to
// [Request.Out]; an absolute one, or one climbing out with "..", is
// refused and nothing is written. A problem with the model belongs in
// [Response] diagnostics, not in an error from [Backend.Generate].
//
// # What a plugin will not see
//
// The declaration-level directives of a dependency's target blocks; only
// their block-scope directives arrive, on the dependency's import. And
// class-scoped directives on types that satisfy a class only through a
// conditional instance: a directive on Auditable reaches Audited and not
// the Page<Audited> that satisfies Auditable through an instance.
//
// # Importing
//
// A backend that also reads its target language back into a model, for
// `tdl import`, implements [Importer] and sets Reverse in its description.
// tdl then shakes hands in [Mode_MODE_IMPORT] and sends [ImportRequest]
// messages instead of [Request] ones. A warning about a fact the
// conversion loses carries a code, such as "lossy.collection", which the
// user can silence; docs/design/reverse.md lists them.
//
// # The wire
//
// Messages are protobuf, framed with a varint length prefix, in both
// directions over one connection. tdl sends a [Handshake] first and a
// plugin answers with a [HandshakeReply], refusing a version it does not
// support. [Conn] is the codec.
package plugin
