APP       := go-k8s-healthcheck
IMAGE     := ghcr.io/plabs/$(APP)
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
NAMESPACE ?= healthcheck

.PHONY: help build test lint docker-build docker-push \
        deploy-dev deploy-prod diff-dev diff-prod clean

## help: show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //'

## build: compile binary locally
build:
	@echo "→ Building $(APP) $(VERSION)"
	CGO_ENABLED=0 go build -trimpath \
		-ldflags="-s -w" \
		-o bin/server ./cmd/server

## test: run all tests
test:
	go test -race -count=1 ./...

## lint: run static analysis (requires golangci-lint)
lint:
	golangci-lint run ./...

## run: run locally (hot-reload friendly)
run:
	PORT=8080 APP_ENV=development LOG_LEVEL=debug go run ./cmd/server

## docker-build: build Docker image
docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		-t $(IMAGE):$(VERSION) \
		-t $(IMAGE):latest \
		.

## docker-push: push Docker image to registry
docker-push:
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE):latest

## deploy-dev: apply dev overlay via Kustomize
deploy-dev:
	kubectl apply -k k8s/overlays/dev

## deploy-prod: apply prod overlay via Kustomize
deploy-prod:
	kubectl apply -k k8s/overlays/prod

## diff-dev: dry-run diff for dev overlay
diff-dev:
	kubectl diff -k k8s/overlays/dev

## diff-prod: dry-run diff for prod overlay
diff-prod:
	kubectl diff -k k8s/overlays/prod

## clean: remove build artifacts
clean:
	rm -rf bin/
