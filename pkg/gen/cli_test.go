package gen_test

import (
	"context"
	"os/exec"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/sink"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Cli", func() {
	var generator *gen.Cli

	BeforeEach(func() {
		// Sanity check
		_, err := exec.LookPath("uml2ts")
		Expect(err).NotTo(HaveOccurred())

		generator = gen.NewCli("uml2ts")
	})

	It("should initialize", func() {
		Expect(generator).NotTo(BeNil())
	})

	It("should write to sink", func(ctx context.Context) {
		sink := sink.NewPipe()
		spec := &tdlv1alpha1.Spec{
			Name: "testing",
			Types: map[string]*tdlv1alpha1.Type{
				"testing": {},
			},
		}

		err := generator.Execute(ctx, spec, sink)

		Expect(err).NotTo(HaveOccurred())
		units := slices.Collect(sink.Units())
		Expect(units).NotTo(BeEmpty())
	})
})
