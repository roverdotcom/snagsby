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

.PHONY: run
run:
	go install && snagsby

.PHONY: fpm
fpm:
	docker build -t snagsby-fpm -f scripts/DockerfileFpm ./scripts
	docker run --rm -it \
		-v $(PWD):/app \
		-w /app \
		snagsby-fpm \
		./scripts/fpm.sh


.PHONY: install
install:
	go install

.PHONY: test
test:
	@go test -v $(shell go list ./... | grep -v vendor)

.DEFAULT_GOAL := test
