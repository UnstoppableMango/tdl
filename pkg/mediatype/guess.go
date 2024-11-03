package mediatype

import (
	"fmt"
	"path/filepath"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

func Guess(token string) (tdl.MediaType, error) {
	switch filepath.Ext(token) {
	case ".yaml":
		fallthrough
	case ".yml":
		return ApplicationYaml, nil
	case ".json":
		return ApplicationJson, nil
	case ".pb":
		return ApplicationProtobuf, nil
	}

	return "", fmt.Errorf("unknown media type: %s", token)
}
