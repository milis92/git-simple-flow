BINARY := git-sf

.PHONY: build test test-integration test-all lint fmt coverage changelog clean

build:
	go build -o $(BINARY) .

test:
	go test ./internal/... -v

test-integration:
	go test -tags integration ./test/... -v -count=1

test-all:
	go test -tags integration ./... -v -count=1

lint:
	go tool golangci-lint run

fmt:
	gofmt -w .

fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Files not formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

coverage:
	go test ./internal/... -v -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

changelog:
	git cliff -o CHANGELOG.md

clean:
	rm -f $(BINARY) coverage.out
