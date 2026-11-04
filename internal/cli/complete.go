package cli

import (
	"bytes"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/parser"
)

// Shell completion is cobra's: `tdl completion <shell>` writes a script
// that calls the hidden __complete command, so every shell is offered the
// values below. Completion never starts a plugin.

// tdlFiles completes a positional argument to .tdl files and directories.
func tdlFiles(*cobra.Command, []string, string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return []cobra.Completion{"tdl"}, cobra.ShellCompDirectiveFilterFileExt
}

// oneTDLFile is tdlFiles for a command taking at most one file.
func oneTDLFile(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return tdlFiles(cmd, args, toComplete)
}

// completeTargets offers every built-in backend and every plugin on PATH.
func completeTargets(_ *cobra.Command, _ []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return backendChoices(gen.BuiltinNames(), toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completeImporters offers the built-in backends that import and every
// plugin on PATH, which is asked whether it imports only when it runs.
func completeImporters(_ *cobra.Command, _ []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return backendChoices(gen.ReverseNames(), toComplete), cobra.ShellCompDirectiveNoFileComp
}

// completeDeclaredTargets offers the targets the files named so far
// declare, as gen lowers them. With no file yet, or one that does not
// lower, it offers what request does.
func completeDeclaredTargets(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	names, ok := declaredTargets(args)
	if !ok {
		return completeTargets(cmd, args, toComplete)
	}
	plugins := pathPlugins()
	var out []cobra.Completion
	for _, name := range names {
		if !strings.HasPrefix(name, toComplete) {
			continue
		}
		if _, builtin := gen.Builtin(name); builtin {
			out = append(out, cobra.CompletionWithDesc(name, "built in"))
		} else if path, ok := plugins[name]; ok {
			out = append(out, cobra.CompletionWithDesc(name, path))
		} else {
			out = append(out, name)
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// declaredTargets lowers each file and returns the names of its target
// blocks, sorted. A target block may name what an import declares, so
// each file lowers with the loader gen uses.
func declaredTargets(paths []string) ([]string, bool) {
	if len(paths) == 0 {
		return nil, false
	}
	var names []string
	for _, path := range paths {
		if isStdin(path) {
			return nil, false
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, false
		}
		file, err := parser.Parse(path, bytes.NewReader(data))
		if err != nil {
			return nil, false
		}
		model, diags := sema.Lower(file, sema.WithLoader(sema.FSLoader{}))
		if len(diags) > 0 {
			return nil, false
		}
		for _, b := range model.GetTargets() {
			names = append(names, b.GetMeta().GetName())
		}
	}
	if len(names) == 0 {
		return nil, false
	}
	slices.Sort(names)
	return slices.Compact(names), true
}

// backendChoices offers builtins, described "built in", then every plugin
// on PATH, described by its path.
func backendChoices(builtins []string, toComplete string) []cobra.Completion {
	var out []cobra.Completion
	for _, name := range builtins {
		if strings.HasPrefix(name, toComplete) {
			out = append(out, cobra.CompletionWithDesc(name, "built in"))
		}
	}
	plugins := pathPlugins()
	for _, name := range slices.Sorted(maps.Keys(plugins)) {
		if strings.HasPrefix(name, toComplete) {
			out = append(out, cobra.CompletionWithDesc(name, plugins[name]))
		}
	}
	return out
}

// pathPlugins maps each tdl-gen-<name> on PATH to the path gen.Find
// would run. A name a built-in serves is left out, since gen.Resolve never
// lets PATH shadow one. Nothing is started.
func pathPlugins() map[string]string {
	plugins := map[string]string{}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name, ok := strings.CutPrefix(e.Name(), gen.CommandPrefix)
			if runtime.GOOS == "windows" {
				name = strings.TrimSuffix(name, filepath.Ext(name))
			}
			if !ok || name == "" {
				continue
			}
			if _, builtin := gen.Builtin(name); builtin {
				continue
			}
			if _, seen := plugins[name]; seen {
				continue
			}
			if sub, err := gen.Find(name); err == nil {
				plugins[name] = sub.Path
			}
		}
	}
	return plugins
}

// commaList completes a comma-separated flag value: each word offered
// follows what is already typed, leaves out the words already given, and
// takes no space after it, so another comma can follow. A word in first
// is offered only as the first value, and nothing follows it.
func commaList(words, first []string) cobra.CompletionFunc {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		directive := cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		given, word := "", toComplete
		if i := strings.LastIndex(toComplete, ","); i >= 0 {
			given, word = toComplete[:i+1], toComplete[i+1:]
		}
		have := strings.Split(strings.TrimSuffix(given, ","), ",")
		candidates := words
		if given == "" {
			candidates = slices.Concat(first, words)
		} else if slices.ContainsFunc(have, func(w string) bool { return slices.Contains(first, w) }) {
			return nil, directive
		}

		var out []cobra.Completion
		for _, w := range candidates {
			if strings.HasPrefix(w, word) && !slices.Contains(have, w) {
				out = append(out, given+w)
			}
		}
		return out, directive
	}
}

// completeLossCodes completes --allow-lossy.
var completeLossCodes = commaList(host.LossCodes, nil)

// registerCompletion attaches a flag's completion. It fails only for a
// flag the command lacks, which the tests would catch.
func registerCompletion(cmd *cobra.Command, flag string, f cobra.CompletionFunc) {
	if err := cmd.RegisterFlagCompletionFunc(flag, f); err != nil {
		panic(err)
	}
}
