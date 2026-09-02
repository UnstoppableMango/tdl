#!/usr/bin/env bash
# Run the conformance corpus through the derived parser.
#
# The corpus is the one parser/conformance_test.go walks, run here by
# tree-sitter rather than by Go, so a check the reference implementation
# passes is one the derived parser has to pass in the same terms.
# docs/design/treesitter-plan.md says why.
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

# Cases the derived parser is not expected to read yet, each naming the
# phase that deletes the entry. A deferred case that parses clean fails
# too, so nothing here outlives the phase that owns it.
deferred() {
	case "$1" in
	constraints) echo "regex literals need the external scanner: phase 6" ;;
	esac
}

status=0

for source in ../testdata/conformance/*/source.tdl; do
	name=$(basename "$(dirname "$source")")
	reason=$(deferred "$name")

	if out=$(tree-sitter parse --quiet --config-path "$config/config.json" "$source" 2>&1); then
		if [ -n "$reason" ]; then
			echo "FAIL  $name parses clean, delete the deferred entry ($reason)"
			status=1
		else
			echo "ok    $name"
		fi
	elif [ -n "$reason" ]; then
		echo "defer $name $reason"
	else
		echo "FAIL  $name"
		echo "$out" | sed 's/^/      /'
		status=1
	fi
done

exit $status
