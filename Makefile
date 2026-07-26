# ------------------------------------------------------------------
# Eumel — Go + HTMX template
# ------------------------------------------------------------------

TAILWIND_VERSION := v4.1.11
BIN_DIR          := bin
TAILWIND         := $(BIN_DIR)/tailwindcss

UNAME_S := $(shell uname -s | tr A-Z a-z)
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),x86_64)
	ARCH := x64
else
	ARCH := arm64
endif
ifeq ($(UNAME_S),darwin)
	TAILWIND_TARGET := macos-$(ARCH)
else
	TAILWIND_TARGET := linux-$(ARCH)
endif

.PHONY: help dev run build export css css-watch tailwind htmx test tidy clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

dev: css ## Build CSS and run the server in dev mode (live template reload)
	ENV=dev go run ./cmd/server

run: ## Run the server (prod mode, embedded assets)
	ENV=prod go run ./cmd/server

build: css ## Build CSS and compile a self-contained production binary
	go build -o $(BIN_DIR)/server ./cmd/server

export: css ## Pre-render the whole site as static files into public/
	go run ./cmd/export

css: $(TAILWIND) ## Build Tailwind CSS once
	$(TAILWIND) -i web/static/css/input.css -o web/static/css/output.css --minify

css-watch: $(TAILWIND) ## Rebuild Tailwind CSS on every template change
	$(TAILWIND) -i web/static/css/input.css -o web/static/css/output.css --watch

tailwind: $(TAILWIND) ## Download the Tailwind standalone binary

$(TAILWIND):
	mkdir -p $(BIN_DIR)
	curl -fsSL -o $(TAILWIND) https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-$(TAILWIND_TARGET)
	chmod +x $(TAILWIND)

htmx: ## Re-download / update the vendored htmx library
	curl -fsSL -o web/static/js/htmx.min.js https://unpkg.com/htmx.org@2/dist/htmx.min.js

test: ## Run all tests
	go test ./...

tidy: ## go mod tidy + gofmt
	go mod tidy
	gofmt -w .

clean: ## Remove build artifacts and the local database
	rm -rf $(BIN_DIR) data
