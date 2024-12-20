package crd2pulumi

import (
	"path/filepath"

	"github.com/spf13/pflag"
)

type LangOptions struct {
	Enabled bool
	Name    string
	Path    string
}

func (o LangOptions) IsZero() bool {
	return !o.Enabled && o.Name == "" && o.Path == ""
}

type Options struct {
	NodeJS  LangOptions
	Python  LangOptions
	Dotnet  LangOptions
	Go      LangOptions
	Java    LangOptions
	Force   bool
	Version string
}

func (o Options) ShouldInclude(path string) bool {
	switch filepath.Ext(path) {
	case ".cs":
		return !o.Dotnet.IsZero()
	case ".go":
		return !o.Go.IsZero()
	case ".ts":
		return !o.NodeJS.IsZero()
	case ".py":
		return !o.Python.IsZero()
	default:
		return false
	}
}

func (o Options) langs() map[string]LangOptions {
	return map[string]LangOptions{
		"nodejs": o.NodeJS,
		"python": o.Python,
		"dotnet": o.Dotnet,
		"golang": o.Go,
		"java":   o.Java,
	}
}

func (o Options) Paths(root string) map[string]string {
	paths := map[string]string{}
	for k, v := range o.langs() {
		if v.Path != "" {
			paths[k] = v.Path
		} else {
			paths[k] = filepath.Join(root, k)
		}
	}

	return paths
}

func (o Options) Args(paths map[string]string) []string {
	args := ArgBuilder{}
	for k, v := range o.langs() {
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

	if o.Version != "" {
		args = args.VersionOpt(o.Version)
	}
	if o.Force {
		args = args.ForceOpt()
	}

	return args
}

func (t *Options) Apply(args []string) error {
	f := pflag.NewFlagSet("crd2pulumi", pflag.ContinueOnError)

	f.BoolVarP(&t.Dotnet.Enabled, "dotnet", "d", t.Dotnet.Enabled, "")
	f.StringVar(&t.Dotnet.Name, "dotnetName", t.Dotnet.Name, "")
	f.StringVar(&t.Dotnet.Path, "dotnetPath", t.Dotnet.Path, "")

	f.BoolVarP(&t.Go.Enabled, "go", "g", t.Go.Enabled, "")
	f.StringVar(&t.Go.Name, "goName", t.Go.Name, "")
	f.StringVar(&t.Go.Path, "goPath", t.Go.Path, "")

	f.BoolVarP(&t.NodeJS.Enabled, "nodejs", "n", t.NodeJS.Enabled, "")
	f.StringVar(&t.NodeJS.Name, "nodejsName", t.NodeJS.Name, "")
	f.StringVar(&t.NodeJS.Path, "nodejsPath", t.NodeJS.Path, "")

	f.BoolVarP(&t.Python.Enabled, "python", "p", t.Python.Enabled, "")
	f.StringVar(&t.Python.Name, "pythonName", t.Python.Name, "")
	f.StringVar(&t.Python.Path, "pythonPath", t.Python.Path, "")

	f.BoolVarP(&t.Force, "force", "f", t.Force, "")
	f.StringVarP(&t.Version, "version", "v", t.Version, "")

	return f.Parse(args)
}

func Parse(args []string) (o Options, err error) {
	err = o.Apply(args)
	return
}
