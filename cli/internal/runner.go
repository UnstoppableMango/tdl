package cli

import (
	"io"
	"log/slog"
	"os"
)

type runnerCmd struct {
	args []string
	log  *slog.Logger
}

func (c *runnerCmd) run(onInput func(key string, input io.Reader) error) error {
	inputs := map[string]io.Reader{}
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		c.log.Debug("found data in stdin")
		inputs["stdin"] = os.Stdin
	}

	c.log.Debug("checking for file args")
	for i, a := range c.args[1:] {
		scoped := c.log.With("index", i, "file", a)
		scoped.Debug("found file in args")
		if input, err := os.Open(a); err == nil {
			scoped.Debug("opened file")
			inputs[a] = input
		} else {
			scoped.Debug("failed to open file")
			return err
		}
	}

	for key, input := range inputs {
		c.log.Debug("executing input callback", "key", key)
		if err := onInput(key, input); err != nil {
			return err
		}
	}

	return nil
}
