# Sourced, hidden, at the start of demo.tape. Each command is followed by
# a blank line, so its output stands apart from the next prompt.
#
# `cat` highlights what it shows through bat, and prints it a line at a
# time so the screen scrolls rather than jumps. nix/demo.nix puts bat on
# PATH with TDL's syntax in its cache.
cat() {
	bat --theme="Catppuccin Mocha" --style=plain --color=always --paging=never "$@" |
		while IFS= read -r line; do
			printf '%s\n' "$line"
			sleep 0.04
		done
	echo
}

tdl() {
	command tdl "$@"
	echo
}
