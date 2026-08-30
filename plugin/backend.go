package plugin

import (
	"context"

	"github.com/unstoppablemango/tdl/ir"
)

// Backend turns a resolved model into files.
//
// This is the whole surface. A backend compiled into tdl implements it and
// is called in process; a backend shipped as tdl-gen-<name> implements it
// and is served over a connection by [Serve]. There is no richer
// interface available to the first kind, because a surface only one of the
// two can reach is one nothing keeps honest.
type Backend interface {
	// Describe reports what this backend is and what it understands. tdl
	// sends it in the handshake reply, and checks a target block against
	// the directives before generating anything.
	Describe() Description

	// Generate returns the files for one request. It returns an error only
	// when it cannot produce a response at all; a problem with the model
	// belongs in Response.diagnostics, where it reaches the user with a
	// position attached.
	Generate(ctx context.Context, req *Request) (*Response, error)
}

// Description is what a backend says about itself.
type Description struct {
	Name       string
	Version    string
	Directives []*DirectiveSpec
	Reuse      bool
}

// Reply turns a description into the handshake reply that carries it.
func (d Description) Reply() *HandshakeReply {
	return &HandshakeReply{
		Accepted:   true,
		Name:       d.Name,
		Version:    d.Version,
		Directives: d.Directives,
		Features:   &Features{Reuse: d.Reuse},
	}
}

// Directives returns the directives on a node that belong to the target
// being served.
//
// A model carries directives for every target block in it, each tagged
// with the block it came from, so a backend filters rather than assuming
// what it is handed is its own.
func Directives(target string, all []*ir.Directive) []*ir.Directive {
	var mine []*ir.Directive
	for _, d := range all {
		if d.GetTarget() == target {
			mine = append(mine, d)
		}
	}
	return mine
}
