package golang_test

import (
	"strings"
	"testing"
)

// leaseSource holds a duration directly, in a list, in a map, as T?, as a
// map key, and under a newtype and a newtype over that.
const leaseSource = `package lease

type Lease {
  ttl: duration
  ttls: [duration]
  byName: Map<string, duration>
  maybe: duration?
  byTtl: Map<duration, string>
  grace: Grace
  notice: Notice
}

type Grace: duration

type Notice: Grace

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
		"ByTtl map[Duration]string `json:\"byTtl\"`",
	)
	contains(t, got["grace.go"], "type Grace Duration")
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
		"P1W1D":                        "not an ISO 8601 duration",
		"P1WT1H":                       "not an ISO 8601 duration",
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
	if err := json.Unmarshal([]byte("90"), &d); err == nil || !strings.Contains(err.Error(), "cannot unmarshal number") {
		t.Errorf("a number: error %v", err)
	}
}

const doc = `+"`"+`{"ttl":"PT1M30S","ttls":["PT0S","P1DT1H30M"],"byName":{"week":"P7D"},"maybe":"PT0.5S","byTtl":{"PT1M":"m"},"grace":"PT1H","notice":"P7D"}`+"`"+`

func TestRoundTrip(t *testing.T) {
	half := Duration(500 * time.Millisecond)
	want := Lease{
		Ttl:    Duration(90 * time.Second),
		Ttls:   []Duration{0, Duration(25*time.Hour + 30*time.Minute)},
		ByName: map[string]Duration{"week": Duration(7 * 24 * time.Hour)},
		Maybe:  &half,
		ByTtl:  map[Duration]string{Duration(time.Minute): "m"},
		Grace:  Grace(time.Hour),
		Notice: Notice(7 * 24 * time.Hour),
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

// Import reads the helper type back as the primitive, and declares
// nothing for it.
func TestImportReadsDuration(t *testing.T) {
	src, _ := importGo(t, files(t, generateTDL(t, leaseSource)))
	contains(t, src, "ttl: duration", "ttls: [duration]", "maybe: duration?", "type Grace: duration", "type Notice: Grace")
	if strings.Contains(src, "type Duration") {
		t.Errorf("import declares the helper type:\n%s", src)
	}
}

// A model declaring its own Duration gets WireDuration in its file, and a
// package declaring both names refuses the field.
func TestJSONDurationNameClash(t *testing.T) {
	resp := generateTDL(t, `package lease

type Duration { length: duration }

target go for lease {
  json
}
`)
	got := files(t, resp)
	contains(t, got["duration.go"], "Length WireDuration `json:\"length\"`")
	contains(t, got["wire_duration.go"], "type WireDuration time.Duration", "func (w *WireDuration) UnmarshalText(text []byte) error")
	goTest(t, resp, "package lease\n")
}

func TestJSONDurationBothNamesTaken(t *testing.T) {
	resp := generateTDL(t, `package lease

type Duration { length: duration }

enum Wire {
  Duration { n: int }
  Other { n: int }
}

target go for lease {
  json
}
`)
	got := files(t, resp)
	if _, ok := got["wire_duration.go"]; ok {
		t.Errorf("a helper is written beside the variant WireDuration")
	}
	var refused bool
	for _, d := range resp.GetDiagnostics() {
		refused = refused || strings.Contains(d.GetMessage(), "declares both")
	}
	if !refused {
		t.Errorf("diagnostics = %v, want one refusing duration", resp.GetDiagnostics())
	}
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

// A type carrying the helper's doc over another type is a newtype.
func TestImportReadsOnlyTheHelper(t *testing.T) {
	src, _ := importGo(t, map[string]string{"lease.go": `package lease

// Duration is a time.Duration that the JSON wire convention writes as an ISO 8601 duration with no year or month part, such as PT1M30S.
type Duration int32
`})
	contains(t, src, "type Duration: int32")
}

// dateSource holds a date directly, in a list, as T?, as a map key, and
// under a newtype.
const dateSource = `package lease

type Lease {
  on: date
  dates: [date]
  until: date?
  byDay: Map<date, string>
  signed: Signed
}

type Signed: date

target go for lease {
  json
}
`

// Under json, a date is the package's Date, which writes YYYY-MM-DD and
// reads it back.
func TestJSONDate(t *testing.T) {
	resp := generateTDL(t, dateSource)
	if w := uncodedWarnings(resp); len(w) != 0 {
		t.Fatalf("diagnostics = %+v", w)
	}
	for _, d := range resp.GetDiagnostics() {
		if strings.Contains(d.GetMessage(), "instant") {
			t.Errorf("a date warns it reads back as instant: %s", d.GetMessage())
		}
	}
	got := files(t, resp)
	contains(t, got["lease.go"], "On Date `json:\"on\"`", "Until *Date `json:\"until,omitempty\"`", "ByDay map[Date]string `json:\"byDay\"`")
	contains(t, got["date.go"], "type Date struct {")

	goTest(t, resp, `package lease

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

const doc = `+"`"+`{"on":"2026-11-13","dates":["0001-01-01","9999-12-31"],"until":"2024-02-29","byDay":{"2026-01-02":"x"},"signed":"2026-11-26"}`+"`"+`

func TestRoundTrip(t *testing.T) {
	leap := Date{2024, time.February, 29}
	want := Lease{
		On:     Date{2026, time.November, 13},
		Dates:  []Date{{1, time.January, 1}, {9999, time.December, 31}},
		Until:  &leap,
		ByDay:  map[Date]string{{2026, time.January, 2}: "x"},
		Signed: Signed{2026, time.November, 26},
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

func TestRefuses(t *testing.T) {
	for _, in := range []string{"2026-13-01", "2026-02-30", "2026-11-13T00:00:00Z", "26-11-13", "2026-1-13", "", " 2026-11-13"} {
		var d Date
		if err := json.Unmarshal([]byte("\"" + in + "\""), &d); err == nil || !strings.Contains(err.Error(), "YYYY-MM-DD") {
			t.Errorf("%q: error %v", in, err)
		}
	}
	for _, d := range []Date{{}, {2026, time.February, 30}, {10000, time.January, 1}, {-1, time.January, 1}} {
		if _, err := json.Marshal(d); err == nil {
			t.Errorf("%v marshals", d)
		}
	}
}
`)
}

// Import reads the helper back as date, and a newtype over it as one over
// date.
func TestImportReadsDate(t *testing.T) {
	src, diags := importGo(t, files(t, generateTDL(t, dateSource)))
	contains(t, src, "on: date", "until: date?", "byDay: {date -> string}", "type Signed: date")
	if strings.Contains(src, "type Date") {
		t.Errorf("import declares the helper type:\n%s", src)
	}
	for _, d := range diags {
		if strings.Contains(d.GetMessage(), "Text") {
			t.Errorf("import warns about a helper method: %s", d.GetMessage())
		}
	}
}

// Without json, a date is a time.Time and still warns that it reads back
// as instant.
func TestDateWithoutJSON(t *testing.T) {
	resp := generateTDL(t, "package lease\n\ntype Lease { on: date }\n\ntarget go for lease {\n  readback\n}\n")
	contains(t, files(t, resp)["lease.go"], "On time.Time")
	var warned bool
	for _, d := range resp.GetDiagnostics() {
		warned = warned || strings.Contains(d.GetMessage(), "reads back as instant")
	}
	if !warned {
		t.Errorf("diagnostics = %v, want the instant warning", resp.GetDiagnostics())
	}
}

// A model declaring its own Date gets WireDate.
func TestJSONDateNameClash(t *testing.T) {
	resp := generateTDL(t, "package lease\n\ntype Date { on: date }\n\ntarget go for lease {\n  json\n}\n")
	contains(t, files(t, resp)["wire_date.go"], "type WireDate struct {")
	goTest(t, resp, "package lease\n")
}
