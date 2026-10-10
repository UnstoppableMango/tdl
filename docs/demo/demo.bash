# Sourced, hidden, at the start of demo.tape: `cat` highlights what it
# shows, TDL through the repository's tree-sitter grammar and everything
# else through bat, and prints it a line at a time so the screen scrolls
# rather than jumps. nix/demo.nix puts both on PATH and points
# TREE_SITTER_DIR at a config holding the grammar and the theme.
cat() {
	case "$1" in
	*.tdl) tree-sitter highlight "$@" 2>/dev/null ;;
	*) bat --theme="Catppuccin Mocha" --style=plain --color=always --paging=never "$@" ;;
	esac | while IFS= read -r line; do
		printf '%s\n' "$line"
		sleep 0.04
	done
}
