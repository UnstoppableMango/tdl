package golang_test

import "testing"

// leaseSource holds a duration directly, in a list, in a map, as T?, and as
// a map key.
const leaseSource = `package lease

type Lease {
  ttl: duration
  ttls: [duration]
  byName: Map<string, duration>
  maybe: duration?
  byTtl: Map<duration, string>
}

target go for lease {
  json
}
`

// Under json, a duration is the package's Duration, which writes the
// convention's ISO 8601 string.
func TestJSONDuration(t *testing.T) {
	resp := generateTDL(t, leaseSource)
	if w := uncodedWarnings(resp); len(w) != 0 {
		t.Fatalf("diagnostics = %+v", w)
	}
	got := files(t, resp)
	contains(t, got["lease.go"],
		"Ttl Duration `json:\"ttl\"`",
		"Ttls []Duration `json:\"ttls\"`",
		"ByName map[string]Duration `json:\"byName\"`",
		"Maybe *Duration `json:\"maybe,omitempty\"`",
		"ByTtl map[Duration]string `json:\"byTtl\"`",
	)
	contains(t, got["duration.go"], "type Duration time.Duration")

	goTest(t, resp, `package lease

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWrites(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0:                               "PT0S",
		time.Nanosecond:                 "PT0.000000001S",
		500 * time.Millisecond:          "PT0.5S",
		90 * time.Second:                "PT1M30S",
		time.Hour:                       "PT1H",
		25*time.Hour + 30*time.Minute:   "P1DT1H30M",
		48 * time.Hour:                  "P2D",
		-1500 * time.Millisecond:        "-PT1.5S",
		time.Duration(-1 << 63):         "-P106751DT23H47M16.854775808S",
		time.Duration(1<<63 - 1):        "P106751DT23H47M16.854775807S",
	} {
		if got := Duration(d).String(); got != want {
			t.Errorf("Duration(%v) = %s, want %s", d, got, want)
		}
	}
}

const doc = `+"`"+`{"ttl":"PT1M30S","ttls":["PT0S","P1DT1H30M"],"byName":{"week":"P7D"},"maybe":"PT0.5S","byTtl":{"PT1M":"m"}}`+"`"+`

func TestMarshals(t *testing.T) {
	half := Duration(500 * time.Millisecond)
	want := Lease{
		Ttl:    Duration(90 * time.Second),
		Ttls:   []Duration{0, Duration(25*time.Hour + 30*time.Minute)},
		ByName: map[string]Duration{"week": Duration(7 * 24 * time.Hour)},
		Maybe:  &half,
		ByTtl:  map[Duration]string{Duration(time.Minute): "m"},
	}
	out, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != doc {
		t.Errorf("wrote %s, want %s", out, doc)
	}
}
`)
}

// Without json, a duration is a time.Duration and no helper is written.
func TestDurationWithoutJSON(t *testing.T) {
	got := files(t, generateTDL(t, "package lease\n\ntype Lease { ttl: duration }\n"))
	contains(t, got["lease.go"], "Ttl time.Duration")
	if _, ok := got["duration.go"]; ok {
		t.Errorf("a helper is written without json")
	}
}
