package uml

import (
	"encoding/json"
	"fmt"
	"mime"

	"google.golang.org/protobuf/proto"
)

func Unmarshal(v string, b []byte, spec *Spec) error {
	media, _, err := mime.ParseMediaType(v)
	if err != nil {
		return err
	}

	switch media {
	case "application/json":
		err = json.Unmarshal(b, spec)
	case "application/x-protobuf":
	case "application/protobuf":
	case "application/vnd.google.protobuf":
		err = proto.Unmarshal(b, spec)
	default:
		err = fmt.Errorf("unsupported media type: %s", media)
	}

	return err
}
