# base.mk - Common Makefile for LazyCAT Apps projects
# This file should be included in your project's Makefile
#
# Required tools:
#   lzc-cli         - LazyCAT CLI tool
#                     Install: npm install -g @lazycatcloud/lzc-cli
#                     Auto-completion: lzc-cli completion >> ~/.zshrc
#
# Required variables (define in your Makefile):
#   PROJECT_TYPE    - lpk-only or docker-lpk
#
# Optional variables:
#   PROJECT_NAME    - Project name (default: current directory name)
#   APP_ID_PREFIX   - Application ID prefix (default: cloud.lazycat.app.)
#   APP_NAME        - Application name (default: current directory name)
#   APP_ID          - Full Application ID (default: APP_ID_PREFIX + APP_NAME)
#   VERSION         - Project version (default: git tag or commit)
#   CLEAN_EXTRA_PATHS - Additional files or directories removed by clean-default
#
#   REGISTRY        - Docker registry (for docker-lpk projects)
#   IMAGE_NAME      - Docker image name (default: PROJECT_NAME)
#   PLATFORM        - Multi-arch build platform (default: linux/amd64)
#
#   DOCKER_BUILD_CONTEXT - Docker build context directory (default: .)
#   DOCKER_BUILD_FILE - Alternate Dockerfile path
#   DOCKER_BUILD_TARGET - Docker build target (for multi-stage Dockerfiles)
#   DOCKER_BUILD_PLATFORM - Docker build platform (passed via --platform)
#   DOCKER_BUILD_EXTRA_ARGS - Additional docker build arguments
#
#   GOOS            - GOOS used for local Go builds (default: linux)
#   GOARCH          - GOARCH used for local Go builds (default: amd64)
#   GO_MODULE_DIR   - Working directory for Go commands (default: .)
#   GO_PACKAGES     - Package pattern for go fmt/test/vet (default: ./...)
#   GO_TEST_COVERPROFILE - Coverage profile filename (default: coverage.out)
#   GO_TEST_COVERHTML - Coverage HTML report filename (default: coverage.html)

# base.mk metadata
BASE_MK_PATH := $(abspath $(lastword $(filter %base.mk,$(MAKEFILE_LIST))))
BASE_MK_VERSION := 2025-10-15 23:45:19
SYNC_TARGET ?= all
LAZYCLI_SYNC_SCRIPT_URL ?= https://raw.githubusercontent.com/lazycatapps/hack/main/scripts/lazycli.sh
LAZYCLI_LOCAL_SCRIPT ?= ../hack/scripts/lazycli.sh

# Configure a portable inline sed command so GNU sed (Linux) and BSD sed (macOS) both work.
SED_SUPPORTS_VERSION := $(shell sed --version >/dev/null 2>&1 && echo yes || echo no)
ifeq ($(SED_SUPPORTS_VERSION),yes)
SED_INPLACE := sed -i -e
else
SED_INPLACE := sed -i '' -e
endif

# Application ID configuration
ifndef APP_ID_PREFIX
    APP_ID_PREFIX := cloud.lazycat.app.
endif

ifndef PROJECT_NAME
    PROJECT_NAME := $(notdir $(CURDIR))
endif

ifndef APP_NAME
    APP_NAME := $(shell basename $(CURDIR))
endif

ifndef APP_ID
    APP_ID := $(APP_ID_PREFIX)$(APP_NAME)
endif

# LazyCAT Box configuration
LAZYCAT_BOX_FALLBACK ?= 0
ifndef LAZYCAT_BOX_NAME
    LAZYCAT_BOX_NAME := $(shell command -v lzc-cli >/dev/null 2>&1 && lzc-cli box default 2>/dev/null)
    ifeq ($(strip $(LAZYCAT_BOX_NAME)),)
        LAZYCAT_BOX_NAME := default
        LAZYCAT_BOX_FALLBACK := 1
    endif
endif

# Version detection
ifndef VERSION
	VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
endif

# Docker-related variables
ifndef REGISTRY
    ifeq ($(LAZYCAT_BOX_FALLBACK),1)
        REGISTRY :=
    else
        REGISTRY := docker-registry-ui.$(LAZYCAT_BOX_NAME).heiyu.space
    endif
endif

ifeq ($(LAZYCAT_BOX_FALLBACK),1)
$(warning LazyCAT box name was not auto-detected; install lzc-cli or set LAZYCAT_BOX_NAME/REGISTRY to avoid using the fallback settings.)
endif

ifdef REGISTRY
    IMAGE_PREFIX := $(REGISTRY)/
else
    IMAGE_PREFIX :=
endif

ifndef IMAGE_NAME
    IMAGE_NAME := $(PROJECT_NAME)
endif

FULL_IMAGE_NAME := $(IMAGE_PREFIX)$(IMAGE_NAME):$(VERSION)

PLATFORM ?= linux/amd64
GOOS ?= linux
GOARCH ?= amd64
GO_MODULE_DIR ?= .
GO_PACKAGES ?= ./...
GO_TEST_COVERPROFILE ?= coverage.out
GO_TEST_COVERHTML ?= coverage.html

# Colors for output
COLOR_RESET   := \033[0m
COLOR_INFO    := \033[34m
COLOR_SUCCESS := \033[32m
COLOR_WARNING := \033[33m
COLOR_ERROR   := \033[31m

define print_info
	printf "%b[INFO]%b %s\n" "$(COLOR_INFO)" "$(COLOR_RESET)" "$(1)"
endef

define print_success
	printf "%b[SUCCESS]%b %s\n" "$(COLOR_SUCCESS)" "$(COLOR_RESET)" "$(1)"
endef

define print_warning
	printf "%b[WARNING]%b %s\n" "$(COLOR_WARNING)" "$(COLOR_RESET)" "$(1)"
endef

define print_error
	printf "%b[ERROR]%b %s\n" "$(COLOR_ERROR)" "$(COLOR_RESET)" "$(1)"
endef

# Default target
.DEFAULT_GOAL := help

##@ General

.PHONY: help-default
help-default: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: base-mk-version
base-mk-version: ## Show the current base.mk version identifier
	@$(call print_info,base.mk version: $(BASE_MK_VERSION))

# base-mk-update-version uses the portable sed wrapper to avoid duplicating sed compatibility logic.
.PHONY: base-mk-update-version
base-mk-update-version: base-mk-version ## Update base.mk version identifier to the current timestamp
	@NEW_VERSION=$$(date "+%Y-%m-%d %H:%M:%S"); \
	printf "%b[INFO]%b Updating base.mk version to %s\n" "$(COLOR_INFO)" "$(COLOR_RESET)" "$$NEW_VERSION"; \
	$(SED_INPLACE) "s/^BASE_MK_VERSION := .*/BASE_MK_VERSION := $${NEW_VERSION}/" "$(BASE_MK_PATH)"; \
	printf "%b[SUCCESS]%b base.mk version updated: %s\n" "$(COLOR_SUCCESS)" "$(COLOR_RESET)" "$$NEW_VERSION"

.PHONY: config-sync
config-sync: ## Synchronize project configuration via LazyCLI
	@SYNC_TARGET_VALUE="$${SYNC_TARGET:-$(SYNC_TARGET)}"; \
	LOCAL_SCRIPT_PATH="$(LAZYCLI_LOCAL_SCRIPT)"; \
	SYNC_PERFORMED=0; \
	$(call print_info,Synchronizing configuration with target $$SYNC_TARGET_VALUE); \
	if [ -n "$$LOCAL_SCRIPT_PATH" ] && [ -f "$$LOCAL_SCRIPT_PATH" ]; then \
		$(call print_info,Using local LazyCLI script at $$LOCAL_SCRIPT_PATH); \
		if "$$LOCAL_SCRIPT_PATH" --sync --sync-target "$$SYNC_TARGET_VALUE"; then \
			SYNC_PERFORMED=1; \
		else \
			exit $$?; \
		fi; \
	elif [ -n "$(LAZYCLI_SYNC_SCRIPT_URL)" ]; then \
		if ! command -v curl >/dev/null 2>&1; then \
			$(call print_error,curl not found. Please install curl or set LAZYCLI_LOCAL_SCRIPT); \
			exit 1; \
		fi; \
		$(call print_info,Using remote LazyCLI script: $(LAZYCLI_SYNC_SCRIPT_URL)); \
		if SYNC_TARGET_VALUE="$$SYNC_TARGET_VALUE" LAZYCLI_SYNC_SCRIPT_URL="$(LAZYCLI_SYNC_SCRIPT_URL)" bash -lc 'bash <(curl -fsSL "$${LAZYCLI_SYNC_SCRIPT_URL}") --sync --sync-target "$${SYNC_TARGET_VALUE}"'; then \
			SYNC_PERFORMED=1; \
		else \
			exit $$?; \
		fi; \
	else \
		$(call print_warning,Neither LAZYCLI_LOCAL_SCRIPT nor LAZYCLI_SYNC_SCRIPT_URL is available; skipping sync); \
	fi; \
	if [ "$$SYNC_PERFORMED" -eq 1 ]; then \
		$(call print_success,Configuration synchronized); \
	fi

.PHONY: info-default
info-default: ## Show project information
	@$(call print_info,Project: $(PROJECT_NAME))
	@$(call print_info,Type: $(PROJECT_TYPE))
	@$(call print_info,Version: $(VERSION))
	@$(call print_info,App ID: $(APP_ID))
ifeq ($(PROJECT_TYPE),docker-lpk)
	@$(call print_info,Image: $(FULL_IMAGE_NAME))
endif
ifeq ($(LAZYCAT_BOX_FALLBACK),1)
	@$(call print_warning,LazyCAT box name not detected. Install lzc-cli or set LAZYCAT_BOX_NAME/REGISTRY for Docker workflows.)
endif

##@ Building

CLEAN_EXTRA_PATHS ?=
DOCKER_BUILD_CONTEXT ?= .
DOCKER_BUILD_FILE ?=
DOCKER_BUILD_TARGET ?=
DOCKER_BUILD_PLATFORM ?=
DOCKER_BUILD_EXTRA_ARGS ?=

.PHONY: clean-default
clean-default: ## Clean build artifacts
	@$(call print_info,Cleaning build artifacts...)
	rm -rf bin/ dist/ build/ *.out coverage.html htmlcov/ *.lpk $(CLEAN_EXTRA_PATHS)
	@$(call print_success,Cleaned)

##@ Docker (docker-lpk projects only)

ifeq ($(PROJECT_TYPE),docker-lpk)

.PHONY: docker-build-default
docker-build-default: ## Build Docker image
	@$(call print_info,Building Docker image: $(FULL_IMAGE_NAME))
	docker build $(if $(DOCKER_BUILD_PLATFORM),--platform $(DOCKER_BUILD_PLATFORM)) $(if $(DOCKER_BUILD_FILE),-f $(DOCKER_BUILD_FILE)) $(if $(DOCKER_BUILD_TARGET),--target $(DOCKER_BUILD_TARGET)) $(DOCKER_BUILD_EXTRA_ARGS) -t $(FULL_IMAGE_NAME) $(DOCKER_BUILD_CONTEXT)
	@$(call print_success,Docker image built: $(FULL_IMAGE_NAME))

.PHONY: docker-push-default
docker-push-default: docker-build-default ## Push Docker image to registry
	@$(call print_info,Pushing Docker image: $(FULL_IMAGE_NAME))
	docker push $(FULL_IMAGE_NAME)
	@$(call print_success,Docker image pushed: $(FULL_IMAGE_NAME))

.PHONY: docker-run-default
docker-run-default: ## Run Docker container locally
	@$(call print_info,Running Docker container...)
	docker run --rm -it $(FULL_IMAGE_NAME)

endif

##@ Release

.PHONY: lpk-default
lpk-default: ## Package LPK
	@$(call print_info,Building LPK package...)
	@command -v lzc-cli >/dev/null 2>&1 || ($(call print_error,lzc-cli not found. Please install LazyCAT CLI) && exit 1)
	lzc-cli project build
	@$(call print_success,LPK package built successfully)

.PHONY: deploy-default
deploy-default: lpk-default ## Build and install LPK package
	@$(call print_info,Installing LPK package...)
	@LPK_FILE=$$(ls -t *.lpk 2>/dev/null | head -n 1); \
	if [ -z "$$LPK_FILE" ]; then \
		$(call print_error,No LPK file found); \
		exit 1; \
	fi; \
	echo "Installing $$LPK_FILE..."; \
	lzc-cli app install "$$LPK_FILE"
	@$(call print_success,Installation completed)

.PHONY: uninstall-default
uninstall-default: ## Uninstall the LPK package
	@$(call print_info,Uninstalling $(APP_ID)...)
	lzc-cli app uninstall $(APP_ID)
	@$(call print_success,Uninstallation completed)

.PHONY: list-packages-default
list-packages-default: ## List all LPK packages in current directory
	@echo "Available LPK packages:"
	@ls -lht *.lpk 2>/dev/null || echo "No LPK packages found"

.PHONY: all-default
all-default: help-default ## Default target: help

.PHONY: release-default
release-default: ## Create a release
ifeq ($(PROJECT_TYPE),docker-lpk)
	@$(MAKE) docker-push-default
endif
	@$(MAKE) lpk-default
	@$(call print_success,Release $(VERSION) completed)

##@ Go

.PHONY: fmt-default
fmt-default: ## Format Go code
	@$(call print_info,Formatting Go packages in $(GO_MODULE_DIR))
	cd $(GO_MODULE_DIR) && go fmt $(GO_PACKAGES)
	@$(call print_success,Go formatting completed)

.PHONY: vet-default
vet-default: ## Run go vet
	@$(call print_info,Running go vet in $(GO_MODULE_DIR))
	cd $(GO_MODULE_DIR) && go vet $(GO_PACKAGES)
	@$(call print_success,go vet completed)

.PHONY: test-default
test-default: ## Run Go tests
	@$(call print_info,Running Go tests in $(GO_MODULE_DIR))
	cd $(GO_MODULE_DIR) && go test -v $(GO_PACKAGES)
	@$(call print_success,Go tests completed)

.PHONY: test-coverage-default
test-coverage-default: ## Run Go tests with coverage report
	@$(call print_info,Running Go tests with coverage in $(GO_MODULE_DIR))
	cd $(GO_MODULE_DIR) && go test -v -coverprofile=$(GO_TEST_COVERPROFILE) $(GO_PACKAGES)
	cd $(GO_MODULE_DIR) && go tool cover -html=$(GO_TEST_COVERPROFILE) -o $(GO_TEST_COVERHTML)
	@$(call print_success,Coverage report generated: $(GO_MODULE_DIR)/$(GO_TEST_COVERHTML))

.PHONY: tidy-default
tidy-default: ## Run go mod tidy
	@$(call print_info,Running go mod tidy in $(GO_MODULE_DIR))
	cd $(GO_MODULE_DIR) && go mod tidy
	@$(call print_success,go mod tidy completed)

.PHONY: lint-default
lint-default: fmt-default vet-default ## Run formatting and vet

.PHONY: check-default
check-default: lint-default test-default ## Run lint and tests

##@ Utilities

.PHONY: install-lzc-cli-default
install-lzc-cli-default: ## Install lzc-cli tool
	@$(call print_info,Installing lzc-cli...)
	@command -v npm >/dev/null 2>&1 || ($(call print_error,npm not found. Please install Node.js first) && exit 1)
	npm install -g @lazycatcloud/lzc-cli
	@$(call print_success,lzc-cli installed successfully)
	@$(call print_info,To enable auto-completion, run: lzc-cli completion >> ~/.zshrc)

.PHONY: version-default
version-default: ## Show version
	@echo $(VERSION)

.PHONY: check-tools-default
check-tools-default: ## Check if required tools are installed
	@$(call print_info,Checking required tools...)
	@command -v lzc-cli >/dev/null 2>&1 || ($(call print_error,lzc-cli not found) && exit 1)
ifeq ($(PROJECT_TYPE),docker-lpk)
	@command -v docker >/dev/null 2>&1 || ($(call print_error,docker not found) && exit 1)
endif
	@$(call print_success,All required tools are installed)

# Pattern rule to allow overriding any -default target
# Usage: Define a target with the same name (without -default) in your Makefile to override
%: %-default
	@ true
