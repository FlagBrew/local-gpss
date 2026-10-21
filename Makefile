# Local GPSS is two programs: the Go server (local-gpss) and a .NET helper
# (GpssConsole) that it shells out to at ./bin/GpssConsole. The layout produced
# here matches the release zips: local-gpss in the repo root, GpssConsole in bin/.
#
#   make                    build both for this machine
#   make RID=linux-arm64    cross-build (linux-x64, linux-arm64, osx-x64, osx-arm64, win-x64)
#   make run                build and start the server
#   make dist               zip a release bundle into dist/

GO     ?= go
DOTNET ?= dotnet

ifeq ($(OS),Windows_NT)
  HOST_OS   := win
  HOST_ARCH := $(if $(filter ARM64,$(PROCESSOR_ARCHITECTURE)),arm64,x64)
else
  HOST_OS   := $(if $(filter Darwin,$(shell uname -s)),osx,linux)
  HOST_ARCH := $(if $(filter arm64 aarch64,$(shell uname -m)),arm64,x64)
endif

# .NET runtime identifier; the Go target is derived from it.
RID ?= $(HOST_OS)-$(HOST_ARCH)

RID_OS   := $(word 1,$(subst -, ,$(RID)))
RID_ARCH := $(word 2,$(subst -, ,$(RID)))
GOOS     := $(patsubst osx,darwin,$(patsubst win,windows,$(RID_OS)))
GOARCH   := $(patsubst x64,amd64,$(RID_ARCH))
EXE      := $(if $(filter windows,$(GOOS)),.exe)

SERVER  := local-gpss$(EXE)
CONSOLE := bin/GpssConsole$(EXE)

# The SQLite driver is pure Go, so cgo stays off and the binary is static.
GO_FLAGS := -trimpath -buildvcs=false -tags=netgo,osusergo -ldflags '-s -w'

DOTNET_FLAGS := -c Release -r $(RID) --self-contained true \
	-p:PublishReadyToRun=true -p:PublishSingleFile=true \
	-p:EnableCompressionInSingleFile=true

.PHONY: all build server console run generate docker dist clean

all: build

build: server console

server:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(GO_FLAGS) -o $(SERVER) .

console:
	$(DOTNET) publish GpssConsole/GpssConsole.csproj $(DOTNET_FLAGS) -o bin

run: build
	./$(SERVER)

# Regenerate the ent database code after editing internal/database/ent/schema.
generate:
	$(GO) generate ./internal/database/ent

docker:
	docker build -t local-gpss .

dist: build
	mkdir -p dist
	rm -f dist/local-gpss-$(RID).zip
	zip -r dist/local-gpss-$(RID).zip $(SERVER) bin

clean:
	rm -rf local-gpss local-gpss.exe bin dist GpssConsole/bin GpssConsole/obj GpssConsole/output
