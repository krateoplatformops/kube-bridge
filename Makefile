# Set the shell to bash always
SHELL := /bin/bash

ORG_NAME := krateoplatformops
PROJECT_NAME := kube-bridge
VENDOR := Kiratech

# Github Container Registry
DOCKER_REGISTRY := ghcr.io/$(ORG_NAME)

#PLATFORM := "linux/amd64,linux/arm64,linux/arm"
PLATFORM := "linux/amd64"

# Tools
KIND=$(shell which kind)
LINT=$(shell which golangci-lint)
KUBECTL=$(shell which kubectl)
DOCKER=$(shell which docker)
HELM=$(shell which helm)
SED=$(shell which sed)

KIND_CLUSTER_NAME ?= local-dev
KUBECONFIG ?= $(HOME)/.kube/config

VERSION := $(shell git describe --always --tags | sed 's/-/./2' | sed 's/-/./2')
ifndef VERSION
VERSION := 0.0.0
endif

BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
REPO_URL := $(shell git config --get remote.origin.url | sed "s/git@/https\:\/\//; s/\.com\:/\.com\//; s/\.git//")
LAST_COMMIT := $(shell git log -1 --pretty=%h)

LD_FLAGS := -s -w -X "main.Version=$(VERSION)" -X "main.Build=$(LAST_COMMIT)"

UNAME := $(uname -s)

.PHONY: help
help:	### Show targets documentation
ifeq ($(UNAME), Linux)
	@grep -P '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
else
	@awk -F ':.*###' '$$0 ~ FS {printf "%15s%s\n", $$1 ":", $$2}' \
		$(MAKEFILE_LIST) | grep -v '@awk' | sort
endif

.PHONY: print.vars
print.vars: ### Print all the build variables
	@echo VENDOR=$(VENDOR)
	@echo ORG_NAME=$(ORG_NAME)
	@echo PROJECT_NAME=$(PROJECT_NAME)
	@echo REPO_URL=$(REPO_URL)
	@echo LAST_COMMIT=$(LAST_COMMIT)
	@echo VERSION=$(VERSION)
	@echo BUILD_DATE=$(BUILD_DATE)
	@echo PLATFORM=$(PLATFORM)
	@echo LD_FLAGS=$(LD_FLAGS)

.PHONY: kind.up
kind.up: ### Starts a KinD cluster for local development
	@$(KIND) get kubeconfig --name $(KIND_CLUSTER_NAME) >/dev/null 2>&1 || $(KIND) create cluster --name=$(KIND_CLUSTER_NAME)


.PHONY: kind.down
kind.down: ### Shuts down the KinD cluster
	@$(KIND) delete cluster --name=$(KIND_CLUSTER_NAME)

.PHONY: image.build
image.build: ### Build the Docker image
	echo "🏗    Building image '$(PROJECT_NAME):$(VERSION)' ..."
	@$(DOCKER) buildx create --name "$(PROJECT_NAME)" --use --append
	@$(DOCKER) buildx build --platform "$(PLATFORM)" --push -t "$(DOCKER_REGISTRY)/$(PROJECT_NAME):$(VERSION)" \
	--build-arg LD_FLAGS="$(LD_FLAGS)" \
	--build-arg KUBE_BRIDGE_PORT=8171 \
	--build-arg VERSION="$(VERSION)" \
	--build-arg BUILD_DATE="$(BUILD_DATE)" \
	--build-arg REPO_URL="$(REPO_URL)" \
	--build-arg LAST_COMMIT="$(LAST_COMMIT)" \
	--build-arg PROJECT_NAME="$(PROJECT_NAME)" \
	--build-arg VENDOR="$(VENDOR)" .


.PHONY: image.push
image.push: ### Push the image to the Docker Registry
	@$(DOCKER) push "$(DOCKER_REGISTRY)/$(PROJECT_NAME):$(VERSION)"

.PHONY: deps
deps:	### Optimize dependencies
	@go mod tidy

.PHONY: fmt
fmt: ### Format
	@gofmt -s -w .

.PHONY: vet
vet: ### Vet
	@go vet ./...

### Lint
.PHONY: lint
lint: fmt vet

.PHONY: clean
clean: ### Clean build files
	@rm -rf ./bin
	@go clean

.PHONY: build
build: ### Build binary
	@CGO_ENABLED=0 go build -tags netgo -a -v -ldflags "${LD_FLAGS}" -o ./bin/service ./main.go
	@chmod +x ./bin/*

.PHONY: chart
chart: ### Build the Helm chart for this service
	@rm deploy/*.tgz
	@$(SED) -E -i "s/version:\s+[0-9]\.[0-9]\.[0-9]/version: $(VERSION)/g" chart/Chart.yaml
	@$(SED) -E -i "s/appVersion:\s+[0-9]\.[0-9]\.[0-9]/appVersion: $(VERSION)/g" chart/Chart.yaml
	@$(HELM) package chart --destination ./deploy

.PHONY: deploy
deploy: chart ### Deploy the service using the generated Helm chart
	@$(HELM) install kubebridge deploy/kube-bridge-0.1.0.tgz  --namespace krateo-system --create-namespace
