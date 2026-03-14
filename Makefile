.PHONY: build test lint fmt quality

# 全ツールのディレクトリを自動検出
TOOLS := $(wildcard cmd/gf-*)

build:
	@for tool in $(TOOLS); do \
		echo "Building $$tool..."; \
		(cd $$tool && go build -o $$(basename $$tool) .); \
	done

test:
	@for tool in $(TOOLS); do \
		echo "Testing $$tool..."; \
		(cd $$tool && go test -v ./...); \
	done

lint:
	@for tool in $(TOOLS); do \
		echo "Linting $$tool..."; \
		(cd $$tool && golangci-lint run --config ../../.golangci.yml ./...); \
	done

fmt:
	@for tool in $(TOOLS); do \
		(cd $$tool && gofmt -l .); \
	done

quality: test lint
	@echo "All quality checks passed."
