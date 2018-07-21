clean:
	rm -rf bin/
	rm -rf dist/

dist:
	./dist.sh

docker-dist:
	docker pull golang:1.10
	docker run --rm \
		-v $(PWD):/go/src/github.com/roverdotcom/snagsby \
		-w /go/src/github.com/roverdotcom/snagsby \
		golang:1.10 \
		make dist

run:
	go install && snagsby

install:
	go install

test:
	@go test -v $(shell go list ./... | grep -v vendor)

.DEFAULT_GOAL := test
.PHONY: test run docker-dist dist install clean
