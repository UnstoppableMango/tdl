package internal

import (
	"io"
	"io/fs"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type writerOutput struct {
	writer io.Writer
}

func (w *writerOutput) Write(output afero.Fs) error {
	return afero.Walk(output, "",
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			file, err := output.Open(path)
			if err != nil {
				return err
			}

			_, err = io.Copy(w.writer, file)
			return err
		},
	)
}

func WriterOutput(writer io.Writer) tdl.Output {
	return &writerOutput{writer}
}
