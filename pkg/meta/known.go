package meta

type wellKnown struct {
	Lang string
	Name string
}

type Langs struct {
	TypeScript string
	Go         string
}

var WellKnown = wellKnown{
	Lang: "lang",
	Name: "name",
}

var Lang = Langs{
	TypeScript: "ts",
	Go:         "go",
}
