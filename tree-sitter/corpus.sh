#!/usr/bin/env bash
# Run the conformance corpus through the derived parser.
#
# testdata/conformance must parse with no ERROR node
# testdata/invalid must produce an ERROR node.

set -uo pipefail

cd "$(dirname "$0")"

# tree-sitter warns on every invocation when no parser directory is
# configured. The grammar here is found by path rather than by that list,
# so the repository is the whole configuration.
config=$(mktemp -d)
trap 'rm -rf "$config"' EXIT
printf '{"parser-directories":["%s"]}\n' "$PWD/.." >"$config/config.json"

status=0

# parse runs one case and reports whether the outcome was the wanted one.
# tree-sitter parse already exits nonzero on an ERROR node, so the exit
# code is the whole result.
parse() {
	local want=$1 source=$2
	local name out
	name=$(basename "$(dirname "$source")")

	if out=$(tree-sitter parse --quiet --config-path "$config/config.json" "$source" 2>&1); then
		if [ "$want" = clean ]; then
			echo "ok    $name"
		else
			echo "FAIL  $name parses clean, and should not"
			status=1
		fi
	elif [ "$want" = error ]; then
		echo "ok    $name rejected"
	else
		echo "FAIL  $name"
		echo "$out" | sed 's/^/      /'
		status=1
	fi
}

for source in ../testdata/conformance/*/source.tdl; do
	parse clean "$source"
done

for source in ../testdata/invalid/*/source.tdl; do
	parse error "$source"
done

# queries/highlights.scm is hand-written, so nothing regenerates it when a
# production is renamed. Compiling it is what catches that: a node name the
# grammar no longer has is an error rather than something quietly left
# uncolored. TestHighlightsCoverKeywords checks the other half, the
# anonymous tokens no tree carries the name of.
if out=$(tree-sitter query --quiet --config-path "$config/config.json" \
	queries/highlights.scm ../testdata/conformance/*/source.tdl 2>&1); then
	echo "ok    highlights.scm"
else
	echo "FAIL  highlights.scm"
	echo "$out" | sed 's/^/      /'
	status=1
fi

exit $status
