package runner

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
)

var (
	binDir = os.Getenv("BIN_DIR")
	bin    = path.Join(binDir, "go_echo")
)

var _ = Describe("runner/cli", func() {
	var runner uml.Runner
	var cliRunner *cli
	var err error

	BeforeEach(func() {
		runner, err = NewCli(bin)
		cliRunner = runner.(*cli)
	})

	Describe("NewCli", func() {
		It("should create a CLI runner", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(runner).NotTo(BeNil())
		})

		It("should use the provided path", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(cliRunner.Path).To(Equal(bin))
		})
	})

	Describe("From", func() {
		It("should work", func() {
			Expect(err).NotTo(HaveOccurred())

			By("marshalling an arbitrary spec")
			input := &uml.Spec{Name: "testing"}
			data, err := proto.Marshal(input)
			Expect(err).NotTo(HaveOccurred())

			By("creating a byte reader")
			reader := bytes.NewReader(data)

			By("executing the runner")
			spec, err := cliRunner.From(context.Background(), reader)
			Expect(err).NotTo(HaveOccurred())

			Expect(spec).To(BeEquivalentTo(input))
		})
	})

	Describe("Gen", func() {
		It("should work", func() {
			Expect(err).NotTo(HaveOccurred())

			input := &uml.Spec{Name: "testing"}
			buf := &bytes.Buffer{}

			By("executing the runner")
			err = cliRunner.Gen(context.Background(), input, buf)
			Expect(err).NotTo(HaveOccurred())

			By("reading from the buffer")
			data, err := io.ReadAll(buf)
			Expect(err).NotTo(HaveOccurred())

			By("unmarshalling the result")
			spec := &uml.Spec{}
			err = proto.Unmarshal(data, spec)
			Expect(err).NotTo(HaveOccurred())

			Expect(spec).To(Equal(input))
		})
	})
})
