package io

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"

	"github.com/unstoppablemango/tdl/pkg/tdl"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
	"google.golang.org/protobuf/proto"
)

type (
	PipeFunc   func(io.Reader, io.Writer) error
	LookupFunc func(string) (PipeFunc, error)
)

func TargetToBin(target string) string {
	return "uml2" + target
}

func BinFromPath(target string) (PipeFunc, error) {
	binary, err := exec.LookPath(TargetToBin(target))
	if err != nil {
		return nil, fmt.Errorf("looking up target: %w", err)
	}

	gen := Exec(binary)
	return UnmarshalProto(gen), nil
}

func UnmarshalProto(generator tdl.Gen) PipeFunc {
	return func(r io.Reader, w io.Writer) error {
		data, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}

		var spec tdlv1alpha1.Spec
		if err := proto.Unmarshal(data, &spec); err != nil {
			return fmt.Errorf("unmarshalling spec: %w", err)
		}

		return generator(&spec, NewSink(w))
	}
}

func Exec(binary string) tdl.Gen {
	return func(s *tdlv1alpha1.Spec, sink tdl.Sink) error {
		data, err := proto.Marshal(s)
		if err != nil {
			return fmt.Errorf("marshalling spec: %w", err)
		}

		cmd := exec.Command(binary)
		cmd.Stdin = bytes.NewReader(data)
		// cmd.Stdout = w // TODO

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("executing binary: %w", err)
		}

		return nil
	}
}
