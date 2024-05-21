TEST?=./...
PRODUCT=project
GO_ARCH=$(shell go env GOARCH)
TARGET_ARCH=$(shell go env GOOS)_${GO_ARCH}
GORELEASER_ARCH=${TARGET_ARCH}

ifeq ($(GO_ARCH), amd64)
GORELEASER_ARCH=${TARGET_ARCH}_$(shell go env GOAMD64)
endif
PKG_NAME=pkg/${PRODUCT}
# if this path ever changes, you need to also update the 'ldflags' value in .goreleaser.yml
PKG_VERSION_PATH=github.com/jfrog/terraform-provider-${PRODUCT}/${PKG_NAME}
VERSION := $(shell git tag --sort=-creatordate | head -1 | sed  -n 's/v\([0-9]*\).\([0-9]*\).\([0-9]*\)/\1.\2.\3/p')
NEXT_VERSION := $(shell echo ${VERSION}| awk -F '.' '{print $$1 "." $$2 "." $$3 +1 }' )
BUILD_PATH=terraform.d/plugins/registry.terraform.io/jfrog/${PRODUCT}/${NEXT_VERSION}/${TARGET_ARCH}
SONAR_SCANNER_VERSION?=4.7.0.2747
SONAR_SCANNER_HOME?=${HOME}/.sonar/sonar-scanner-${SONAR_SCANNER_VERSION}-macosx

default: build

install: clean build
	rm -fR .terraform.d && \
	mkdir -p ${BUILD_PATH} && \
		mv -v dist/terraform-provider-${PRODUCT}_${GORELEASER_ARCH}/terraform-provider-${PRODUCT}_v${NEXT_VERSION}* ${BUILD_PATH} && \
		rm -f .terraform.lock.hcl && \
		sed -i.bak '0,/version = ".*"/s//version = "${NEXT_VERSION}"/' sample.tf && rm sample.tf.bak && \
		terraform init

install_tfc: clean build_tfc
	mkdir -p tfc-testing/${BUILD_PATH} && \
	mkdir -p tfc-testing/terraform.d/plugins/registry.terraform.io/jfrog/${PRODUCT}/${NEXT_VERSION}/linux_amd64 && \
		mv -v dist/terraform-provider-${PRODUCT}_${GORELEASER_ARCH}/terraform-provider-${PRODUCT}_v${NEXT_VERSION}* tfc-testing/${BUILD_PATH} && \
		mv -v dist/terraform-provider-${PRODUCT}_linux_amd64_v1/terraform-provider-${PRODUCT}_v${NEXT_VERSION}* tfc-testing/terraform.d/plugins/registry.terraform.io/jfrog/${PRODUCT}/${NEXT_VERSION}/linux_amd64 && \
		sed -i.bak '0,/version = ".*"/s//version = "${NEXT_VERSION}"/' tfc-testing/sample.tf && rm tfc-testing/sample.tf.bak && \
		cd tfc-testing && \
		terraform providers lock -platform=linux_amd64 -platform=darwin_amd64 -fs-mirror=terraform.d/plugins && \
		terraform init

clean:
	rm -fR dist terraform.d/ .terraform terraform.tfstate* .terraform.lock.hcl

clean_tfc:
	rm -fR dist tfc-testing/terraform.d/ tfc-testing/.terraform tfc-testing/terraform.tfstate* tfc-testing/.terraform.lock.hcl

release:
	@git tag ${NEXT_VERSION} && git push --mirror
	@echo "Pushed ${NEXT_VERSION}"
	GOPROXY=https://proxy.golang.org GO111MODULE=on go get github.com/jfrog/terraform-provider-${PRODUCT}@v${NEXT_VERSION}
	@echo "Updated pkg cache"

update_pkg_cache:
	GOPROXY=https://proxy.golang.org GO111MODULE=on go get github.com/jfrog/terraform-provider-${PRODUCT}@v${VERSION}

build: fmt
	GORELEASER_CURRENT_TAG=${NEXT_VERSION} goreleaser build --single-target --clean --snapshot

build_tfc: fmt
	GORELEASER_CURRENT_TAG=${NEXT_VERSION} goreleaser build --clean --snapshot --config tfc-testing/.goreleaser.yml

test:
	@echo "==> Starting unit tests"
	go test $(TEST) -timeout=30s -parallel=4

test_tfc: install_tfc
	cd tfc-testing && \
	terraform plan

attach:
	dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient attach $$(pgrep terraform-provider-${PRODUCT})

acceptance: fmt
	export TF_ACC=true && \
		go test -cover -coverprofile=coverage.txt -ldflags="-X '${PKG_VERSION_PATH}/provider.Version=${NEXT_VERSION}-test'" -v -p 1 -parallel 20 -timeout 20m ./pkg/...

# To generate coverage.txt run `make acceptance` first
coverage:
	go tool cover -html=coverage.txt

# SONAR_TOKEN (project token) must be set to run `make scan`. Check file sonar-project.properties for the configuration.
scan:
	${SONAR_SCANNER_HOME}/bin/sonar-scanner -Dsonar.projectVersion=${VERSION} -Dsonar.go.coverage.reportPaths=coverage.txt

fmt:
	@echo "==> Fixing source code with gofmt..."
	@go fmt ./...

doc:
	rm -rfv docs/*
	go generate

.PHONY: build fmt