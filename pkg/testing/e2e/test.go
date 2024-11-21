package e2e

import (
	"errors"
	"path/filepath"
	"regexp"

	"github.com/spf13/afero"
	"github.com/unmango/go/iter"
	"github.com/unmango/go/iter/seqs"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var (
	InputRegex  = regexp.MustCompile("(input|source).*")
	OutputRegex = regexp.MustCompile(".*/?(output|target).*")
)

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

func ListTests(fsys afero.Fs, path string) (iter.Seq[*Test], error) {
	infos, err := afero.ReadDir(fsys, path)
	if err != nil {
		return nil, err
	}

	seq := iter.Empty[*Test]()
	for _, info := range infos {
		test, err := ReadTest(fsys,
			filepath.Join(path, info.Name()),
		)
		if err != nil {
			return nil, err
		}

		seq = seqs.Append(seq, test)
	}

	return seq, nil
}

func ReadTest(fsys afero.Fs, path string) (*Test, error) {
	filename, err := FindInput(fsys, path)
	if err != nil {
		return nil, err
	}

	inputpath := filepath.Join(path, filename)
	media, err := mediatype.Guess(inputpath)
	if err != nil {
		return nil, err
	}

	data, err := afero.ReadFile(fsys, inputpath)
	if err != nil {
		return nil, err
	}

	var spec tdlv1alpha1.Spec
	err = mediatype.Unmarshal(data, &spec, media)
	if err != nil {
		return nil, err
	}

	return &Test{
		Name: filepath.Base(path),
		Spec: &spec,
		Expected: afero.NewRegexpFs(
			afero.NewBasePathFs(fsys, path),
			OutputRegex,
		),
	}, nil
}

func FindInput(fs afero.Fs, path string) (string, error) {
	files, err := afero.ReadDir(fs, path)
	if err != nil {
		return "", err
	}

	for _, f := range files {
		name := f.Name()
		if InputRegex.MatchString(name) {
			return name, nil
		}
	}

	return "", errors.New("no input file found")
}
