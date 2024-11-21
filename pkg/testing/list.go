package testing

import (
	"path/filepath"
	"regexp"

	"github.com/spf13/afero"
	"github.com/unmango/go/iter"
	"github.com/unmango/go/iter/seqs"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var outputRegex = regexp.MustCompile(".*/output.*")

func List(fsys afero.Fs, path string) (iter.Seq[*Test], error) {
	dirs, err := afero.ReadDir(fsys, path)
	if err != nil {
		return nil, err
	}

	seq := iter.Empty[*Test]()
	for _, d := range dirs {
		name := d.Name()
		t, err := ReadTest(fsys, name, filepath.Join(path, name))
		if err != nil {
			return nil, err
		}

		seq = seqs.Append(seq, t)
	}

	return seq, nil
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
		Expected: afero.NewRegexpFs(fsys, outputRegex),
	}, nil
}
