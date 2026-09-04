#!/usr/bin/env bash
# Package this extension and install it into a running VS Code.
#
# The extension is committed as a directory, and `code --install-extension`
# wants a .vsix, so one is built here and thrown away. The two files a
# .vsix carries beyond the extension itself are written below rather than
# by `vsce`, which would pull npm in for a zip and a manifest.
#
# The supported way in is this command. A directory dropped into an
# extensions folder registers on a remote server and never reaches the
# client, which looks exactly like the grammar not working.
#
# The installed copy is a copy: regenerate the grammar with `make textmate`
# and run this again to see the change.
#
#	command make vscode-install

set -euo pipefail

cd "$(dirname "$0")"

version=$(sed -n 's/^[[:space:]]*"version": "\(.*\)".*/\1/p' package.json | head -1)
publisher=$(sed -n 's/^[[:space:]]*"publisher": "\(.*\)".*/\1/p' package.json | head -1)
name=$(sed -n 's/^[[:space:]]*"name": "\(.*\)".*/\1/p' package.json | head -1)

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

mkdir -p "$work/extension"
cp -RL package.json language-configuration.json syntaxes "$work/extension/"

cat >"$work/extension.vsixmanifest" <<EOF
<?xml version="1.0" encoding="utf-8"?>
<PackageManifest Version="2.0.0" xmlns="http://schemas.microsoft.com/developer/vsx-schema/2011">
  <Metadata>
    <Identity Language="en-US" Id="${name}" Version="${version}" Publisher="${publisher}" />
    <DisplayName>TDL</DisplayName>
    <Description xml:space="preserve">Syntax highlighting for the Type Description Language</Description>
    <Categories>Programming Languages</Categories>
  </Metadata>
  <Installation>
    <InstallationTarget Id="Microsoft.VisualStudio.Code" />
  </Installation>
  <Dependencies />
  <Assets>
    <Asset Type="Microsoft.VisualStudio.Code.Manifest" Path="extension/package.json" Addressable="true" />
  </Assets>
</PackageManifest>
EOF

cat >"$work/[Content_Types].xml" <<'EOF'
<?xml version="1.0" encoding="utf-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension=".json" ContentType="application/json" />
  <Default Extension=".vsixmanifest" ContentType="text/xml" />
</Types>
EOF

vsix="$work/${publisher}.${name}-${version}.vsix"
(cd "$work" && zip -qr "$vsix" extension extension.vsixmanifest '[Content_Types].xml')

# The CLI talks to the window over a socket named in the environment, which
# VS Code's own terminal sets and nothing else does. Falling back to the
# newest socket is what makes this work from an agent's shell or a plain
# ssh session; a stale one refuses the connection rather than installing
# into the wrong window.
install() {
	code --install-extension "$vsix" --force && return 0

	for socket in $(ls -t /run/user/"$(id -u)"/vscode-ipc-*.sock 2>/dev/null); do
		if VSCODE_IPC_HOOK_CLI="$socket" code --install-extension "$vsix" --force; then
			return 0
		fi
	done
	return 1
}

if ! install; then
	echo "install failed: is VS Code running, and is 'code' on PATH?" >&2
	exit 1
fi

echo "Reload the VS Code window to pick it up."
