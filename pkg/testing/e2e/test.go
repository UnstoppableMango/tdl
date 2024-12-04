package e2e

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/spf13/afero"
	"github.com/unmango/go/iter"
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

func ListTests(fs afero.Fs, path string) (iter.Seq[*Test], error) {
	infos, err := afero.ReadDir(fs, path)
	if err != nil {
		return nil, err
	}

	seq := iter.Empty[*Test]()
	for _, info := range infos {
		test, err := ReadTest(fs,
			filepath.Join(path, info.Name()),
		)
		if err != nil {
			return nil, err
		}

		seq = iter.Append(seq, test)
	}

	return seq, nil
}

func ReadTest(fs afero.Fs, path string) (*Test, error) {
	filename, err := FindInput(fs, path)
	if err != nil {
		return nil, fmt.Errorf("reading test: %w", err)
	}

	inputpath := filepath.Join(path, filename)
	media, err := mediatype.Guess(inputpath)
	if err != nil {
		return nil, fmt.Errorf("reading test: %w", err)
	}

	data, err := afero.ReadFile(fs, inputpath)
	if err != nil {
		return nil, fmt.Errorf("reading test: %w", err)
	}

	var spec tdlv1alpha1.Spec
	err = mediatype.Unmarshal(data, &spec, media)
	if err != nil {
		return nil, fmt.Errorf("reading test: %w", err)
	}

	expected := afero.NewRegexpFs(
		afero.NewBasePathFs(fs, path),
		OutputRegex,
	)
	empty, err := afero.IsEmpty(expected, "")
	if err != nil {
		return nil, fmt.Errorf("reading test: %w", err)
	}
	if empty {
		return nil, errors.New("no output found")
	}

	return &Test{
		Name:     filepath.Base(path),
		Spec:     &spec,
		Expected: expected,
	}, nil
}

func FindInput(fs afero.Fs, path string) (string, error) {
	files, err := afero.ReadDir(fs, path)
	if err != nil {
		return "", fmt.Errorf("finding input: %w", err)
	}

	for _, f := range files {
		name := f.Name()
		if InputRegex.MatchString(name) {
			return name, nil
		}
	}

	return "", errors.New("no input file found")
}
