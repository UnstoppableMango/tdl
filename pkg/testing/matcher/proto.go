package matcher

import (
	"fmt"

	"github.com/onsi/gomega/types"
	"google.golang.org/protobuf/proto"
)

type equalProtoMatcher struct {
	expected proto.Message
}

// Match implements types.GomegaMatcher.
func (p *equalProtoMatcher) Match(actual interface{}) (success bool, err error) {
	if message, ok := actual.(proto.Message); !ok {
		return false, fmt.Errorf("not a proto message: %#v", actual)
	} else {
		return proto.Equal(message, p.expected), nil
	}
}

// FailureMessage implements types.GomegaMatcher.
func (p *equalProtoMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto equal\n\t%#v", actual, p.expected)
}

// NegatedFailureMessage implements types.GomegaMatcher.
func (p *equalProtoMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to equal\n\t%#v", actual, p.expected)
}

func EqualProto(expected proto.Message) types.GomegaMatcher {
	return &equalProtoMatcher{expected}
}
