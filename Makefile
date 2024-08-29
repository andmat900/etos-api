# Install tools locally instead of in $HOME/go/bin.
export GOBIN := $(CURDIR)/bin
export GOBUILD := go build
export PATH := $(GOBIN):$(PATH)

export RELEASE_VERSION ?= $(shell git describe --always)
export DOCKER_REGISTRY ?= registry.nordix.org
export DOCKER_NAMESPACE ?= eiffel
export DEPLOY ?= etos-sse

PROGRAMS = sse logarea iut
COMPILEDAEMON = $(GOBIN)/CompileDaemon
GIT = git
GOLANGCI_LINT = $(GOBIN)/golangci-lint

GOLANGCI_LINT_VERSION = v1.52.2

# Generates a rule for building a single binary from cmd/$1.
#
# $1: Name of the program
define build-binary
.PHONY: $(strip $(1))
$(strip $(1)):
	$$(GOBUILD) -ldflags="-w -s -buildid=" -trimpath -o bin/$(strip $(1)) ./cmd/$(strip $(1))
endef

# Generates a rule named $(program_name)-docker to to build the named program
# and copy the locally compiled binary into a Docker image. The image will be
# tagged with the name of the program.
#
# $1: Name of the program
define build-docker-image
.PHONY: $(strip $(1))-docker
# Including the parameter name!
EXTRA_DOCKER_ARGS=
export EXTRA_DOCKER_ARGS
$(strip $(1))-docker: $(strip $(1))
	docker build . \
    $(EXTRA_DOCKER_ARGS) \
		-f deploy/etos-$(strip $(1))/Dockerfile \
		-t $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/etos-$(strip $(1)):$(RELEASE_VERSION)
endef

# Generates a rule named $(program_name)-docker-push to push
# a local Docker image to a remote registry.
#
# $1: Name of the program
define push-docker-image
.PHONY: $(strip $(1))-docker-push
$(strip $(1))-docker-push:
	echo docker push $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(strip $(1))_provider:$(RELEASE_VERSION)
endef

.PHONY: all
all: test $(PROGRAMS)

.PHONY: clean
clean:
	$(RM) $(GOBIN)/*
	docker compose --project-directory . -f deploy/$(DEPLOY)/docker-compose.yml rm || true

.PHONY: check
check: staticcheck test

.PHONY: staticcheck
staticcheck: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run

.PHONY: test
test:
	go test -cover -timeout 30s -race $(shell go list ./... | grep -v "etos-api/test")

# Start a development docker with a database that restarts on file changes.
.PHONY: start
start: $(COMPILEDAEMON)
	docker compose --project-directory . -f deploy/$(DEPLOY)/docker-compose.yml up

.PHONY: stop
stop:
	docker compose --project-directory . -f deploy/$(DEPLOY)/docker-compose.yml down

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: check-dirty
check-dirty:
	$(GIT) diff --exit-code HEAD


# Setup the dynamic commands
#
$(foreach prog,$(PROGRAMS), \
    $(eval $(call build-binary,$(prog))) \
    $(eval $(call build-docker-image,$(prog))) \
    $(eval $(call push-docker-image,$(prog))) \
)

# Build dependencies

$(COMPILEDAEMON):
	mkdir -p $(dir $@)
	go install github.com/githubnemo/CompileDaemon@v1.3.0

$(GOLANGCI_LINT):
	mkdir -p $(dir $@)
	curl -sfL \
			https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
		| sh -s -- -b $(GOBIN) $(GOLANGCI_LINT_VERSION)
