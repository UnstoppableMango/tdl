package flags

import "github.com/spf13/pflag"

func ConformanceTest(flags *pflag.FlagSet, p *bool) error {
	const name = "conformance-test"
	flags.BoolVar(p, name, false,
		"Signals that the current execution is for a conformance test",
	)

	return flags.MarkHidden(name)
}
