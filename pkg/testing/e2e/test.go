package e2e

import (
	"path/filepath"
	"regexp"

	"github.com/spf13/afero"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var OutputRegex = regexp.MustCompile(".*/output.*")

type Test struct {
	Name     string
	Spec     *tdlv1alpha1.Spec
	Expected afero.Fs
}

type RawTest struct {
	Name   string
	Input  []byte
	Output []byte
}

func ReadTest(fsys afero.Fs, name, path string) (*Test, error) {
	files, err := afero.ReadDir(fsys, path)
	if err != nil {
		return nil, err
	}

	var filename string
	for _, f := range files {
		match, err := filepath.Match("input.*", f.Name())
		if err != nil {
			panic(err)
		}
		if match {
			filename = f.Name()
			break
		}
	}

	filepath := filepath.Join(path, filename)
	media, err := mediatype.Guess(filepath)
	if err != nil {
		return nil, err
	}

	data, err := afero.ReadFile(fsys, filepath)
	if err != nil {
		return nil, err
	}

	var spec tdlv1alpha1.Spec
	err = mediatype.Unmarshal(data, &spec, media)
	if err != nil {
		return nil, err
	}

	return &Test{
		Name:     name,
		Spec:     &spec,
		Expected: afero.NewRegexpFs(fsys, OutputRegex),
	}, nil
}
