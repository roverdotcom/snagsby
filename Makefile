VERSION ?= $(shell cat VERSION)
GOLANG_DOCKER_IMAGE := golang:1.11


.PHONY: clean
clean:
	rm -rf bin/
	rm -rf dist/


.PHONY: dist
dist:
	./scripts/dist.sh


.PHONY: docker-dist
docker-dist:
	docker pull $(GOLANG_DOCKER_IMAGE)
	docker run --rm \
		-v $(PWD):/go/src/github.com/roverdotcom/snagsby \
		-w /go/src/github.com/roverdotcom/snagsby \
		$(GOLANG_DOCKER_IMAGE) \
		make dist


.PHONY: docker-test
docker-test:
	docker pull $(GOLANG_DOCKER_IMAGE)
	docker run --rm \
		-v $(PWD):/go/src/github.com/roverdotcom/snagsby \
		-w /go/src/github.com/roverdotcom/snagsby \
		$(GOLANG_DOCKER_IMAGE) \
		make test


.PHONY: install
install:
	go install -ldflags "-X main.Version=$(VERSION)"


.PHONY: run
run: install
	@$(GOPATH)/bin/snagsby


.PHONY: fpm
fpm:
	docker build -t snagsby-fpm -f scripts/DockerfileFpm ./scripts
	docker run --rm -it \
		-v $(PWD):/app \
		-w /app \
		snagsby-fpm \
		./scripts/fpm.sh


.PHONY: test
test:
	@go test -v $(shell go list ./... | grep -v vendor)


.PHONY: e2e
e2e: dist
	./e2e/e2e.sh


.PHONY: e2e-quick
e2e-quick: install
	SNAGSBY_BIN=$(GOPATH)/bin/snagsby ./e2e/e2e.sh


.DEFAULT_GOAL := test
