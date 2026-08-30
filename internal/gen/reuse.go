package gen

import (
	"context"
	"os"
	"time"

	"github.com/unstoppablemango/tdl/plugin"
)

// Session is a plugin kept alive across generations.
//
// A plugin that declared reuse serves more than one request on one
// connection, which is what makes a watch loop cheap. It must treat each
// request as independent; carrying state between them is a plugin bug, and
// nothing here can catch it.
type Session struct {
	sub      *Subprocess
	live     *session
	modTime  time.Time
	restarts int
}

// Open starts a plugin and holds the connection if it declared reuse.
//
// A plugin that did not gets a fresh process per generation, which is the
// default: reuse is something a backend opts into by saying it can.
func Open(ctx context.Context, sub *Subprocess) (*Session, error) {
	s := &Session{sub: sub}
	if !sub.Describe().Reuse {
		return s, nil
	}
	if err := s.start(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Session) start(ctx context.Context) error {
	live, err := s.sub.start(ctx)
	if err != nil {
		return err
	}
	if _, err := live.shake(true); err != nil {
		live.close()
		return err
	}

	s.live = live
	s.modTime = binaryTime(s.sub.Path)
	return nil
}

// Describe reports what the plugin said about itself.
func (s *Session) Describe() plugin.Description { return s.sub.Describe() }

// Generate serves one request, over the held connection when there is one.
func (s *Session) Generate(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
	if err := s.refresh(ctx); err != nil {
		return nil, err
	}
	if s.live == nil {
		return s.sub.Generate(ctx, req)
	}

	if err := s.live.conn.Send(req); err != nil {
		return nil, s.live.wrap(err)
	}
	var resp plugin.Response
	if err := s.live.conn.Recv(&resp); err != nil {
		return nil, s.live.wrap(err)
	}
	return &resp, nil
}

// refresh restarts a held plugin whose binary changed.
//
// Developing a plugin should not mean killing the watch that is exercising
// it, and a connection to the old process would keep serving the old code
// with nothing to say it had.
func (s *Session) refresh(ctx context.Context) error {
	if s.live == nil {
		return nil
	}
	if now := binaryTime(s.sub.Path); now.Equal(s.modTime) {
		return nil
	}

	s.Close()
	if err := s.start(ctx); err != nil {
		return err
	}
	s.restarts++
	return nil
}

// Close stops a held plugin.
func (s *Session) Close() {
	if s.live == nil {
		return
	}
	s.live.close()
	s.live = nil
}

func binaryTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// Reused reports whether this session is holding a connection, which is
// what a test asserts rather than inferring from timing.
func (s *Session) Reused() bool { return s.live != nil }

// Restarts counts how many times the held plugin was replaced because its
// binary changed.
func (s *Session) Restarts() int { return s.restarts }

// A Session is a Backend, so a watch loop and a single run take the same
// path.
var _ plugin.Backend = (*Session)(nil)
