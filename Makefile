PACKAGE_NAME          := github.com/shareed2k/honey
GOLANG_CROSS_VERSION  ?= v1.15.7

export GO111MODULE=on
export CGO_ENABLED=0

.PHONY: all
all: test build

#---------------
#-- test, lint
#---------------

.PHONY: test
test: tools.gotestsum lint
	@echo "==> Running tests..."
	@gotestsum --format short-verbose --junitfile junit.xml -- -coverprofile=codecov.out -covermode=atomic ./...

.PHONY: lint
lint: tools.golangci-lint
	@echo "==> Running lints..."
	@golangci-lint run

.PHONY: release-dry-run
release-dry-run:
	@docker run \
		--privileged \
		-e CGO_ENABLED=0 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		troian/golang-cross:${GOLANG_CROSS_VERSION} \
		--rm-dist --skip-validate --skip-publish

.PHONY: release
release:
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
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		troian/golang-cross:${GOLANG_CROSS_VERSION} \
		release --rm-dist

.PHONY: build
build:
	@go build -v -o bin/honey

#---------------
#-- tools
#---------------
.PHONY: tools
tools: tools.golangci-lint tools.gotestsum tools.easyjson

.PHONY: tools.golangci-lint
tools.golangci-lint:
	@command -v golangci-lint >/dev/null || { \
		echo "==> Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint; \
	}

.PHONY: tools.gotestsum
tools.gotestsum:
	@command -v gotestsum >/dev/null || { \
		echo "==> Installing gotestsum..."; \
		go install gotest.tools/gotestsum; \
	}

.PHONY: tools.easyjson
tools.easyjson:
	@command -v easyjson >/dev/null || { \
		echo "==> Installing easyjson..."; \
		go install github.com/mailru/easyjson; \
	}