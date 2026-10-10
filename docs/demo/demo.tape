# The README's demo. Render it with `make demo`, which builds
# `nix build .#demo` and copies the GIF beside this file.
#
# It runs in a scratch directory holding copies of shop.tdl and demo.bash,
# with `tdl` on PATH, so `tdl gen` writes nowhere in the repository.

Output demo.gif

Set Shell "bash"
Set FontFamily "JetBrains Mono"
Set FontSize 18
Set Width 1200
Set Height 820
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

# demo.bash makes `cat` highlight syntax and print a line at a time, so
# each file scrolls the last one up. Go indents with tabs.
Hide
Type "source demo.bash && tabs -4 && clear"
Enter
Show

Type "cat shop.tdl"
Sleep 400ms
Enter
Sleep 4s

Type "tdl gen shop.tdl"
Sleep 400ms
Enter
Sleep 2s

Type "cat ts/shop.ts"
Sleep 400ms
Enter
Sleep 4s

Type "cat go/order.go go/status.go"
Sleep 400ms
Enter
Sleep 4s
