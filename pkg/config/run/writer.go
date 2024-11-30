package run

import (
	"fmt"
	"io"
	"io/fs"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type writerOutput struct {
	writer io.Writer
}

// String implements tdl.Output.
func (w *writerOutput) String() string {
	return fmt.Sprintf("writer")
}

func (w *writerOutput) Write(output afero.Fs) error {
	count := 0
	err := afero.Walk(output, "",
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			count++
			file, err := output.Open(path)
			if err != nil {
				return err
			}

			n, err := io.Copy(w.writer, file)
			log.Debugf("wrote %d bytes", n)

			return err
		},
	)

	log.Debugf("copied %d files", count)
	return err
}

func WriterOutput(writer io.Writer) tdl.Output {
	return &writerOutput{writer}
}
