.PHONY: build run install test vet clean

build:
	go build -o bin/ai-top ./cmd/ai-top/

run: build
	./bin/ai-top

install: build
	go install ./cmd/ai-top/

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f bin/ai-top
