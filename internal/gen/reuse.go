package gen

import (
	"context"
	"os"
	"time"

	"github.com/unstoppablemango/tdl/plugin"
)

// Session is a plugin kept alive across generations. A plugin that
// declared reuse serves many requests on one connection and must treat
// each as independent.
type Session struct {
	sub      *Subprocess
	desc     plugin.Description // what the last handshake said
	live     *session
	modTime  time.Time
	restarts int
}

// Open starts a plugin and holds the connection if it declared reuse;
// otherwise each generation gets a fresh process. A failed handshake is
// an error.
func Open(ctx context.Context, sub *Subprocess) (*Session, error) {
	s := &Session{sub: sub}
	if err := s.start(ctx); err != nil {
		return nil, err
	}
	if !s.desc.Reuse {
		s.Close()
	}
	return s, nil
}

func (s *Session) start(ctx context.Context) error {
	live, err := s.sub.start(ctx)
	if err != nil {
		return err
	}
	reply, err := live.shake(true)
	if err != nil {
		live.close()
		return err
	}

	s.live = live
	s.desc = description(reply)
	s.modTime = binaryTime(s.sub.Path)
	return nil
}

// Describe reports what the plugin said about itself when it was opened.
func (s *Session) Describe() plugin.Description { return s.desc }

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

// refresh restarts a held plugin whose binary changed, so a watch keeps
// running while the plugin is rebuilt.
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

// Reused reports whether this session is holding a connection.
func (s *Session) Reused() bool { return s.live != nil }

// Restarts counts how many times the held plugin was replaced because its
// binary changed.
func (s *Session) Restarts() int { return s.restarts }

var _ plugin.Backend = (*Session)(nil)
