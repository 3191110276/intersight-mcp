GOCACHE ?= $(CURDIR)/.cache/go-build
GOTMPDIR ?= $(CURDIR)/.tmp
BUILD_TARGETS ?= darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64
ALL_PROVIDERS := $(sort $(patsubst %-mcp,%,$(notdir $(patsubst %/,%,$(wildcard cmd/*-mcp/)))))
PROVIDERS ?= intersight
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS ?= -X main.version=$(VERSION)

.PHONY: generate build verify list-providers

list-providers:
	@printf '%s\n' $(ALL_PROVIDERS)

generate:
	mkdir -p $(GOCACHE) $(GOTMPDIR)
	GOCACHE=$(GOCACHE) GOTMPDIR=$(GOTMPDIR) go generate ./...

build:
	mkdir -p bin $(GOCACHE) $(GOTMPDIR)
	@for provider in $(PROVIDERS); do \
		cmd=./cmd/$${provider}-mcp; \
		if [ ! -d $$cmd ]; then echo "missing command for provider $$provider: $$cmd"; exit 1; fi; \
		for target in $(BUILD_TARGETS); do \
			os=$${target%/*}; \
			arch=$${target#*/}; \
			output=bin/$${provider}-mcp-$${os}-$${arch}; \
			if [ "$$os" = "windows" ]; then output=$${output}.exe; fi; \
			echo "building $$output (version $(VERSION))"; \
			GOOS=$$os GOARCH=$$arch GOCACHE=$(GOCACHE) GOTMPDIR=$(GOTMPDIR) go build -ldflags "$(LDFLAGS)" -o $$output $$cmd || exit 1; \
		done; \
	done

verify:
	$(MAKE) generate
	GOCACHE=$(GOCACHE) GOTMPDIR=$(GOTMPDIR) go test ./...
	$(MAKE) build
