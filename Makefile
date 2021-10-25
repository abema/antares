.PHONY: build
build:
	go build

.PHONY: vet
vet:
	go vet

.PHONY: lint
lint:
	golint ./...

.PHONY: test
test:
	go test ./...

.PHONY: coverage
coverage:
	go test -coverprofile=cover.out -covermode=count ./...
	go tool cover -html=cover.out -o cover.html
	if [ $(shell uname) == Darwin ]; then open cover.html; fi
