# Sourced, hidden, at the start of demo.tape: `cat` highlights what it
# shows, TDL through the repository's tree-sitter grammar and everything
# else through bat. nix/demo.nix puts both on PATH and points
# TREE_SITTER_DIR at a config holding the grammar and the theme.
cat() {
	case "$1" in
	*.tdl) tree-sitter highlight "$@" 2>/dev/null ;;
	*) bat --theme="Catppuccin Mocha" --style=plain --paging=never "$@" ;;
	esac
}
