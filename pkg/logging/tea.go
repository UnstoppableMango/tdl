package logging

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

type TeaLogger struct{ *tea.Program }

func (log *TeaLogger) Write(p []byte) (n int, err error) {
	log.Program.Printf("%s", p)
	return len(p), nil
}

func LogToProgram(prog *tea.Program) {
	log.SetOutput(&TeaLogger{prog})
}
