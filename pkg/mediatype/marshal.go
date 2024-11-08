package mediatype

import (
	"encoding/json"

	tdl "github.com/unstoppablemango/tdl/pkg"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

func Marshal(m proto.Message, media tdl.MediaType) ([]byte, error) {
	switch media {
	case ApplicationJson:
		fallthrough
	case TextJson:
		return json.Marshal(m)
	case ApplicationGoogleProtobuf:
		fallthrough
	case ApplicationProtobuf:
		fallthrough
	case ApplicationXProtobuf:
		return proto.Marshal(m)
	case ApplicationXYaml:
		fallthrough
	case ApplicationYaml:
		fallthrough
	case TextYaml:
		return yaml.Marshal(m)
	}

	return nil, UnsupportedErr(media)
}

func Unmarshal(data []byte, m proto.Message, media tdl.MediaType) error {
	switch media {
	case ApplicationJson:
		fallthrough
	case TextJson:
		return json.Unmarshal(data, m)
	case ApplicationGoogleProtobuf:
		fallthrough
	case ApplicationProtobuf:
		fallthrough
	case ApplicationXProtobuf:
		return proto.Unmarshal(data, m)
	case ApplicationXYaml:
		fallthrough
	case ApplicationYaml:
		fallthrough
	case TextYaml:
		return yaml.Unmarshal(data, m)
	}

	return UnsupportedErr(media)
}
