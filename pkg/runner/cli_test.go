package runner

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
)

// We can't seem to compare proto specs with reflect.DeepEquals
// because proto fields (i.e. sizeCache) differ

var (
	binDir = os.Getenv("BIN_DIR")
	bin    = path.Join(binDir, "go_echo")
)

var _ = Describe("runner/cli", func() {
	var runner uml.Runner
	var cliRunner *cli
	var err error

	BeforeEach(func() {
		runner, err = NewCli(bin, WithArgs("ts"))
		cliRunner, _ = runner.(*cli)
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

		It("should initialize a default logger", func() {
			Expect(cliRunner.log).NotTo(BeNil())
		})

		It("should use the supplied logger", func() {
			expected := slog.New(slog.NewTextHandler(os.Stdout, nil))

			sut, err := NewCli(bin, WithLogger(expected))
			cli, ok := sut.(*cli)

			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(cli.log).To(Equal(expected))
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

			Expect(spec.Name).To(Equal(input.Name))
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

			Expect(spec.Name).To(Equal(input.Name))
		})
	})
})
