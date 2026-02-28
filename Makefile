BIN := bin/lazyenv
SRC := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: build run test vet clean

build: $(BIN)

$(BIN): $(SRC) go.mod go.sum
	@mkdir -p bin
	go build -o $(BIN) .

run: $(BIN)
	@$(BIN) $(ARGS)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf bin
