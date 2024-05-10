# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM mcr.microsoft.com/dotnet/sdk:9.0.100-preview.3 AS base
ARG BUILDPLATFORM
ARG TARGETARCH
ARG TARGETOS

FROM base AS build
WORKDIR /build
COPY src/Broker/UnMango.Tdl.Broker.csproj ./Broker/

RUN dotnet restore \
		--arch $TARGETARCH \
		Broker/UnMango.Tdl.Broker.csproj

COPY proto ./proto
COPY Directory.Build.props .
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

FROM --platform=$BUILDPLATFORM mcr.microsoft.com/dotnet/aspnet:9.0.0-preview.3 AS final

RUN addgroup tdl && useradd -G tdl broker

WORKDIR /app
COPY --from=build /out/ ./
RUN chown broker:tdl -R .

USER broker
ENTRYPOINT ["dotnet", "UnMango.Tdl.Broker.dll"]
