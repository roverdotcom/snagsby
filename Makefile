dist:
	./dist.sh
.PHONY: dist

run:
	go install
	snagsby
.PHONY: run

test:
	go test ./...
.DEFAULT_GOAL := test
.PHONY: test
