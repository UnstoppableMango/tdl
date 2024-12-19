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

func (t *Options) apply(args []string) error {
	f := pflag.NewFlagSet("crd2pulumi", pflag.ContinueOnError)

	f.BoolVarP(&t.Dotnet.Enabled, "dotnet", "d", false, "")
	f.StringVar(&t.Dotnet.Name, "dotnetName", "", "")
	f.StringVar(&t.Dotnet.Path, "dotnetPath", "", "")

	f.BoolVarP(&t.Go.Enabled, "go", "g", false, "")
	f.StringVar(&t.Go.Name, "goName", "", "")
	f.StringVar(&t.Go.Path, "goPath", "", "")

	f.BoolVarP(&t.NodeJS.Enabled, "nodejs", "n", false, "")
	f.StringVar(&t.NodeJS.Name, "nodejsName", "", "")
	f.StringVar(&t.NodeJS.Path, "nodejsPath", "", "")

	f.BoolVarP(&t.Python.Enabled, "python", "p", false, "")
	f.StringVar(&t.Python.Name, "pythonName", "", "")
	f.StringVar(&t.Python.Path, "pythonPath", "", "")

	f.BoolVarP(&t.Force, "force", "f", false, "")
	f.StringVarP(&t.Version, "version", "v", "", "")

	return f.Parse(args)
}

func Parse(args []string) (o Options, err error) {
	err = o.apply(args)
	return
}
