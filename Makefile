GOCACHE ?= $(CURDIR)/.cache/go-build
GOTMPDIR ?= $(CURDIR)/.tmp
BUILD_TARGETS ?= darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS ?= -X main.version=$(VERSION)

.PHONY: generate build verify

generate:
	mkdir -p $(GOCACHE) $(GOTMPDIR)
	GOCACHE=$(GOCACHE) GOTMPDIR=$(GOTMPDIR) go generate ./...

build:
	mkdir -p bin $(GOCACHE) $(GOTMPDIR)
	@for target in $(BUILD_TARGETS); do \
		os=$${target%/*}; \
		arch=$${target#*/}; \
		output=bin/intersight-mcp-$${os}-$${arch}; \
		if [ "$$os" = "windows" ]; then output=$${output}.exe; fi; \
		echo "building $$output (version $(VERSION))"; \
		GOOS=$$os GOARCH=$$arch GOCACHE=$(GOCACHE) GOTMPDIR=$(GOTMPDIR) go build -ldflags "$(LDFLAGS)" -o $$output ./cmd/intersight-mcp || exit 1; \
	done

verify:
	$(MAKE) generate
	GOCACHE=$(GOCACHE) GOTMPDIR=$(GOTMPDIR) go test ./...
	$(MAKE) build
