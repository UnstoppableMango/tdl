# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM mcr.microsoft.com/dotnet/sdk:9.0.100-preview.4 AS base
ARG BUILDPLATFORM
ARG TARGETARCH
ARG TARGETOS

FROM --platform=$BUILDPLATFORM base AS build
WORKDIR /build
COPY src/Abstractions/UnMango.Tdl.Abstractions.csproj ./Abstractions/
COPY src/FSharp/UnMango.Tdl.FSharp.fsproj ./FSharp/
COPY src/Broker/UnMango.Tdl.Broker.csproj ./Broker/

RUN dotnet restore \
		--arch $TARGETARCH \
		Broker/UnMango.Tdl.Broker.csproj

COPY proto ./proto
COPY Directory.Build.props .
COPY src/Abstractions/ ./Abstractions/
COPY src/FSharp/ ./FSharp/
COPY src/Broker/ ./Broker/

RUN MINVERDISABLE=true \
	dotnet publish \
		--no-restore \
		--os $TARGETOS \
		--arch $TARGETARCH \
		--configuration Release \
		--output /out \
		/p:PublishAot=false \
		Broker/UnMango.Tdl.Broker.csproj

FROM --platform=$BUILDPLATFORM mcr.microsoft.com/dotnet/aspnet:9.0.0-preview.4 AS final

RUN addgroup tdl && useradd -G tdl broker

WORKDIR /app
COPY --from=build /out/ ./

RUN mkdir /var/run/tdl /app/plugins && \
	chown broker:tdl -R /var/run/tdl /app

USER broker:tdl
VOLUME /var/run/tdl
VOLUME /app/plugins

ENTRYPOINT ["dotnet", "UnMango.Tdl.Broker.dll"]
