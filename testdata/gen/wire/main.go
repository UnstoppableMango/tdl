// Command main prints, one per line, the JSON the generated Go writes for
// values checks.gen-go-zod parses with the generated zod schemas.
package main

import (
	"encoding/json"
	"fmt"
	"time"
)

func main() {
	half := Duration(500 * time.Millisecond)
	for _, v := range []Lease{
		{On: Date{1, time.January, 1}},
		{
			Ttl:    Duration(90 * time.Second),
			Ttls:   []Duration{0, Duration(time.Nanosecond), Duration(25*time.Hour + 30*time.Minute)},
			ByName: map[string]Duration{"week": Duration(7 * 24 * time.Hour)},
			Maybe:  &half,
			ByTtl:  map[Duration]string{Duration(time.Minute): "m"},
			On:     Date{2026, time.November, 13},
			Until:  &Date{2024, time.February, 29},
			Grace:  Grace(time.Hour),
		},
	} {
		b, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(b))
	}
}
