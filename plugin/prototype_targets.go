package plugin

// PROTOTYPE for #1398 (wayfinder map #1394): one plugin serving several
// backends. Throwaway; it lives on a branch and never merges. The real
// change would fold into serve.go.
//
// Shape under test: Serve stays the one entry point and becomes variadic,
// so every existing main still compiles, and a shared plugin's main is
//
//	plugin.Serve(csharp.Backend{}, fsharp.Backend{})
//
// Each backend is keyed by Describe().Name, which #1396 made the target it
// serves, so there is no second map from target to backend to keep in step.
// Dispatch happens once, at the handshake; after that the connection is
// exactly today's single-backend ServeConn.

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

// choose picks the backend a connection serves. requested is
// Handshake.target (empty from a host that predates it) and argv0 is
// os.Args[0]. The rules are #1396's:
//
//   - the handshake's target wins;
//   - with none, the executable's name with CommandPrefix stripped;
//   - a plugin serving one backend keeps today's behaviour and serves it
//     whatever it was asked for (open question Q2);
//   - anything else is refused, naming what was asked for and what is
//     served.
func choose(requested, argv0 string, bs []Backend) (Backend, string, bool) {
	if len(bs) == 1 && requested == "" {
		return bs[0], "", true
	}
	want := requested
	if want == "" {
		want = strings.TrimPrefix(filepath.Base(argv0), CommandPrefix)
	}
	var served []string
	for _, b := range bs {
		name := b.Describe().Name
		if name == want {
			return b, "", true
		}
		served = append(served, name)
	}
	slices.Sort(served)
	return nil, fmt.Sprintf("serves %s; asked for %s", strings.Join(served, ", "), want), false
}

// CommandPrefix mirrors gen.CommandPrefix; the plugin package cannot import
// internal/gen.
const CommandPrefix = "tdl-gen-"
