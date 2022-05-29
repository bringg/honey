PACKAGE_NAME          := github.com/bringg/honey
GOLANG_CROSS_VERSION  ?= v1.17.5
VERSION               ?=beta-$(shell git rev-parse --short HEAD)
GIT_COMMIT            ?=$(shell git rev-parse --short HEAD)
BUILD_TIME            ?=$(shell date -u '+%F_%T')
BUILD_BY              ?=shareed2k

export GO111MODULE=on
export CGO_ENABLED=0

.PHONY: all
all: test build

#---------------
#-- test, lint
#---------------

.PHONY: test
test: tools
	@echo "==> Running tests..."
	@gotestsum --format short-verbose --junitfile junit.xml -- -coverprofile=codecov.out -covermode=atomic ./...

.PHONY: lint
lint: tools
	@echo "==> Running lints..."
	@golangci-lint run

.PHONY: release-dry-run
release-dry-run: ui
	@docker run \
		--privileged \
		-e CGO_ENABLED=0 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v ${PWD}:/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		ghcr.io/troian/golang-cross:${GOLANG_CROSS_VERSION} \
		--rm-dist --skip-validate --skip-publish

.PHONY: release
release: ui
	@if [ ! -f ".release-env" ]; then \
		echo "\033[91m.release-env is required for release\033[0m";\
		exit 1;\
	fi
	docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=0 \
		--env-file .release-env \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v ~/.docker:/root/.docker \
		-v ${PWD}:/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		ghcr.io/troian/golang-cross:${GOLANG_CROSS_VERSION} \
		release --rm-dist

.PHONY: build
build:
	@echo "==> Building..."
	@go build \
		-ldflags="-s -w \
			-X github.com/bringg/honey/cmd.version=${VERSION} \
			-X github.com/bringg/honey/cmd.commit=${GIT_COMMIT} \
			-X github.com/bringg/honey/cmd.date=${BUILD_TIME} \
			-X github.com/bringg/honey/cmd.builtBy=${BUILD_BY}" \
		-o ./bin/honey

.PHONY: ui
ui:
	@echo "==> Building UI..."
	@docker run \
		--rm \
		-e PUBLIC_URL=. \
		-w /opt/src \
		-v ${PWD}/ui:/opt/src \
		node:16-alpine yarn ui

#---------------
#-- tools
#---------------
.PHONY: tools
tools:
	@echo "==> installing tools from tools.go..."
	@awk -F'"' '/_/ {print $$2}' tools.go | xargs -tI % go install %
