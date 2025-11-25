# Project variables
BINARY=gh-sync
IMAGE_NAME=gh-sync-go
REGISTRY=github.com/faildruid/gh-sync-go
CONFIG=config.yaml

.PHONY: build
build:
	@echo "Building $(BINARY)..."
	go build -o $(BINARY) ./cmd/gh-sync

.PHONY: test
test:
	@echo "Running go tests..."
	go test ./...

.PHONY: tidy
tidy:
	@echo "Running go mod tidy..."
	go mod tidy

.PHONY: coverage
coverage:
	@echo "Running tests with coverage..."
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY) coverage.out
	go clean ./...

.PHONY: watch
watch:
	@echo "Watching for file changes and running tests. Requires entr."
	find . -name '*.go' | entr -r go test ./...

.PHONY: dev
dev:
	@echo "Running $(BINARY) in development mode..."
	go run ./cmd/gh-sync -c $(CONFIG)

.PHONY: docker-build
docker-build:
	@echo "Building Docker image $(IMAGE_NAME)..."
	docker build -t $(IMAGE_NAME) .

.PHONY: docker-run
docker-run:
	@echo "Running Docker image $(IMAGE_NAME)..."
	@if [ ! -f "$(CONFIG)" ]; then \
		echo "Config $(CONFIG) not found"; \
		exit 1; \
	fi
	docker run --rm \
		-v $$PWD/$(CONFIG):/config/config.yaml \
		-v $$PWD/keys:/keys \
		$(IMAGE_NAME) \
		-c /config/config.yaml

.PHONY: publish
publish:
	@echo "Tagging and pushing Docker image to $(REGISTRY)..."
	docker tag $(IMAGE_NAME) $(REGISTRY)
	docker push $(REGISTRY)
