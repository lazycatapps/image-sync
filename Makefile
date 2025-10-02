.PHONY: help build build-backend build-backend-dev build-dist clean-dist build-lpk dev-backend dev-frontend build-local test test-coverage fmt vet lint tidy check audit clean push deploy-prod deploy-dev-full deploy-backend-dev deploy-frontend deploy-lpk deploy all

# Default registry and image names
LAZYCAT_BOX_NAME ?= mybox
REGISTRY ?= docker-registry-ui.${LAZYCAT_BOX_NAME}.heiyu.space
BACKEND_IMAGE ?= $(REGISTRY)/image-sync/backend
VERSION ?= latest
PLATFORM ?= linux/amd64
GOOS ?= linux
GOARCH ?= amd64

# Environment variables for default registries
DEFAULT_SOURCE_REGISTRY ?= registry.lazycat.cloud/
DEFAULT_DEST_REGISTRY ?= registry.${LAZYCAT_BOX_NAME}.heiyu.space/

HELP_FUN = \
	%help; while(<>){push@{$$help{$$2//'options'}},[$$1,$$3] \
	if/^([\w-_]+)\s*:.*\#\#(?:@(\w+))?\s(.*)$$/}; \
	print"\033[1m$$_:\033[0m\n", map"  \033[36m$$_->[0]\033[0m".(" "x(20-length($$_->[0])))."$$_->[1]\n",\
	@{$$help{$$_}},"\n" for keys %help; \

help: ##@General Show this help
	@echo -e "Usage: make \033[36m<target>\033[0m\n"
	@perl -e '$(HELP_FUN)' $(MAKEFILE_LIST)

dev-backend: ##@Development Run backend locally for development
	cd backend && \
	export DEFAULT_SOURCE_REGISTRY="$(DEFAULT_SOURCE_REGISTRY)" && \
	export DEFAULT_DEST_REGISTRY="$(DEFAULT_DEST_REGISTRY)" && \
	go run cmd/server/main.go

dev-frontend: ##@Development Run frontend locally for development
	cd frontend && npm start

test: ##@Development Run backend tests
	cd backend && go test -v ./...

test-coverage: ##@Development Run backend tests with coverage
	cd backend && go test -v -coverprofile=coverage.out ./...
	cd backend && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: backend/coverage.html"

fmt: ##@Development Format Go code
	cd backend && go fmt ./...

vet: ##@Development Run go vet
	cd backend && go vet ./...

lint: fmt vet ##@Development Run fmt and vet

tidy: ##@Development Run go mod tidy
	cd backend && go mod tidy

check: lint test ##@Development Run lint and tests

audit: ##@Development Scan frontend for security vulnerabilities
	@echo "Scanning frontend dependencies for vulnerabilities..."
	@CURRENT_REGISTRY=$$(npm config get registry); \
	npm config set registry https://registry.npmjs.org/; \
	cd frontend && npm audit; \
	AUDIT_EXIT=$$?; \
	npm config set registry $$CURRENT_REGISTRY; \
	exit $$AUDIT_EXIT

build: build-backend build-dist ##@Build Build backend image and frontend dist

build-backend: ##@Build Build backend Docker image (production)
	@echo "Building backend image: $(BACKEND_IMAGE):$(VERSION) for platform $(PLATFORM)"
	cd backend && docker build --platform $(PLATFORM) -t $(BACKEND_IMAGE):$(VERSION) .
	@echo "Backend image built successfully!"

build-backend-dev: build-local ##@Build Build backend Docker image (development with local binary)
	@echo "Building backend dev image: $(BACKEND_IMAGE):${VERSION} for platform $(PLATFORM)"
	cd backend && docker build --platform $(PLATFORM) -f Dockerfile.dev -t $(BACKEND_IMAGE):${VERSION} .
	@echo "Backend dev image built successfully!"

build-dist: ##@Build Build frontend into dist directory
	@echo "Building frontend to dist directory..."
	sh build.sh
	@echo "Frontend dist built successfully!"

build-lpk: ##@Build Build LPK package (requires lzc-cli)
	@echo "Building LPK package..."
	lzc-cli project build
	@echo "LPK package built successfully!"

build-local: ##@Build Build backend binary locally
	cd backend && CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o image-sync-server cmd/server/main.go

clean: ##@Maintenance Clean dist directory and backend binary
	@echo "Cleaning dist directory..."
	-rm -rf ./dist
	@echo "Cleaning backend binary..."
	-cd backend && rm -f image-sync-server
	@echo "Clean completed!"

clean-dist: ##@Maintenance Clean dist directory only
	@echo "Cleaning dist directory..."
	-rm -rf ./dist

push: ##@Maintenance Push backend image to registry
	@echo "Pushing backend image..."
	docker push $(BACKEND_IMAGE):$(VERSION)
	@echo "Backend image pushed successfully!"

deploy-prod: build-backend push build-dist deploy-lpk ##@Deploy Production deployment (backend prod + frontend + lpk)

deploy-dev-full: build-backend-dev push build-dist deploy-lpk ##@Deploy Development full deployment (backend dev + frontend + lpk)

deploy-backend-dev: build-backend-dev push deploy-lpk ##@Deploy Deploy backend dev only (backend dev + lpk)

deploy-frontend: build-dist deploy-lpk ##@Deploy Deploy frontend only (frontend + lpk)

deploy-lpk: build-lpk ##@Deploy Deploy LPK only (no rebuild frontend/backend)
	@echo "Deploying LPK package..."
	@LPK_FILE=$$(ls -t *-v*.lpk 2>/dev/null | head -n 1); \
	if [ -z "$$LPK_FILE" ]; then \
		echo "Error: No LPK file found"; \
		exit 1; \
	fi; \
	echo "Installing $$LPK_FILE..."; \
	lzc-cli app install "$$LPK_FILE"
	@echo "Deployment completed!"

deploy: deploy-prod ##@Aliases Alias for deploy-prod (backward compatibility)

all: deploy-prod ##@Aliases Alias for deploy-prod (default target)
