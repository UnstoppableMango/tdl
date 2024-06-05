# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM oven/bun:1.1.12 AS base
ARG BUILDPLATFORM
WORKDIR /build

FROM --platform=$BUILDPLATFORM base AS install

RUN mkdir -p gen/proto/{es,ts} packages/{ts,uml,uml2ts}

COPY package.json .
COPY gen/proto/es/package.json gen/proto/es/
COPY gen/proto/ts/package.json gen/proto/ts/
COPY packages/ts/package.json packages/ts/
COPY packages/uml/package.json packages/uml/
COPY packages/uml2ts/package.json packages/uml2ts/
COPY bun.lockb .

RUN bun install --frozen-lockfile --production

FROM --platform=$BUILDPLATFORM install AS build
COPY --from=install /build/node_modules .

COPY gen/proto/es/ gen/proto/es/
COPY gen/proto/ts/ gen/proto/ts/
COPY packages/ts/ packages/ts/
COPY packages/uml/ packages/uml/
COPY packages/uml2ts/ .

RUN bun build \
			--compile \
			--minify \
			--sourcemap ./index.ts \
			--outfile ./dist/uml2ts

FROM --platform=$BUILDPLATFORM ubuntu:noble-20240530 AS test
COPY --from=build /build/dist/uml2ts .

FROM --platform=$BUILDPLATFORM oven/bun:1.1.12-distroless
COPY --from=build /build/dist/uml2ts /
ENTRYPOINT ["/uml2ts"]
