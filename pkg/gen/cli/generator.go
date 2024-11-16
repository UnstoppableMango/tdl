package cli

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/unmango/go/option"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type cli struct {
	name string
	args []string
	enc  tdl.MediaType
}

type Option func(*cli)

// Execute implements tdl.Generator.
func (c cli) Execute(spec *tdlv1alpha1.Spec, output afero.Fs) error {
	log.Debug("creating temp directory")
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return fmt.Errorf("creating exec context: %w", err)
	}

	stderr := &bytes.Buffer{}
	cmd := exec.Command(c.name, c.args...)
	cmd.Stdin = mediatype.NewReader(spec, c.enc)
	cmd.Stderr = stderr
	cmd.Dir = tmp

	log.Debug("executing command")
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("executing generator: %w", err)
	}
	if stderr.Len() > 0 {
		return fmt.Errorf("executing generator: %s", stderr)
	}

	log.Debug("walking output directory")
	return afero.Walk(afero.NewOsFs(), tmp,
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				log.Debugf("creating directory: %s", path)
				return output.Mkdir(path, os.ModeDir)
			}

			src, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("opening output: %w", err)
			}

			rel, err := filepath.Rel(tmp, path)
			if err != nil {
				return fmt.Errorf("output path: %w", err)
			}

			err = afero.WriteReader(output, rel, src)
			if err != nil {
				return fmt.Errorf("writing output output: %w", err)
			}

			log.Debugf("wrote %d bytes to %s", info.Size(), rel)
			return nil
		},
	)
}

func New(name string, options ...Option) tdl.Generator {
	gen := cli{
		name: name,
		enc:  mediatype.ApplicationProtobuf,
	}
	option.ApplyAll(&gen, options)

	return gen
}

func WithArgs(args ...string) Option {
	return func(c *cli) {
		c.args = args
	}
}

func WithEncoding(media tdl.MediaType) Option {
	return func(c *cli) {
		c.enc = media
	}
}
