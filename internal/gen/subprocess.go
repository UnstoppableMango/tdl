package gen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/unstoppablemango/tdl/plugin"
)

// CommandPrefix is what a target name is prefixed with to find its
// executable. A target tdl has no backend for resolves to
// tdl-gen-<name> on PATH, the way git and protoc find their subcommands.
const CommandPrefix = "tdl-gen-"

// DefaultTimeout bounds how long a plugin has to answer.
//
// A hung plugin in CI should fail with a diagnosis rather than consume the
// job's whole time budget and report nothing. A legitimately slow backend
// will want to raise this, which is what tdl.toml is for; until that
// exists it is a constant.
const DefaultTimeout = 2 * time.Minute

// Subprocess is a backend running as tdl-gen-<name>.
//
// It implements [plugin.Backend], so everything above it treats a plugin
// and a compiled-in backend the same way. That is the protocol's one real
// claim, and this type is what makes it testable rather than asserted.
type Subprocess struct {
	Name    string
	Path    string
	Timeout time.Duration
}

// Find locates the executable for a target name.
func Find(name string) (*Subprocess, error) {
	path, err := exec.LookPath(CommandPrefix + name)
	if err != nil {
		return nil, fmt.Errorf("no backend named %s: %w", name, err)
	}
	return &Subprocess{Name: name, Path: path, Timeout: DefaultTimeout}, nil
}

// Describe starts the plugin, shakes hands, and stops it again.
//
// Describing costs a process. Generate starts its own, so a description
// fetched here is not carried into it; keeping one connection across both
// is what phase 8's reuse is for.
func (s *Subprocess) Describe() plugin.Description {
	desc, err := s.describe()
	if err != nil {
		// Describe cannot report an error, so a plugin that will not shake
		// hands describes itself as nothing and Generate reports why.
		return plugin.Description{Name: s.Name}
	}
	return desc
}

func (s *Subprocess) describe() (plugin.Description, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout())
	defer cancel()

	session, err := s.start(ctx)
	if err != nil {
		return plugin.Description{}, err
	}
	defer session.close()

	reply, err := session.shake(false)
	if err != nil {
		return plugin.Description{}, err
	}
	return plugin.Description{
		Name:       reply.GetName(),
		Version:    reply.GetVersion(),
		Directives: reply.GetDirectives(),
		Reuse:      reply.GetFeatures().GetReuse(),
	}, nil
}

// Generate runs one request through the plugin.
func (s *Subprocess) Generate(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout())
	defer cancel()

	session, err := s.start(ctx)
	if err != nil {
		return nil, err
	}
	defer session.close()

	if _, err := session.shake(false); err != nil {
		return nil, err
	}
	if err := session.conn.Send(req); err != nil {
		return nil, session.wrap(err)
	}

	var resp plugin.Response
	if err := session.conn.Recv(&resp); err != nil {
		return nil, session.wrap(err)
	}
	return &resp, nil
}

func (s *Subprocess) timeout() time.Duration {
	if s.Timeout <= 0 {
		return DefaultTimeout
	}
	return s.Timeout
}

// session is one running plugin process and the connection to it.
type session struct {
	name   string
	cmd    *exec.Cmd
	conn   *plugin.Conn
	stderr *safeBuffer
	ctx    context.Context

	// waitOnce guards cmd.Wait, which both wrap and close need and
	// neither may call twice.
	waitOnce sync.Once
}

// wait reaps the child. os/exec finishes copying its stderr before Wait
// returns, so calling this is what makes the buffer complete rather than
// whatever had arrived by then.
//
// Only safe once every read from the stdout pipe has finished, which is
// why wrap calls it on the path where the plugin is already gone and
// close calls it after shutting stdin.
func (s *session) wait() {
	s.waitOnce.Do(func() { _ = s.cmd.Wait() })
}

// safeBuffer collects a plugin's stderr.
//
// os/exec copies the child's stderr from a goroutine it owns, and wrap
// reads it from whichever goroutine hit the error, so the two need a
// lock between them. On the path where the plugin has died, wrap reaps
// it first and the copy is complete; on any other, reading gives
// whatever has arrived, because a diagnostic that blocks behind a live
// plugin is worse than a partial one.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (s *Subprocess) start(ctx context.Context) (*session, error) {
	cmd := exec.CommandContext(ctx, s.Path)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.Name, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.Name, err)
	}

	var stderr safeBuffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting %s: %w", s.Path, err)
	}
	return &session{
		name:   s.Name,
		cmd:    cmd,
		conn:   plugin.NewConn(stdout, stdin),
		stderr: &stderr,
		ctx:    ctx,
	}, nil
}

// shake sends the handshake and reads the reply, refusing a plugin that
// will not accept what is about to be sent.
func (s *session) shake(watch bool) (*plugin.HandshakeReply, error) {
	err := s.conn.Send(&plugin.Handshake{
		FramingVersion: plugin.FramingVersion,
		IrVersion:      plugin.IRVersion,
		Watch:          watch,
	})
	if err != nil {
		return nil, s.wrap(err)
	}

	var reply plugin.HandshakeReply
	if err := s.conn.Recv(&reply); err != nil {
		return nil, s.wrap(err)
	}
	if !reply.GetAccepted() {
		return nil, fmt.Errorf("%s refused framing %d and %s: %s",
			s.name, plugin.FramingVersion, plugin.IRVersion, reply.GetRefusal())
	}
	return &reply, nil
}

// wrap turns a connection failure into something that says which plugin
// failed and what it said on the way out.
//
// A plugin that dies mid-response leaves a read error that describes the
// pipe rather than the problem; its stderr is usually the actual answer.
func (s *session) wrap(err error) error {
	if errors.Is(s.ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%s did not answer within the timeout", s.name)
	}

	msg := fmt.Sprintf("%s: %v", s.name, err)
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		msg = fmt.Sprintf("%s stopped before answering", s.name)
		// The plugin is gone and nothing else will read its stdout, so
		// its stderr is finite and worth waiting for. Reading without
		// this gives whatever the copy had reached, and losing it
		// defeats the point of the message. Bounded by the context,
		// since CommandContext kills the child at the deadline.
		s.wait()
	}
	if out := strings.TrimSpace(s.stderr.String()); out != "" {
		msg += "\n" + indent(out)
	}
	return errors.New(msg)
}

func indent(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = "\t" + l
	}
	return strings.Join(lines, "\n")
}

func (s *session) close() {
	if p, ok := s.conn.Writer().(io.Closer); ok {
		_ = p.Close()
	}
	s.wait()
}
