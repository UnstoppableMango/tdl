package media

import (
	"fmt"
	"path/filepath"

	"github.com/unstoppablemango/tdl/pkg/tdl"
)

func Guess(token string) (tdl.MediaType, error) {
	switch filepath.Ext(token) {
	case ".yaml":
		fallthrough
	case ".yml":
		return ApplicationYaml, nil
	case ".json":
		return ApplicationJson, nil
	case ".proto":
		fallthrough
	case ".pb":
		return ApplicationProtobuf, nil
	}

	return "", fmt.Errorf("unknown media type: %s", token)
}
