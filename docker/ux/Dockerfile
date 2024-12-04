# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.23.4 AS base
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

FROM --platform=$BUILDPLATFORM base AS download
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

FROM --platform=$BUILDPLATFORM download AS build
WORKDIR /build
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY cmd/ ./cmd/

WORKDIR /build/cmd/ux
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/ux

FROM --platform=$BUILDPLATFORM ubuntu:noble-20241015 AS test
COPY --from=build /out/ux .

FROM --platform=$BUILDPLATFORM scratch
COPY --from=build /out/ux /bin/
ENTRYPOINT ["/bin/ux"]
