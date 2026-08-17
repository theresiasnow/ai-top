.PHONY: build run install test vet clean

build:
	go build -o bin/ai-top ./cmd/ai-top/

run: build
	./bin/ai-top

# Installs by symlinking ~/.local/bin/ai-top at the freshly built binary, so the
# installed command always tracks this working copy.
#
# Deliberately not `go install`: that writes a second, independent copy into
# GOPATH/bin, which sits earlier in PATH and would silently shadow this symlink
# with whatever was built the last time someone ran it.
install: build
	@mkdir -p $(HOME)/.local/bin
	@ln -sfn $(CURDIR)/bin/ai-top $(HOME)/.local/bin/ai-top
	@echo "linked $(HOME)/.local/bin/ai-top -> $(CURDIR)/bin/ai-top"

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f bin/ai-top
