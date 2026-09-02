#!/usr/bin/env bash
# Run the conformance corpus through the derived parser.
#
# The corpus is the one parser/conformance_test.go walks, run here by
# tree-sitter rather than by Go, so a check the reference implementation
# passes is one the derived parser has to pass in the same terms.
# docs/design/treesitter.md says why.
#
# testdata/conformance must parse with no ERROR node, and testdata/invalid
# must produce one. The invalid half checks the ERROR and not the message:
# error.golden is the reference implementation's wording, and a second
# parser agreeing on the diagnosis is a different promise from agreeing
# that the file is bad.
#
#	command make test-treesitter

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

exit $status
