package crd2pulumi

import "fmt"

type builder []string

func (b builder) append(args ...string) builder {
	return append(b, args...)
}

func (b builder) forceopt() builder {
	return b.append("--force")
}

func (b builder) langopt(lang string) builder {
	return b.append(fmt.Sprintf("--%s", lang))
}

func (b builder) nameopt(lang, name string) builder {
	return b.append(fmt.Sprintf("--%s", lang), name)
}

func (b builder) pathopt(lang, path string) builder {
	return b.append(fmt.Sprintf("--%sPath", lang), path)
}

func (b builder) versionopt(version string) builder {
	return b.append("--version", version)
}
