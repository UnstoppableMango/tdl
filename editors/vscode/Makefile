.PHONY: check bundle install test

check: node_modules/.package-lock.json
	npm run typecheck
	npm run check

bundle: dist/extension.js

install: dist/extension.js syntaxes/tdl.tmLanguage.json
	./install.sh

# VS_CODE names the editor, which `nix develop .#vscode` sets; the server
# is built fresh so the run tests this checkout's tdl.
test: node_modules/.package-lock.json syntaxes/tdl.tmLanguage.json
	go build -C ../.. -o $(CURDIR)/.vscode-test/tdl ./cmd/tdl
	TDL=$(CURDIR)/.vscode-test/tdl xvfb-run -a npm test

# npm writes this hidden lockfile on every install.
node_modules/.package-lock.json: package.json package-lock.json
	npm ci --no-audit --no-fund

dist/extension.js: src/extension.ts node_modules/.package-lock.json
	npm run bundle
