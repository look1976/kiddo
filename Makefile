.PHONY: deps build test clean run-debug install uninstall help

# Build output directory
OUTDIR := bin
BINARY := $(OUTDIR)/kiddo.exe

# Dependencies
help:
	@echo "Kiddo Service - Build Targets"
	@echo ""
	@echo "  make deps       - Download Go dependencies"
	@echo "  make build      - Build the service (Windows .exe)"
	@echo "  make test       - Run unit tests"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make run-debug  - Run service in debug/console mode (Windows only)"
	@echo ""

deps:
	@echo "[1/1] Downloading Go dependencies..."
	go mod download
	@echo "Dependencies ready"

build: deps
	@echo "[1/1] Building binary..."
	@mkdir -p $(OUTDIR)
	GOOS=windows GOARCH=amd64 go build -o $(BINARY) -ldflags="-s -w" .
	@echo "Build successful: $(BINARY)"

test: deps
	@echo "[1/1] Running tests..."
	go test -v -race ./...
	@echo "Tests complete"

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(OUTDIR)
	go clean
	@echo "Clean complete"

# Note: Install/uninstall require Windows
install: build
	@echo "Installing service..."
	$(BINARY) install
	@echo "Start the service with: net start Kiddo"

uninstall:
	@echo "Uninstalling service..."
	$(BINARY) uninstall
	@echo "Service uninstalled"

# Debug mode - runs service in console (helpful for testing on Windows)
run-debug: build
	@echo "Running service in debug mode (console)..."
	@echo "Press Ctrl+C to stop"
	$(BINARY)
