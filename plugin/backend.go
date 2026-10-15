package plugin

import (
	"context"

	"github.com/unstoppablemango/tdl/ir"
)

// Backend turns a resolved model into files, in process or served as a
// plugin by [Serve].
type Backend interface {
	// Describe reports what this backend is and what directives it
	// understands. tdl checks a target block against them before
	// generating.
	Describe() Description

	// Generate returns the files for one request. It returns an error only
	// when it cannot produce a response at all; a problem with the model
	// belongs in Response.diagnostics.
	Generate(ctx context.Context, req *Request) (*Response, error)
}

// Importer is a backend that also reads its target language back into a
// model, for `tdl import`. A backend implementing it declares Reverse in
// its description.
type Importer interface {
	Backend

	// Import returns the model the request's files describe. It returns an
	// error only when it cannot produce a response at all; a problem with
	// the files belongs in ImportResponse.diagnostics.
	Import(ctx context.Context, req *ImportRequest) (*ImportResponse, error)
}

// Description is what a backend says about itself.
type Description struct {
	Name       string
	Version    string
	Directives []*DirectiveSpec
	Reuse      bool

	// Reverse says the backend is an [Importer].
	Reverse bool

	// Numbering is how the backend numbers fields and variants, or nil
	// for a backend that does not.
	Numbering *Numbering
}

// Reply turns a description into the handshake reply that carries it.
func (d Description) Reply() *HandshakeReply {
	return &HandshakeReply{
		Accepted:   true,
		Name:       d.Name,
		Version:    d.Version,
		Directives: d.Directives,
		Features:   &Features{Reuse: d.Reuse, Reverse: d.Reverse},
		Numbering:  d.Numbering,
	}
}

// Directives returns the directives in all that belong to target. A request
// from tdl carries only its target's directives, so this matters for a
// model built some other way, where a node carries every target block's.
func Directives(target string, all []*ir.Directive) []*ir.Directive {
	var mine []*ir.Directive
	for _, d := range all {
		if d.GetTarget() == target {
			mine = append(mine, d)
		}
	}
	return mine
}
