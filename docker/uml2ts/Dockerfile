# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM oven/bun:1.1.22 AS base
ARG BUILDPLATFORM
WORKDIR /build

FROM --platform=$BUILDPLATFORM base AS install

RUN mkdir -p gen/proto/{es,ts} packages/{ts,uml,uml2ts}

# Need everything for the lockfile
COPY package.json .
COPY gen/proto/es/package.json gen/proto/es/
COPY gen/proto/ts/package.json gen/proto/ts/
COPY packages/echo/package.json packages/echo/
COPY packages/tdl/package.json packages/tdl/
COPY packages/ts/package.json packages/ts/
COPY packages/uml2ts/package.json packages/uml2ts/
COPY bun.lockb .

RUN bun install --frozen-lockfile --production

FROM --platform=$BUILDPLATFORM install AS build
COPY --from=install /build/node_modules .

COPY gen/proto/es/ gen/proto/es/
COPY packages/tdl/ packages/tdl/
COPY packages/ts/ packages/ts/
COPY packages/uml2ts/ .

RUN bun build \
		--compile \
		--minify \
		--sourcemap ./index.ts \
		--outfile ./dist/uml2ts

FROM --platform=$BUILDPLATFORM ubuntu:noble-20240605 AS test
COPY --from=build /build/dist/uml2ts .

FROM --platform=$BUILDPLATFORM oven/bun:1.1.22-distroless
COPY --from=build /build/dist/uml2ts /
ENTRYPOINT ["/uml2ts"]
