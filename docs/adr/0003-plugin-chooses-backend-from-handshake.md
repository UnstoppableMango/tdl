# A plugin chooses its backend from the handshake's target

One plugin executable may serve several backends (#1522), and a plugin's handshake reply describes exactly one backend: its directives, features, and numbering.
So the plugin has to know which target the host wants before it replies.
We put the target in the protocol: `Handshake.target`, field 5, beside the `Request.target` that already names the block.
A plugin serving several backends keys them by `Describe().Name`, which is the target it serves, and answers for that one alone.
The host still starts each target as `tdl-gen-<target>`, typically a link to the shared executable, or through a `[plugins]` entry in `tdl.toml`.
A connection serves one target.

## Considered Options

- **The executable's name alone, as protoc plugins and busybox do.** No protocol change, but a `[plugins]` entry could not point a target at an executable named anything else, and `plugins.md` keeps everything a backend reads in versioned messages rather than in how it was started.
- **One executable per target, with wrappers passing the target as an argument.** Nothing in a request travels in argv, so this would open a second channel to keep compatible.
- **The target only in `Request`.** It is there already, but it arrives after the handshake reply, which is where the backend-specific description lives.

## Consequences

- From an older host, which sends no target, a plugin serving several backends falls back to `basename(argv[0])` with `tdl-gen-` stripped. It refuses only when started under the shared executable's own name. When the field and `argv[0]` disagree, the field wins.
- An unserved target is refused with a `refusal` naming the requested and the served targets, and the host relays that text instead of wording every refusal as a version mismatch.
- `HandshakeReply.name` means the target served and `version` the executable's version. The host does not check `name` against the target, which keeps the trust model for anything on `PATH`.
- A plugin serving one backend still serves it when no target is named, and refuses a different named target, so a mistyped `[plugins]` entry is caught.
- Decided through the wayfinder map #1394; the detail is in the resolutions of #1396 and #1398.
