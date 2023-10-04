# Install tools locally instead of in $HOME/go/bin.
export GOBIN := $(CURDIR)/bin
export PATH := $(GOBIN):$(PATH)

export RELEASE_VERSION ?= $(shell git describe --always)
export DOCKER_REGISTRY ?= registry.nordix.org
export DOCKER_NAMESPACE ?= eiffel
export DEPLOY ?= etos-sse

COMPILEDAEMON = $(GOBIN)/CompileDaemon
GIT = git
SSE = $(GOBIN)/etos-sse
GOLANGCI_LINT = $(GOBIN)/golangci-lint

GOLANGCI_LINT_VERSION = v1.52.2

.PHONY: all
all: test build start

.PHONY: build
build:
	go build -ldflags="-w -s -buildid=" -trimpath -o $(SSE) ./cmd/sse

.PHONY: clean
clean:
	$(RM) $(SSE)
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

# Build a docker using the production Dockerfiler
.PHONY: docker
# Including the parameter name!
EXTRA_DOCKER_ARGS=
export EXTRA_DOCKER_ARGS
docker:
	docker build $(EXTRA_DOCKER_ARGS) -t $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(DEPLOY):$(RELEASE_VERSION) -f ./deploy/$(DEPLOY)/Dockerfile .

.PHONY: push
push:
	docker push $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(DEPLOY):$(RELEASE_VERSION)

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: check-dirty
check-dirty:
	$(GIT) diff --exit-code HEAD

# Build dependencies

$(COMPILEDAEMON):
	mkdir -p $(dir $@)
	go install github.com/githubnemo/CompileDaemon@v1.3.0

$(GOLANGCI_LINT):
	mkdir -p $(dir $@)
	curl -sfL \
			https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
		| sh -s -- -b $(GOBIN) $(GOLANGCI_LINT_VERSION)
