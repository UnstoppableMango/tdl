package golang_test

import "testing"

// leaseSource holds a duration directly, in a list, in a map, and as T?.
const leaseSource = `package lease

type Lease {
  ttl: duration
  ttls: [duration]
  byName: Map<string, duration>
  maybe: duration?
}

target go for lease {
  json
}
`

// Under json, a duration is the package's Duration, which writes the
// convention's ISO 8601 string and reads it back.
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
	)
	contains(t, got["duration.go"], "type Duration time.Duration")

	goTest(t, resp, `package lease

import (
	"encoding/json"
	"reflect"
	"strings"
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

func TestReads(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"PT0S":                          0,
		"P0D":                           0,
		"PT1M30S":                       90 * time.Second,
		"PT0.5S":                        500 * time.Millisecond,
		"PT0,5S":                        500 * time.Millisecond,
		"PT1.0000000019S":               time.Second + time.Nanosecond,
		"P1W":                           7 * 24 * time.Hour,
		"P1DT1H30M":                     25*time.Hour + 30*time.Minute,
		"+PT1H":                         time.Hour,
		"-PT1.5S":                       -1500 * time.Millisecond,
		"PT36H":                         36 * time.Hour,
		"-P106751DT23H47M16.854775808S": time.Duration(-1 << 63),
	} {
		var d Duration
		if err := json.Unmarshal([]byte("\"" + in + "\""), &d); err != nil {
			t.Errorf("%s: %v", in, err)
		} else if time.Duration(d) != want {
			t.Errorf("%s reads as %v, want %v", in, time.Duration(d), want)
		}
	}
}

func TestRefuses(t *testing.T) {
	for in, want := range map[string]string{
		"P1Y":                          "year or month",
		"P1M":                          "year or month",
		"P1Y2DT3H":                     "year or month",
		"":                             "not an ISO 8601 duration",
		"P":                            "not an ISO 8601 duration",
		"PT":                           "not an ISO 8601 duration",
		"1H":                           "not an ISO 8601 duration",
		"PT1H1H":                       "not an ISO 8601 duration",
		"PT1S1M":                       "not an ISO 8601 duration",
		"P1H":                          "not an ISO 8601 duration",
		"PT1D":                         "not an ISO 8601 duration",
		"PT1.5M":                       "not an ISO 8601 duration",
		"PT.5S":                        "not an ISO 8601 duration",
		"PT1.S":                        "not an ISO 8601 duration",
		"PTT1S":                        "not an ISO 8601 duration",
		"P106751DT23H47M16.854775808S": "does not fit",
		"P99999999999999999999D":       "does not fit",
	} {
		var d Duration
		err := json.Unmarshal([]byte("\"" + in + "\""), &d)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: error %v, want one containing %q", in, err, want)
		}
	}
	var d Duration
	if err := json.Unmarshal([]byte("90"), &d); err == nil || !strings.Contains(err.Error(), "ISO 8601 string") {
		t.Errorf("a number: error %v", err)
	}
}

const doc = `+"`"+`{"ttl":"PT1M30S","ttls":["PT0S","P1DT1H30M"],"byName":{"week":"P7D"},"maybe":"PT0.5S"}`+"`"+`

func TestRoundTrip(t *testing.T) {
	half := Duration(500 * time.Millisecond)
	want := Lease{
		Ttl:    Duration(90 * time.Second),
		Ttls:   []Duration{0, Duration(25*time.Hour + 30*time.Minute)},
		ByName: map[string]Duration{"week": Duration(7 * 24 * time.Hour)},
		Maybe:  &half,
	}
	out, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != doc {
		t.Errorf("wrote %s, want %s", out, doc)
	}
	var got Lease
	if err := json.Unmarshal([]byte(doc), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("read %#v, want %#v", got, want)
	}
}
`)
}

// A model declaring its own Duration gets WireDuration in its file.
func TestJSONDurationNameClash(t *testing.T) {
	resp := generateTDL(t, `package lease

type Duration { length: duration }

target go for lease {
  json
}
`)
	got := files(t, resp)
	contains(t, got["duration.go"], "Length WireDuration `json:\"length\"`")
	contains(t, got["wire_duration.go"], "type WireDuration time.Duration", "func (w *WireDuration) UnmarshalJSON(data []byte) error")
	goTest(t, resp, "package lease\n")
}

// A file directive puts the helper in the one file.
func TestJSONDurationInOneFile(t *testing.T) {
	resp := generateTDL(t, `package lease

type Lease { ttl: duration }

target go for lease {
  json
  file("lease.go")
}
`)
	got := files(t, resp)
	if len(got) != 1 {
		t.Fatalf("files = %v, want lease.go alone", sources(got))
	}
	contains(t, got["lease.go"], "Ttl Duration", "type Duration time.Duration")
	goTest(t, resp, "package lease\n")
}

// Without json, a duration is a time.Duration and no helper is written.
func TestDurationWithoutJSON(t *testing.T) {
	got := files(t, generateTDL(t, "package lease\n\ntype Lease { ttl: duration }\n"))
	contains(t, got["lease.go"], "Ttl time.Duration")
	if _, ok := got["duration.go"]; ok {
		t.Errorf("a helper is written without json")
	}
}

// A foreign directive on duration still names its type.
func TestJSONDurationForeign(t *testing.T) {
	resp := generateTDL(t, `package lease

type Lease { ttl: duration }

target go for lease {
  json
  duration => foreign("net/netip", "Addr")
}
`)
	got := files(t, resp)
	contains(t, got["lease.go"], "Ttl netip.Addr")
	if _, ok := got["duration.go"]; ok {
		t.Errorf("a helper is written for a foreign duration")
	}
}
