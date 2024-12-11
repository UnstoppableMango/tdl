package crd2pulumi

import "path/filepath"

type LangOptions struct {
	Enabled bool
	Name    string
	Path    string
}

type Options struct {
	NodeJS  *LangOptions
	Python  *LangOptions
	Dotnet  *LangOptions
	Go      *LangOptions
	Java    *LangOptions
	Force   bool
	Version string
}

func (t Options) langs() map[string]*LangOptions {
	return map[string]*LangOptions{
		"nodejs": t.NodeJS,
		"python": t.Python,
		"dotnet": t.Dotnet,
		"golang": t.Go,
		"java":   t.Java,
	}
}

func (t Options) Paths(root string) map[string]string {
	paths := map[string]string{}
	for k, v := range t.langs() {
		if v == nil {
			continue
		}

		if v.Path != "" {
			paths[k] = v.Path
		} else {
			paths[k] = filepath.Join(root, k)
		}
	}

	return paths
}

func (t Options) Args(paths map[string]string) []string {
	args := ArgBuilder{}
	for k, v := range t.langs() {
		if v == nil {
			continue
		}

		if v.Enabled {
			args = args.LangOpt(k)
		}
		if v.Name != "" {
			args = args.NameOpt(k, v.Name)
		}
		if v.Enabled || v.Path != "" {
			args = args.PathOpt(k, paths[k])
		}
	}

	if t.Version != "" {
		args = args.VersionOpt(t.Version)
	}
	if t.Force {
		args = args.ForceOpt()
	}

	return args
}
