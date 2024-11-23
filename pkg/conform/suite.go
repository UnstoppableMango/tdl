package conform

import (
	"context"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"github.com/unmango/go/iter"
	"github.com/unmango/go/slices"
	"github.com/unmango/go/vcs/git"
	"github.com/unstoppablemango/tdl/pkg/testing/e2e"
)

type DumbSuite interface {
	Expect(string, ...e2e.Assertion)
}

type builder struct {
	tests      iter.Seq[*e2e.Test]
	assertions map[string][]e2e.Assertion
}

func (b *builder) Expect(name string, assertions ...e2e.Assertion) {
	b.assertions[name] = assertions
}

type Suite interface {
	GeneratorSuite
}

func NewSuite(tests ...*e2e.Test) DumbSuite {
	return &builder{tests: slices.Values(tests)}
}

// func NewSuite(name string, tests ...*e2e.Test) Suite {
// 	if len(tests) == 0 {
// 		panic("no tests defined")
// 	}

// 	return &suite{tests: slices.Values(tests)}
// }

// func IncludeTests(s e2e.Suite) Suite {
// 	return &suite{tests: s.Tests()}
// }

// TODO: Currently this is executing as the CLI runs, meaning it can execute outside of the repo and thus fail
// var (
// 	TypeScriptSuite = RequireLocalSuite("typescript")
// )

func ReadLocalSuite(ctx context.Context, fs afero.Fs, name string) (e2e.Suite, error) {
	root, err := git.Root(ctx)
	if err != nil {
		return nil, err
	}

	return e2e.ReadSuite(fs,
		filepath.Join(root, "conformance", name),
	)
}
