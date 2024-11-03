package mediatype

import (
	"errors"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

type MatchFunc[T any] func() (T, error)

type Matcher[T any] struct {
	Protobuf MatchFunc[T]
	Json     MatchFunc[T]
	Yaml     MatchFunc[T]
	Other    MatchFunc[T]
}

// Error returned by Match when no match was found
var Unmatched = errors.New("no match for mediatype")

func Match[T any](media tdl.MediaType, match Matcher[T]) (T, error) {
	switch media {
	case ApplicationJson:
		fallthrough
	case TextJson:
		if match.Json != nil {
			return match.Json()
		}
	case ApplicationGoogleProtobuf:
		fallthrough
	case ApplicationProtobuf:
		fallthrough
	case ApplicationXProtobuf:
		if match.Protobuf != nil {
			return match.Protobuf()
		}
	case ApplicationXYaml:
		fallthrough
	case ApplicationYaml:
		fallthrough
	case TextYaml:
		if match.Yaml != nil {
			return match.Yaml()
		}
	}

	if match.Other != nil {
		return match.Other()
	} else {
		var t T
		return t, Unmatched
	}
}
