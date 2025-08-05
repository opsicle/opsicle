BASE_NAME := opsicle

# ___.         .__.__       .___
# \_ |__  __ __|__|  |    __| _/
#  | __ \|  |  \  |  |   / __ | 
#  | \_\ \  |  /  |  |__/ /_/ | 
#  |___  /____/|__|____/\____ | 
#      \/                    \/ 

OUTPUT_DIR := bin
GOOS ?= linux
GOARCH ?= amd64
LDFLAGS := -s -w -extldflags "-static"
SHA_FILE := $(OUTPUT_DIR)/$(BINARY_NAME).sha256

-include Makefile.properties

BINARY_NAME := $(BASE_NAME)_$(GOOS)_$(GOARCH)
SHA_FILE := $(OUTPUT_DIR)/$(BINARY_NAME).sha256
TARGETS := \
	linux_amd64 \
	darwin_amd64 \
	darwin_arm64 \
	windows_amd64
.PHONY: build build_debug build_all clean

build:
	@echo "Building production binary: $(BASE_NAME)_$(GOOS)_$(GOARCH)"
	@EXT=""
	@if [ "$(GOOS)" = "windows" ]; then EXT=".exe"; fi; \
	OUT_FILE="$(OUTPUT_DIR)/$(BASE_NAME)_$(GOOS)_$(GOARCH)$$EXT"; \
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build \
		-a -installsuffix cgo \
		-ldflags '$(LDFLAGS)' \
		-o $$OUT_FILE main.go
	@echo "Generating SHA256 checksum..."
	sha256sum $(OUTPUT_DIR)/$(BINARY_NAME) > $(SHA_FILE)

build_debug:
	@echo "Building debug binary: $(BASE_NAME)_$(GOOS)_$(GOARCH)_debug"
	@EXT=""
	@if [ "$(GOOS)" = "windows" ]; then EXT=".exe"; fi; \
	OUT_FILE="$(OUTPUT_DIR)/$(BASE_NAME)_$(GOOS)_$(GOARCH)_debug$$EXT"; \
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build \
		-gcflags "all=-N -l" \
		-o $$OUTFILE main.go

build_all:
	@mkdir -p $(OUTPUT_DIR)
	@for target in $(TARGETS); do \
		OS=$${target%_*}; \
		ARCH=$${target#*_}; \
		echo "Building for $$OS/$$ARCH..."; \
		$(MAKE) build GOOS=$$OS GOARCH=$$ARCH; \
		$(MAKE) checksum GOOS=$$OS GOARCH=$$ARCH; \
	done

clean:
	@echo "Cleaning up..."
	rm -rf $(OUTPUT_DIR)/*

#     .___                 .__                 
#   __| _/_______  __ ____ |  |   ____ ______  
#  / __ |/ __ \  \/ // __ \|  |  /  _ \\____ \ 
# / /_/ \  ___/\   /\  ___/|  |_(  <_> )  |_> >
# \____ |\___  >\_/  \___  >____/\____/|   __/ 
#      \/    \/          \/            |__|    

.PHONY: deps compose_up compose_down kind_up kind_down

deps:
	go mod tidy && go mod vendor

compose_up:
	docker-compose up -d

compose_down:
	docker-compose down

install_local: build
	@EXT=""
	@if [ "$(GOOS)" = "windows" ]; then EXT=".exe"; fi; \
		OUT_FILE="$(OUTPUT_DIR)/$(BASE_NAME)_$(GOOS)_$(GOARCH)$$EXT"; \
		sudo ln -sf $$(pwd)/$$OUT_FILE /usr/bin/$(BASE_NAME)

KIND_CLUSTER_NAME := opsicle-dev

kind_up:
	kind create cluster --name $(KIND_CLUSTER_NAME)

kind_down:
	kind delete cluster --name $(KIND_CLUSTER_NAME)

docs_approver:
	swag init \
		--tags approver-service \
		--output ./internal/approver/docs \
		--parseDependencyLevel 1
	sed -i 's|swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)||g' ./internal/approver/docs/docs.go

docs_controller:
	swag init \
		--tags controller-service \
		--output ./internal/controller/docs \
		--parseDependencyLevel 1
	sed -i 's|swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)||g' ./internal/controller/docs/docs.go

migration:
	migrate create -dir ./internal/database/migrations -ext sql new

mysql_shell:
	@mysql -uopsicle -h127.0.0.1 -P3306 -ppassword opsicle

mysql_reset:
	@mysql -uopsicle -h127.0.0.1 -P3306 -ppassword -e 'DROP SCHEMA `opsicle`'
	@mysql -uopsicle -h127.0.0.1 -P3306 -ppassword -e 'CREATE SCHEMA `opsicle`'
	@go run . run migrations

#                .__                               
# _______   ____ |  |   ____ _____    ______ ____  
# \_  __ \_/ __ \|  | _/ __ \\__  \  /  ___// __ \ 
#  |  | \/\  ___/|  |_\  ___/ / __ \_\___ \\  ___/ 
#  |__|    \___  >____/\___  >____  /____  >\___  >
#              \/          \/     \/     \/     \/ 

IMAGE_REGISTRY ?= docker.io
IMAGE_NAME := opsicle
IMAGE_TAG ?= latest
FULL_IMAGE := $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
FULL_IMAGE_DEBUG := $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)-debug

.PHONY: image image_debug docker_push docker_clean

# Build the production image (scratch-based)
image:
	docker build --target production -t $(IMAGE_NAME):$(IMAGE_TAG) .

# Build the debug image (with curl, ping, etc.)
image_debug:
	docker build --target debugger -t $(IMAGE_NAME):$(IMAGE_TAG)-debug .

# Push both images to registry
docker_push:
	docker push $(FULL_IMAGE)
	docker push $(FULL_IMAGE_DEBUG)

# Remove local images
docker_clean:
	docker rmi -f $(FULL_IMAGE) || true
	docker rmi -f $(FULL_IMAGE_DEBUG) || true

#     .___            .__                
#   __| _/____ ______ |  |   ____ ___.__.
#  / __ |/ __ \\____ \|  |  /  _ <   |  |
# / /_/ \  ___/|  |_> >  |_(  <_> )___  |
# \____ |\___  >   __/|____/\____// ____|
#      \/    \/|__|               \/     

HELM_RELEASE_NAME := opsicle
HELM_CHART_PATH := charts/opsicle
NAMESPACE := opsicle
.PHONY: deploy_helm

deploy_helm:
	helm upgrade --install \
		--namespace $(NAMESPACE) \
		--create-namespace \
		$(HELM_RELEASE_NAME) \
		$(HELM_CHART_PATH)
