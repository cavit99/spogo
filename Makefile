COVERAGE_THRESHOLD ?= 90

.PHONY: build check clean coverage format lint race spogo test

spogo: build

build:
	go build -o spogo ./cmd/spogo

test:
	go test ./...

race:
	go test -race ./...

coverage:
	./scripts/check-coverage.sh $(COVERAGE_THRESHOLD)

lint:
	./scripts/lint.sh

format:
	./scripts/format.sh

check: lint test race coverage

clean:
	rm -f spogo coverage.out
