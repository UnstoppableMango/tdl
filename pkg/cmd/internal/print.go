package internal

import (
	"fmt"
	"io/fs"

	"github.com/spf13/afero"
)

func PrintFs(fsys afero.Fs) error {
	var count int

	err := afero.Walk(fsys, "",
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == "" || info.IsDir() {
				return nil
			}

			count++
			_, err = fmt.Println(path)
			return err
		},
	)
	if err != nil {
		return err
	}

	if count == 0 {
		fmt.Println("no output")
	}

	return nil
}
