package mediatype

import (
	"fmt"
	"path/filepath"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

func Guess(path string) (tdl.MediaType, error) {
	switch filepath.Ext(path) {
	case ".yaml":
		fallthrough
	case ".yml":
		return ApplicationYaml, nil
	case ".json":
		return ApplicationJson, nil
	case ".pb":
		return ApplicationProtobuf, nil
	}

	return "", fmt.Errorf("unable to guess media type: %s", path)
}
