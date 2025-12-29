FROM mcr.microsoft.com/dotnet/sdk:10.0 AS dotnet-build
WORKDIR /app

COPY ./GpssConsole .

RUN dotnet publish GpssConsole.csproj -c Release -o ./output/ \
    --self-contained true \
    -p:PublishReadyToRun=true -p:PublishSingleFile=true \
    -p:EnableCompressionInSingleFile=true

FROM golang:alpine AS go-build

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 go build \
    -ldflags '-d -s -w -extldflags=-static' \
    -tags=netgo,osusergo,static_build \
    -installsuffix netgo \
    -buildvcs=false \
    -trimpath \
    -o local-gpss

FROM alpine:latest AS runner

WORKDIR /app

RUN mkdir bin

COPY --from=dotnet-build /app/output/GpssConsole ./bin/GpssConsole
COPY --from=go-build /app/local-gpss ./local-gpss

RUN echo "MODE=docker" > .env

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
CMD ["/app/local-gpss"]
