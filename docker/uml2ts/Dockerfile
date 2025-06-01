# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM oven/bun:1.2.15 AS base
ARG BUILDPLATFORM
WORKDIR /build

FROM --platform=$BUILDPLATFORM base AS install

RUN mkdir -p gen/proto/{es,ts} packages/{ts,uml,uml2ts}

# Need everything because the lockfile references everything
COPY package.json .
COPY packages/tdl/package.json packages/tdl/
COPY packages/ts/package.json packages/ts/
COPY packages/uml2ts/package.json packages/uml2ts/
COPY packages/zod/package.json packages/zod/
COPY packages/zod2uml/package.json packages/zod2uml/
COPY bun.lockb .

RUN bun install --frozen-lockfile --production

FROM --platform=$BUILDPLATFORM install AS build
COPY --from=install /build/node_modules .

COPY packages/tdl/ packages/tdl/
COPY packages/ts/ packages/ts/
COPY packages/uml2ts/ .

RUN bun build \
	--compile \
	--minify \
	--sourcemap ./index.ts \
	--outfile ./dist/uml2ts

FROM --platform=$BUILDPLATFORM ubuntu:noble-20241118.1 AS test
COPY --from=build /build/dist/uml2ts .

FROM --platform=$BUILDPLATFORM oven/bun:1.2.2-distroless
COPY --from=build /build/dist/uml2ts /bin/
ENTRYPOINT ["/bin/uml2ts"]
