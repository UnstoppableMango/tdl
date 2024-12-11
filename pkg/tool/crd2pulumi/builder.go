package crd2pulumi

import "fmt"

type ArgBuilder []string

func (b ArgBuilder) append(args ...string) ArgBuilder {
	return append(b, args...)
}

func (b ArgBuilder) ForceOpt() ArgBuilder {
	return b.append("--force")
}

func (b ArgBuilder) LangOpt(lang string) ArgBuilder {
	return b.append(fmt.Sprintf("--%s", lang))
}

func (b ArgBuilder) NameOpt(lang, name string) ArgBuilder {
	return b.append(fmt.Sprintf("--%sName", lang), name)
}

func (b ArgBuilder) PathOpt(lang, path string) ArgBuilder {
	return b.append(fmt.Sprintf("--%sPath", lang), path)
}

func (b ArgBuilder) VersionOpt(version string) ArgBuilder {
	return b.append("--version", version)
}
