TEST?=./...
TARGET_ARCH?=darwin_amd64
PKG_NAME=pkg/project
PKG_VERSION_PATH=github.com/jfrog/terraform-provider-project/${PKG_NAME}
VERSION := $(shell git tag --sort=-creatordate | head -1 | sed -n 's/v\([0-9]*\).\([0-9]*\).\([0-9]*\)/\1.\2.\3/p')
NEXT_VERSION := $(shell echo ${VERSION}| awk -F '.' '{print $$1 "." $$2 "." $$3 +1 }' )
BINARY_NAME=terraform-provider-project
BUILD_PATH=terraform.d/plugins/registry.terraform.io/jfrog/project/${NEXT_VERSION}/${TARGET_ARCH}

default: build

install:
	mkdir -p ${BUILD_PATH} && \
		(test -f ${BINARY_NAME} || go build -o ./${BINARY_NAME} -ldflags="-X '${PKG_VERSION_PATH}.Version=${NEXT_VERSION}'") && \
		mv ${BINARY_NAME} ${BUILD_PATH} && \
		rm -f .terraform.lock.hcl && \
		sed -i 's/version = ".*"/version = "${NEXT_VERSION}"/' sample.tf && \
		terraform init

clean:
	rm -fR .terraform.d/ .terraform terraform.tfstate* terraform.d/ .terraform.lock.hcl

release:
	@git tag v${NEXT_VERSION} && git push --mirror
	@echo "Pushed v${NEXT_VERSION}"
	GOPROXY=https://proxy.golang.org GO111MODULE=on go get github.com/jfrog/${BINARY_NAME}@v${NEXT_VERSION}
	@echo "Updated pkg cache"

update_pkg_cache:
	GOPROXY=https://proxy.golang.org GO111MODULE=on go get github.com/jfrog/${BINARY_NAME}@v${VERSION}

build: fmtcheck
	go build -ldflags="-X '${PKG_VERSION_PATH}.Version=${NEXT_VERSION}'"

debug_install:
	mkdir -p ${BUILD_PATH} && \
		(test -f ${BINARY_NAME} || go build ./${BINARY_NAME} -gcflags "all=-N -l" -ldflags="-X '${PKG_VERSION_PATH}.Version=${NEXT_VERSION}-develop'") && \
		mv ${BINARY_NAME} ${BUILD_PATH} && \
		terraform init

test:
	@echo "==> Starting unit tests"
	go test $(TEST) -timeout=30s -parallel=4

attach:
	dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient attach $$(pgrep terraform-provider-projects)

acceptance: fmtcheck
	export TF_ACC=true && \
		go test -ldflags="-X '${PKG_VERSION_PATH}.Version=${NEXT_VERSION}-test'" -v -parallel 20 ./pkg/...

fmt:
	@echo "==> Fixing source code with gofmt..."
	@gofmt -s -w ./$(PKG_NAME)
	(command -v goimports &> /dev/null || go get golang.org/x/tools/cmd/goimports) && goimports -w ${PKG_NAME}

fmtcheck:
	@echo "==> Checking that code complies with gofmt requirements..."
	@sh -c "find . -name '*.go' -not -name '*vendor*' -print0 | xargs -0 gofmt -l -s"

doc:
	go generate

.PHONY: build fmt
