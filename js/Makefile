.PHONY: test check

PACKAGES := package.json package-lock.json $(wildcard */package.json)

# The same compiled tests under both runtimes.
test: node_modules/.package-lock.json
	npm test
	npm run test:bun

check: node_modules/.package-lock.json
	npm run build
	npm run check

# npm writes this hidden lockfile on every install.
node_modules/.package-lock.json: ${PACKAGES}
	npm ci --no-audit --no-fund
