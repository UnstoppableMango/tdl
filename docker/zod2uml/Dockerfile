# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM oven/bun:1.1.45 AS base
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
COPY packages/zod/ packages/zod/
COPY packages/zod2uml/ .

RUN bun build \
	--compile \
	--minify \
	--sourcemap ./index.ts \
	--outfile ./dist/zod2uml

FROM --platform=$BUILDPLATFORM ubuntu:noble-20241015 AS test
COPY --from=build /build/dist/zod2uml .

FROM --platform=$BUILDPLATFORM oven/bun:1.1.45-distroless
COPY --from=build /build/dist/zod2uml /bin/
ENTRYPOINT ["/bin/zod2uml"]
