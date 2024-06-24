package uml

import (
	"encoding/json"
	"fmt"
	"mime"
	"regexp"

	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

var (
	jsonMatcher = regexp.MustCompile(`.*\.json`)
	yamlMatcher = regexp.MustCompile(`.*\.ya?ml`)
)

func Marshal(typ string, spec *Spec) ([]byte, error) {
	mediaType, err := parseMediaType(typ)
	if err != nil {
		return nil, err
	}

	switch mediaType {
	case "application/json":
	case "text/json":
		return json.Marshal(spec)
	case "application/x-protobuf":
	case "application/protobuf":
	case "application/vnd.google.protobuf":
		return proto.Marshal(spec)
	case "application/x-yaml":
	case "application/yaml":
	case "text/yaml":
		return yaml.Marshal(spec)
	}

	return nil, fmt.Errorf("unsupported media type: %s", mediaType)
}

func Unmarshal(typ string, data []byte, spec *Spec) error {
	mediaType, err := parseMediaType(typ)
	if err != nil {
		return err
	}

	switch mediaType {
	case "application/json":
	case "text/json":
		return json.Unmarshal(data, spec)
	case "application/x-protobuf":
	case "application/protobuf":
	case "application/vnd.google.protobuf":
		return proto.Unmarshal(data, spec)
	case "application/x-yaml":
	case "application/yaml":
	case "text/yaml":
		return yaml.Unmarshal(data, spec)
	}

	return fmt.Errorf("unsupported media type: %s", mediaType)
}

func GuessMediaType(x string) (string, error) {
	if x == "stdin" {
		return "application/protobuf", nil
	}

	if yamlMatcher.MatchString(x) {
		return "application/yaml", nil
	}

	if jsonMatcher.MatchString(x) {
		return "application/json", nil
	}

	return "", fmt.Errorf("failed to guess media type for: %s", x)
}

func parseMediaType(x string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(x)
	return mediaType, err
}
