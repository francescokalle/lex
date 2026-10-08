.PHONY: build test lint clean release

BINARY := lex
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/lex

test:
	go test -race -cover ./...

lint:
	golangci-lint run --timeout=5m

clean:
	rm -f $(BINARY)
	rm -f coverage.out
	rm -rf dist/

release: clean
	@mkdir -p dist
	@for os in linux darwin; do \
		for arch in amd64 arm64; do \
			output="dist/$(BINARY)-$(VERSION)-$$os-$$arch"; \
			echo "Building $$output..."; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
				go build -ldflags="$(LDFLAGS)" -o "$$output" ./cmd/lex; \
			sha256sum "$$output" > "$$output.sha256"; \
		done; \
	done
	@echo "Release artifacts in dist/"
