# The host keeps a nondeterministic backend's output while its request is unchanged

A delegated backend (#1505) hands the mapping to a language model, so the same request can come back as different files every time.
`tdl gen --verify`, the flake module's `tdl-gen` check, and CI all assume that a request produces one set of bytes, and they run without a network or a key.
We decided that a backend may declare itself nondeterministic in its handshake (`Features.nondeterministic`), and that `tdl`, not the backend, then keeps what it wrote.
It records a hash of each request in `.tdl-output`, together with how many files it wrote, a hash of each, and the diagnostics the backend reported.
While a later request hashes the same and those files are all there and unchanged, it keeps them, reports the recorded diagnostics again so a lossy generation stays visible, and sends no request.
So regenerating is a deliberate act, and `--verify` means "the request and the files are what was generated".
A plugin still never reads the disk, and any later nondeterministic backend gets the same behaviour.

## Considered Options

- A lock the plugin owns, `<out>/yolo.lock`, read from `Request.out`: rejected, because it breaks the rule that a plugin returns files and `tdl` owns writing, verifying, and cleaning, and because every such backend would rebuild it.
- Declaring the backend unverifiable, so `--verify` skips it: rejected, because then a stale output never fails a check.

## Consequences

- The hash covers everything the request carries except what varies with where `tdl` runs: positions, the output path, and the dry-run flag. A change to a directive, to a model name, or to a declaration regenerates, while moving the model within its file does not.
- `--verify` never sends a nondeterministic backend's request. A changed request, a target never generated, a record that lost file lines, and a kept file edited or removed are stale without asking the backend. `--clean` always regenerates.
- The handshake still runs, because building a request needs the backend's directives.
- The record lives in `.tdl-output` as `#@` comment lines, which older readers of the marker skip.
