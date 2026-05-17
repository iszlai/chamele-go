.PHONY: bootstrap build test lint bdd parity clean

bootstrap:
	bash scripts/fetch-upstream.sh

build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run

bdd:
	go test ./features/...

parity:
	go test -tags parity ./test/parity/...

clean:
	rm -rf bin/ coverage.out
