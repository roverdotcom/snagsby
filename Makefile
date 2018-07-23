VERSION := $(shell cat VERSION)


.PHONY: clean
clean:
	rm -rf bin/
	rm -rf dist/


.PHONY: dist
dist:
	./scripts/dist.sh


.PHONY: docker-dist
docker-dist:
	docker pull golang:1.10
	docker run --rm \
		-v $(PWD):/go/src/github.com/roverdotcom/snagsby \
		-w /go/src/github.com/roverdotcom/snagsby \
		golang:1.10 \
		make dist


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


.DEFAULT_GOAL := test
