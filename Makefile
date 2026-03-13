.PHONY: build test quality

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

quality: test
	@for tool in $(TOOLS); do \
		echo "Vetting $$tool..."; \
		(cd $$tool && go vet ./...); \
	done
	@echo "All quality checks passed."
