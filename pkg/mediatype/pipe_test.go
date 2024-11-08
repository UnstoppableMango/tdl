package mediatype_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/mediatype"
	"github.com/unstoppablemango/tdl/pkg/sink"
	"github.com/unstoppablemango/tdl/pkg/spec"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

var _ = Describe("Pipe", func() {
	Describe("PipeRead", func() {
		DescribeTable("Yaml",
			Entry(nil, mediatype.ApplicationXYaml),
			Entry(nil, mediatype.ApplicationYaml),
			Entry(nil, mediatype.TextYaml),
			func(media tdl.MediaType) {
				s := &tdlv1alpha1.Spec{Name: "testing"}
				data, err := yaml.Marshal(s)
				Expect(err).NotTo(HaveOccurred())
				reader := bytes.NewReader(data)
				sink := sink.NewPipe()

				pipeline := mediatype.PipeRead[gen.FromReader](gen.NoOp, media, spec.Zero)
				err = pipeline(reader, sink)

				Expect(err).NotTo(HaveOccurred())
			},
		)
	})
})
