# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.22.4 AS base
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

FROM --platform=$BUILDPLATFORM base AS download
WORKDIR /build
COPY gen/go.mod gen/go.sum ./gen/
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY cli/go.mod cli/go.sum ./cli/

WORKDIR /build/cli
RUN go mod download

FROM --platform=$BUILDPLATFORM download AS build
WORKDIR /build
COPY gen/ ./gen/
COPY pkg/ ./pkg/
COPY cli/ ./cli/

WORKDIR /build/cli/uml2pcl
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/uml2pcl

FROM --platform=$BUILDPLATFORM ubuntu:noble-20240605 AS test
COPY --from=build /out/uml2pcl .

FROM --platform=$BUILDPLATFORM scratch
COPY --from=build /out/uml2pcl /
ENTRYPOINT ["/uml2pcl"]
