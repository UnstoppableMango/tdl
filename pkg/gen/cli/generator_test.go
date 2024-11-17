package cli_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"github.com/unstoppablemango/tdl/internal/util"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen/cli"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	"github.com/unstoppablemango/tdl/pkg/testing"
	. "github.com/unstoppablemango/tdl/pkg/testing/matcher"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Generator", func() {
	It("should write protobuf to default path", func(ctx context.Context) {
		fs := afero.NewMemMapFs()
		spec := &tdlv1alpha1.Spec{Name: "CLI Generator Test"}
		expected, err := mediatype.Marshal(spec, mediatype.ApplicationProtobuf)
		Expect(err).NotTo(HaveOccurred())
		c := cli.New(util.BinPath("uml2uml"))

		err = c.Execute(ctx, spec, fs)

		Expect(err).NotTo(HaveOccurred())
		Expect(fs).To(ContainFileWithBytes("out", expected))
	})

	DescribeTable("Encoding",
		testing.MediaTypeEntries(),
		func(ctx context.Context, media tdl.MediaType) {
			fs := afero.NewMemMapFs()
			spec := &tdlv1alpha1.Spec{Name: "CLI Generator Test"}
			expected, err := mediatype.Marshal(spec, media)
			Expect(err).NotTo(HaveOccurred())
			c := cli.New(util.BinPath("uml2uml"), cli.WithEncoding(media))

			err = c.Execute(ctx, spec, fs)

			Expect(err).NotTo(HaveOccurred())
			Expect(fs).To(ContainFileWithBytes("out", expected))
		},
	)

	It("should pass args", func(ctx context.Context) {
		fs := afero.NewMemMapFs()
		spec := &tdlv1alpha1.Spec{}
		c := cli.New("echo", cli.WithArgs("blah"))

		err := c.Execute(ctx, spec, fs)

		Expect(err).NotTo(HaveOccurred())
	})
})
