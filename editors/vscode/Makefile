.PHONY: check bundle install

check: node_modules/.package-lock.json
	npm run typecheck
	npm run check

bundle: dist/extension.js

install: dist/extension.js syntaxes/tdl.tmLanguage.json
	./install.sh

# npm writes this hidden lockfile on every install.
node_modules/.package-lock.json: package.json package-lock.json
	npm ci --no-audit --no-fund

dist/extension.js: src/extension.ts node_modules/.package-lock.json
	npm run bundle
