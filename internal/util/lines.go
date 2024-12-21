package util

import (
	"bufio"
	"io"
)

func Lines(r io.Reader) (lines []string, err error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		lines = append(lines, s.Text())
	}
	if s.Err() != nil {
		return nil, err
	} else {
		return
	}
}
