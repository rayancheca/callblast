.PHONY: all build go-build web-build test dev run clean

BINARY    := callblast
WEB_DIR   := web
DIST_DIR  := $(WEB_DIR)/dist
PORT      ?= 7332

all: build

# Full production build (frontend + binary)
build: web-build go-build

go-build:
	go build -o $(BINARY) ./cmd/callblast

web-build:
	cd $(WEB_DIR) && npm install --prefer-offline && npm run build

# Run all Go tests with race detector
test:
	go test -race -count=1 ./...

# Development: Go binary in background, Vite dev server in foreground
dev: go-build
	@echo "Starting callblast backend on port $(PORT)…"
	@./$(BINARY) --port $(PORT) --static "" &
	@echo "Starting Vite dev server…"
	cd $(WEB_DIR) && npm run dev

# Build then run
run: build
	./$(BINARY) --port $(PORT)

clean:
	rm -f $(BINARY)
	rm -rf $(DIST_DIR)
