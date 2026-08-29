package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/parser"
)

// scratchTemplate seeds a new playground file with one of everything the
// parser reads today, so there is something to twist immediately.
const scratchTemplate = `package scratch

primitive string
primitive int
primitive instant

type Email: string where {
  matches(/^[^@]+@[^@]+$/)
  length(3..254)
}

entity User {
  key id: string
  email: Email
  name: string? where { length(1..120) }
  tags: {string}
  role: Role = Member
}

enum Role { Member Admin }

target go for scratch {
  out("./gen/go")
  User.email => tag("json:email")
}
`

// playViews are the panes tdl play can render, in the order they appear.
var playViews = []string{"source", "fmt", "ast", "tokens", "stats"}

func newPlayCmd() *cobra.Command {
	var (
		views    []string
		interval time.Duration
		once     bool
		noClear  bool
	)

	cmd := &cobra.Command{
		Use:   "play [file]",
		Short: "Watch a TDL file and re-render it on every save",
		Long: "Watch a TDL file and re-render it on every save.\n\n" +
			"With no argument, play uses scratch.tdl in the current directory,\n" +
			"creating it from a starter template if it does not exist.\n\n" +
			"Views: " + strings.Join(playViews, ", ") + ".",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "scratch.tdl"
			if len(args) == 1 {
				path = args[0]
			}

			selected, err := resolveViews(views)
			if err != nil {
				return err
			}

			if err := seedScratch(path, len(args) == 1); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if once {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				render(out, path, string(data), selected)
				return nil
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
			defer stop()
			return watch(ctx, out, path, selected, interval, !noClear)
		},
	}

	cmd.Flags().StringSliceVar(&views, "views", []string{"fmt", "ast"},
		"panes to render, comma separated: "+strings.Join(playViews, ", ")+", or all")
	cmd.Flags().DurationVar(&interval, "interval", 200*time.Millisecond, "how often to poll the file for changes")
	cmd.Flags().BoolVar(&once, "once", false, "render a single time and exit instead of watching")
	cmd.Flags().BoolVar(&noClear, "no-clear", false, "append each render instead of clearing the screen")
	return cmd
}

// resolveViews validates the requested view names and returns them in
// canonical order, so --views output is stable regardless of flag order.
func resolveViews(requested []string) ([]string, error) {
	want := map[string]bool{}
	for _, v := range requested {
		v = strings.TrimSpace(v)
		if v == "all" {
			want = map[string]bool{}
			for _, name := range playViews {
				want[name] = true
			}
			break
		}
		if !slices.Contains(playViews, v) {
			return nil, fmt.Errorf("unknown view %q: pick from %s, or all", v, strings.Join(playViews, ", "))
		}
		want[v] = true
	}

	var selected []string
	for _, name := range playViews {
		if want[name] {
			selected = append(selected, name)
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("no views selected")
	}
	return selected, nil
}

// seedScratch writes the starter template when the playground file does
// not exist. An explicitly named file that is missing is an error rather
// than an invitation to create it.
func seedScratch(path string, explicit bool) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if explicit {
		return fmt.Errorf("%s does not exist", path)
	}
	return os.WriteFile(path, []byte(scratchTemplate), 0o644)
}

// watch polls path and re-renders whenever its contents change, until ctx
// is cancelled. Polling keeps the playground dependency-free and is
// plenty responsive at human typing speed.
func watch(ctx context.Context, out io.Writer, path string, views []string, interval time.Duration, clear bool) error {
	var last []byte
	first := true

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		data, err := os.ReadFile(path)
		switch {
		case err != nil && !errors.Is(err, os.ErrNotExist):
			return err
		case err != nil:
			data = nil
		}

		if first || !bytes.Equal(data, last) {
			if clear {
				fmt.Fprint(out, "\033[H\033[2J")
			}
			render(out, path, string(data), views)
			fmt.Fprintf(out, "\nwatching %s (ctrl-c to stop)\n", path)
			last = data
			first = false
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// render writes every selected view for src to out.
func render(out io.Writer, path, src string, views []string) {
	file, parseErr := parser.Parse(path, strings.NewReader(src))

	for _, view := range views {
		switch view {
		case "source":
			section(out, "source")
			fmt.Fprint(out, numberLines(src))
		case "fmt":
			section(out, "fmt")
			if file != nil {
				fmt.Fprint(out, ast.Fprint(file))
			}
		case "ast":
			section(out, "ast")
			if file != nil {
				fmt.Fprint(out, ast.Dump(file))
			}
		case "tokens":
			section(out, "tokens")
			fmt.Fprint(out, dumpTokens(path, src))
		case "stats":
			section(out, "stats")
			if file != nil {
				fmt.Fprint(out, stats(file))
			}
		}
	}

	if parseErr != nil {
		section(out, "errors")
		fmt.Fprint(out, annotateErrors(src, parseErr))
	}
}

func section(out io.Writer, name string) {
	fmt.Fprintf(out, "\n── %s %s\n\n", name, strings.Repeat("─", max(0, 60-len(name))))
}

// numberLines prefixes each line of src with its 1-based line number.
func numberLines(src string) string {
	lines := strings.Split(strings.TrimRight(src, "\n"), "\n")
	width := len(strconv.Itoa(len(lines)))

	var b strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&b, "%*d  %s\n", width, i+1, line)
	}
	return b.String()
}

// annotateErrors renders each parse error with the offending source line
// and a caret under the reported column.
func annotateErrors(src string, err error) string {
	var list parser.ErrorList
	if !errors.As(err, &list) {
		return err.Error() + "\n"
	}

	lines := strings.Split(src, "\n")

	var b strings.Builder
	for _, e := range list {
		fmt.Fprintf(&b, "%d:%d: %s\n", e.Pos.Line, e.Pos.Col, e.Msg)
		if e.Pos.Line >= 1 && e.Pos.Line <= len(lines) {
			line := strings.ReplaceAll(lines[e.Pos.Line-1], "\t", " ")
			fmt.Fprintf(&b, "  %s\n", line)
			fmt.Fprintf(&b, "  %s^\n", strings.Repeat(" ", max(0, e.Pos.Col-1)))
		}
	}
	return b.String()
}

// stats summarizes the shape of a parsed file: enough numbers to feel the
// difference between two ways of modelling the same data.
func stats(file *ast.File) string {
	var primitives, entities, values, enums, newtypes, aliases, targets int
	var fields, optional, keys, variants int

	countFields := func(fs []*ast.Field) {
		for _, f := range fs {
			fields++
			if f.Key {
				keys++
			}
			if f.Type != nil && f.Type.Optional {
				optional++
			}
		}
	}

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.PrimitiveDecl:
			primitives++
		case *ast.AliasDecl:
			aliases++
		case *ast.NewtypeDecl:
			newtypes++
		case *ast.TargetDecl:
			targets++
		case *ast.EnumDecl:
			enums++
			variants += len(d.Variants)
			for _, v := range d.Variants {
				countFields(v.Fields)
			}
		case *ast.StructDecl:
			switch d.Keyword {
			case "entity":
				entities++
			case "value":
				values++
			}
			for _, m := range d.Members {
				if f, ok := m.(*ast.Field); ok {
					countFields([]*ast.Field{f})
				}
			}
		}
	}

	var b strings.Builder
	row := func(name string, n int) { fmt.Fprintf(&b, "%-14s %d\n", name, n) }
	row("imports", len(file.Imports))
	row("declarations", len(file.Decls))
	row("primitives", primitives)
	row("entities", entities)
	row("values", values)
	row("enums", enums)
	row("newtypes", newtypes)
	row("aliases", aliases)
	row("targets", targets)
	row("fields", fields)
	row("keys", keys)
	row("optional", optional)
	row("variants", variants)
	return b.String()
}
