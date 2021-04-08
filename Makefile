.PHONY: all docs docs-serve

SRCS = $(shell git ls-files '*.go' | grep -v '^vendor/')

TAG_NAME := $(shell git tag -l --contains HEAD)
SHA := $(shell git rev-parse HEAD)
VERSION_GIT := $(if $(TAG_NAME),$(TAG_NAME),$(SHA))
VERSION := $(if $(VERSION),$(VERSION),$(VERSION_GIT))

BIND_DIR := "dist"

GIT_BRANCH := $(subst heads/,,$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null))
TRAEFIK_DEV_IMAGE := traefik-dev$(if $(GIT_BRANCH),:$(subst /,-,$(GIT_BRANCH)))

REPONAME := $(shell echo $(REPO) | tr '[:upper:]' '[:lower:]')
TRAEFIK_IMAGE := $(if $(REPONAME),$(REPONAME),"traefik/traefik")

INTEGRATION_OPTS := $(if $(MAKE_DOCKER_HOST),-e "DOCKER_HOST=$(MAKE_DOCKER_HOST)", -e "TEST_CONTAINER=1" -v "/var/run/docker.sock:/var/run/docker.sock")
DOCKER_BUILD_ARGS := $(if $(DOCKER_VERSION), "--build-arg=DOCKER_VERSION=$(DOCKER_VERSION)",)

TRAEFIK_ENVS := \
	-e OS_ARCH_ARG \
	-e OS_PLATFORM_ARG \
	-e TESTFLAGS \
	-e VERBOSE \
	-e VERSION \
	-e CODENAME \
	-e TESTDIRS \
	-e CI \
	-e CONTAINER=DOCKER		# Indicator for integration tests that we are running inside a container.

TRAEFIK_MOUNT := -v "$(CURDIR)/$(BIND_DIR):/go/src/github.com/traefik/traefik/$(BIND_DIR)"
DOCKER_RUN_OPTS := $(TRAEFIK_ENVS) $(TRAEFIK_MOUNT) "$(TRAEFIK_DEV_IMAGE)"
DOCKER_NON_INTERACTIVE ?= false
DOCKER_RUN_TRAEFIK := docker run --add-host=host.docker.internal:127.0.0.1 $(INTEGRATION_OPTS) $(if $(DOCKER_NON_INTERACTIVE), , -it) $(DOCKER_RUN_OPTS)
DOCKER_RUN_TRAEFIK_NOTTY := docker run $(INTEGRATION_OPTS) $(if $(DOCKER_NON_INTERACTIVE), , -i) $(DOCKER_RUN_OPTS)

PRE_TARGET ?= build-dev-image

PLATFORM_URL := $(if $(PLATFORM_URL),$(PLATFORM_URL),"https://pilot.traefik.io")

default: binary

## Build Dev Docker image
build-dev-image: dist
	docker build $(DOCKER_BUILD_ARGS) -t "$(TRAEFIK_DEV_IMAGE)" -f build.Dockerfile .

## Build Dev Docker image without cache
build-dev-image-no-cache: dist
	docker build --no-cache -t "$(TRAEFIK_DEV_IMAGE)" -f build.Dockerfile .

## Create the "dist" directory
dist:
	mkdir dist

## Build WebUI Docker image
build-webui-image:
	docker build -t traefik-webui --build-arg ARG_PLATFORM_URL=$(PLATFORM_URL) -f webui/Dockerfile webui

## Generate WebUI
generate-webui: build-webui-image
	if [ ! -d "static" ]; then \
		mkdir -p static; \
		docker run --rm -v "$$PWD/static":'/src/static' traefik-webui npm run build:nc; \
		docker run --rm -v "$$PWD/static":'/src/static' traefik-webui chown -R $(shell id -u):$(shell id -g) ../static; \
		echo 'For more information show `webui/readme.md`' > $$PWD/static/DONT-EDIT-FILES-IN-THIS-DIRECTORY.md; \
	fi

## Build the linux binary
binary: generate-webui $(PRE_TARGET)
	$(if $(PRE_TARGET),$(DOCKER_RUN_TRAEFIK)) ./script/make.sh generate binary

## Build the binary for the standard platforms (linux, darwin, windows)
crossbinary-default: generate-webui build-dev-image
	$(DOCKER_RUN_TRAEFIK_NOTTY) ./script/make.sh generate crossbinary-default

## Build the binary for the standard platforms (linux, darwin, windows) in parallel
crossbinary-default-parallel:
	$(MAKE) generate-webui
	$(MAKE) build-dev-image crossbinary-default

## Run the unit and integration tests
test: build-dev-image
	$(DOCKER_RUN_TRAEFIK) ./script/make.sh generate test-unit binary test-integration

## Run the unit tests
test-unit: $(PRE_TARGET)
	$(if $(PRE_TARGET),$(DOCKER_RUN_TRAEFIK)) ./script/make.sh generate test-unit

## Pull all images for integration tests
pull-images:
	grep --no-filename -E '^\s+image:' ./integration/resources/compose/*.yml | awk '{print $$2}' | sort | uniq | xargs -P 6 -n 1 docker pull

## Run the integration tests
test-integration: $(PRE_TARGET)
	$(if $(PRE_TARGET),$(DOCKER_RUN_TRAEFIK),TEST_CONTAINER=1) ./script/make.sh generate binary test-integration
	TEST_HOST=1 ./script/make.sh test-integration

## Validate code and docs
validate-files: $(PRE_TARGET)
	$(if $(PRE_TARGET),$(DOCKER_RUN_TRAEFIK)) ./script/make.sh generate validate-lint validate-misspell
	bash $(CURDIR)/script/validate-shell-script.sh

## Validate code, docs, and vendor
validate: $(PRE_TARGET)
	$(if $(PRE_TARGET),$(DOCKER_RUN_TRAEFIK)) ./script/make.sh generate validate-lint validate-misspell validate-vendor
	bash $(CURDIR)/script/validate-shell-script.sh

## Clean up static directory and build a Docker Traefik image
build-image: binary
	rm -rf static
	docker build -t $(TRAEFIK_IMAGE) .

## Build a Docker Traefik image
build-image-dirty: binary
	docker build -t $(TRAEFIK_IMAGE) .

## Start a shell inside the build env
shell: build-dev-image
	$(DOCKER_RUN_TRAEFIK) /bin/bash

## Build documentation site
docs:
	make -C ./docs docs

## Serve the documentation site localy
docs-serve:
	make -C ./docs docs-serve

## Pull image for doc building
docs-pull-images:
	make -C ./docs docs-pull-images

## Generate CRD clientset
generate-crd:
	./script/update-generated-crd-code.sh

## Create packages for the release
release-packages: generate-webui build-dev-image
	rm -rf dist
	$(DOCKER_RUN_TRAEFIK_NOTTY) goreleaser release --skip-publish --timeout="60m"
	$(DOCKER_RUN_TRAEFIK_NOTTY) tar cfz dist/traefik-${VERSION}.src.tar.gz \
		--exclude-vcs \
		--exclude .idea \
		--exclude .travis \
		--exclude .semaphoreci \
		--exclude .github \
		--exclude dist .
	$(DOCKER_RUN_TRAEFIK_NOTTY) chown -R $(shell id -u):$(shell id -g) dist/

## Format the Code
fmt:
	gofmt -s -l -w $(SRCS)

run-dev:
	go generate
	GO111MODULE=on go build ./cmd/traefik
	./traefik
