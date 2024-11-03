package mediatype

import (
	"fmt"

	"github.com/unstoppablemango/tdl/pkg/tdl"
)

type Option func() tdl.MediaType

var (
	ApplicationGoogleProtobuf tdl.MediaType = "application/vnd.google.protobuf"
	ApplicationJson           tdl.MediaType = "application/json"
	ApplicationProtobuf       tdl.MediaType = "application/protobuf"
	ApplicationXProtobuf      tdl.MediaType = "application/x-protobuf"
	ApplicationXYaml          tdl.MediaType = "application/x-yaml"
	ApplicationYaml           tdl.MediaType = "application/yaml"
	TextJson                  tdl.MediaType = "text/json"
	TextYaml                  tdl.MediaType = "text/yaml"
)

func Parse(value string) (tdl.MediaType, error) {
	switch value {
	case string(ApplicationJson):
		return ApplicationJson, nil
	case string(ApplicationProtobuf):
		return ApplicationProtobuf, nil
	case string(ApplicationGoogleProtobuf):
		return ApplicationGoogleProtobuf, nil
	case string(ApplicationXProtobuf):
		return ApplicationXProtobuf, nil
	case string(ApplicationXYaml):
		return ApplicationXYaml, nil
	case string(ApplicationYaml):
		return ApplicationYaml, nil
	case string(TextJson):
		return TextJson, nil
	case string(TextYaml):
		return TextYaml, nil
	}

	return "", UnsupportedErr(value)
}

func Supported(media tdl.MediaType) bool {
	_, err := Parse(string(media))
	return err == nil
}

func UnsupportedErr[M string | tdl.MediaType](media M) error {
	return fmt.Errorf("unsupported media type: %s", media)
}
