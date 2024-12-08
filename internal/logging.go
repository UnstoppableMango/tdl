package internal

import (
	"os"

	"github.com/charmbracelet/log"
)

func InitLogging() {
	if env, ok := os.LookupEnv("UX_LOG_LEVEL"); ok {
		if level, err := log.ParseLevel(env); err == nil {
			log.SetLevel(level)
		}
	}
}
