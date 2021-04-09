VERSION ?= $(shell cat VERSION)
GOLANG_VERSION ?= 1.16.3
GOLANG_DOCKER_IMAGE ?= golang:$(GOLANG_VERSION)
GO_LDFLAGS := -X github.com/roverdotcom/snagsby/pkg.Version=$(VERSION)
export

.PHONY: clean
clean:
	rm -rf bin/
	rm -rf dist/
	rm -f ./snagsby


.PHONY: dist
dist:
	./scripts/dist.sh


.PHONY: docker-dist
docker-dist:
	docker pull $(GOLANG_DOCKER_IMAGE)
	docker run --rm \
		-v $(PWD):/go/src/github.com/roverdotcom/snagsby \
		-w /go/src/github.com/roverdotcom/snagsby \
		-e VERSION=$(VERSION) \
		$(GOLANG_DOCKER_IMAGE) \
		make dist


.PHONY: docker-test
docker-test:
	docker pull $(GOLANG_DOCKER_IMAGE)
	docker run --rm \
		-v $(PWD):/go/src/github.com/roverdotcom/snagsby \
		-w /go/src/github.com/roverdotcom/snagsby \
		-e VERSION=$(VERSION) \
		$(GOLANG_DOCKER_IMAGE) \
		make test


.PHONY: install
install:
	CGO_ENABLED=0 go install -ldflags "$(GO_LDFLAGS)"


.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags "$(GO_LDFLAGS)" -o snagsby


.PHONY: run
run: install
	@$(GOPATH)/bin/snagsby


.PHONY: fpm
fpm:
	docker build -t snagsby-fpm -f scripts/DockerfileFpm ./scripts
	docker run --rm -it \
		-v $(PWD):/app \
		-w /app \
		-e VERSION=$(VERSION) \
		snagsby-fpm \
		./scripts/fpm.sh


.PHONY: test
test:
	@go test -v $(shell go list ./... | grep -v vendor)


.PHONY: e2e
e2e: dist
	./e2e/e2e.sh


.PHONY: e2e-quick
e2e-quick:
	./e2e/e2e.sh


.PHONY: docker-build-images
docker-build-images:
	docker build --pull -t snagsby:v$(VERSION) .
	docker build --pull -t snagsby:v$(VERSION)-dev --target dev .


.DEFAULT_GOAL := test
