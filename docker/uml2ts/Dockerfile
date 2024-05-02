FROM oven/bun:1 AS base
ARG BUILDPLATFORM
WORKDIR /build

FROM base AS install

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

RUN bun run build

FROM ubuntu:noble-20240429 as test
COPY --from=build /build/dist/uml2ts .

FROM --platform=$BUILDPLATFORM scratch as dist
COPY --from=build /build/dist/uml2ts .
ENTRYPOINT ["uml2ts"]
