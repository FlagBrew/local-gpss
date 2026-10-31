# Both build stages run on the build host's own architecture and cross-compile,
# so arm64 images don't have to be built under (slow) emulation.
FROM --platform=$BUILDPLATFORM mcr.microsoft.com/dotnet/sdk:10.0 AS dotnet-build
ARG TARGETARCH
WORKDIR /app

COPY ./GpssConsole .

RUN case "$TARGETARCH" in \
      amd64) RID=linux-x64 ;; \
      arm64) RID=linux-arm64 ;; \
      *) echo "unsupported arch: $TARGETARCH" >&2; exit 1 ;; \
    esac \
    && dotnet publish GpssConsole.csproj -c Release -o ./output/ -r "$RID" \
    --self-contained true \
    -p:PublishReadyToRun=true -p:PublishSingleFile=true \
    -p:EnableCompressionInSingleFile=true

FROM --platform=$BUILDPLATFORM golang:alpine AS go-build
ARG TARGETARCH

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build \
    -ldflags '-d -s -w -extldflags=-static' \
    -tags=netgo,osusergo,static_build \
    -installsuffix netgo \
    -buildvcs=false \
    -trimpath \
    -o local-gpss

FROM alpine:latest AS base

WORKDIR /app

RUN mkdir bin

COPY --from=dotnet-build /app/output/GpssConsole ./bin/GpssConsole
COPY --from=go-build /app/local-gpss ./local-gpss

RUN echo "MODE=docker" > .env
RUN apk add --no-cache gcompat libstdc++ libgcc icu-libs

# Somewhere to keep the sqlite database (point the config's connection string
# at /data/<file>) that can be mounted without hiding the binaries in /app.
RUN mkdir /data

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
CMD ["/app/local-gpss"]

# Same as the default image, but runs as an unprivileged user. The app writes
# config.json to its working directory and the database to /data, so both must
# be writable by that user (mounted volumes need to be too).
#
# OpenShift-compatible: it runs the container with a random UID and GID 0, so
# /app and /data are group-owned by root with group permissions equal to the
# owner's, and nothing relies on the user existing in /etc/passwd.
FROM base AS runner-nonroot

RUN chgrp -R 0 /app /data && chmod -R g=u /app /data
ENV HOME=/app \
    DOTNET_BUNDLE_EXTRACT_BASE_DIR=/tmp/.net
USER 10001:0

# Default image (root). Kept last so a plain `docker build` still produces it.
FROM base AS runner
