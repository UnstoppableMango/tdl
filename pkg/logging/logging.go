package logging

import (
	"os"

	"github.com/charmbracelet/log"
)

type channel chan<- []byte

func (c channel) Write(p []byte) (n int, err error) {
	c <- p
	return len(p), nil
}

func Init() {
	if env, ok := os.LookupEnv("UX_LOG_LEVEL"); ok {
		if level, err := log.ParseLevel(env); err == nil {
			log.SetLevel(level)
		}
	}
}
