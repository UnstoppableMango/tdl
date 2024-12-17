package internal

import (
	"fmt"
	"io/fs"

	"github.com/spf13/afero"
)

func PrintFs(fsys afero.Fs) error {
	return afero.Walk(fsys, "",
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == "" || info.IsDir() {
				return nil
			}

			_, err = fmt.Println(path)
			return err
		},
	)
}
