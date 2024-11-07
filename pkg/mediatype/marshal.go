package mediatype

import (
	tdl "github.com/unstoppablemango/tdl/pkg"
	"gopkg.in/yaml.v3"
)

func Marshal(val interface{}, media tdl.MediaType) ([]byte, error) {
	switch media {
	case ApplicationXYaml:
		fallthrough
	case ApplicationYaml:
		return yaml.Marshal(val)
	}

	return nil, UnsupportedErr(media)
}

func Unmarshal(data []byte, out interface{}, media tdl.MediaType) error {
	switch media {
	case ApplicationXYaml:
		fallthrough
	case ApplicationYaml:
		return yaml.Unmarshal(data, out)
	}

	return UnsupportedErr(media)
}
