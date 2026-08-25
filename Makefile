BINARY := bin/dojo
PKG    := ./...

.PHONY: all build install test vet validate clean fmt

all: build

build:
	@mkdir -p bin
	go build -o $(BINARY) ./cmd/dojo

install:
	go install ./cmd/dojo

test:
	go test $(PKG)

vet:
	go vet $(PKG)

fmt:
	gofmt -l -w .

# Content linting uses the freshly built binary against the in-repo content.
validate: build
	DOJO_CONTENT=$(CURDIR) $(BINARY) content validate

check: vet test validate

clean:
	rm -rf bin
