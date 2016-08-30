dist:
	./dist.sh
.PHONY: dist

docker-dist:
	docker pull golang:1.7
	docker run --rm \
		-v $(PWD):/go/src/github.com/roverdotcom/snagsby \
		-w /go/src/github.com/roverdotcom/snagsby \
		golang:1.7 \
		go get && make dist
.PHONY: docker-dist

run:
	go install && snagsby
.PHONY: run

test:
	go test -v ./...
.DEFAULT_GOAL := test
.PHONY: test
