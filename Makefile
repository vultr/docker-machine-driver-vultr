SOURCES := $(shell find . -name '*.go')
PKG := $(shell go list ./machine/)
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
BINARY := docker-machine-driver-vultr

BUILD=`date +%FT%T%z`
PLATFORM=`uname`

LDFLAGS=-ldflags "-w -s"

build: docker-machine-driver-vultr

test: $(SOURCES)
	go test -v -short -race -timeout 30s ./...

clean:
	@rm -rf build/$(BINARY)

local:
	CGO_ENABLED=0 go build -o /usr/local/bin/$(BINARY) -${LDFLAGS} machine/main.go

check: ## Static Check Golang files
	@staticcheck ./...

vet: ## go vet files
	@go vet ./...

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/$(BINARY) -${LDFLAGS} machine/main.go