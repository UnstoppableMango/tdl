//go:build makeplugin

// Command tdl-gmk is a GNU make loadable object. Built as tdl.so and loaded
// with make's load directive, it defines:
//
//	$(tdl-files dir...)     the .tdl files under each dir
//	$(tdl-imports file...)  each file and every file it imports, transitively
//
// Each returns sorted paths relative to the directory of the makefile that
// loaded it. A failure stops make with the error, rather than handing a rule
// part of a list.
//
// The build tag keeps it out of go build ./..., which cannot link a main
// package against symbols that exist only inside make:
//
//	go build -tags makeplugin -buildmode=c-shared -o tdl.so ./cmd/tdl-gmk
package main

/*
// ELF leaves the symbols make defines for the loader; Mach-O refuses to
// link without them unless told to look them up at load time.
#cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup
#include <gnumake.h>
*/
import "C"

import (
	"strings"

	"github.com/unmango/go-gmk"

	"github.com/unstoppablemango/tdl/internal/makefn"
)

func main() {}

// root is the directory of the makefile that loaded this object. Paths are
// resolved against it and returned relative to it. make calls setup while it
// is reading the load line, and at that point the makefile holding that line
// is the last entry in MAKEFILE_LIST.
var root string

//export tdl_gmk_setup
func tdl_gmk_setup(*C.gmk_floc) C.int {
	root = gmk.Expand("$(abspath $(dir $(lastword $(MAKEFILE_LIST))))")
	gmk.AddFunction("tdl-files", list(makefn.Files), 0, 0, gmk.FuncDefault)
	gmk.AddFunction("tdl-imports", list(makefn.Imports), 0, 0, gmk.FuncDefault)
	return 1
}

// list adapts f to a make function. make splits arguments at commas; a list
// is separated by spaces, so both separate paths here.
func list(f func(root string, paths []string) ([]string, error)) gmk.Func {
	return func(name string, args []string) string {
		out, err := f(root, strings.Fields(strings.Join(args, " ")))
		if err != nil {
			gmk.Expand("$(error " + name + ": " + strings.ReplaceAll(err.Error(), "$", "$$") + ")")
			return ""
		}
		return strings.Join(out, " ")
	}
}
