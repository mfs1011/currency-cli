BINARY := cx
VERSION := 0.1.0
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"

.PHONY: build build-all linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 clean install

build:
	go build $(LDFLAGS) -o $(BINARY) .

build-all: linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64

linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 .

linux-arm64:
	GOOS=linux GOARCH=arm64 go build -buildmode=pie $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 .

darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 .

darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 .

windows-amd64:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe .

clean:
	rm -f $(BINARY) dist/*

install:
	go build $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY) .

# Termux install: copy linux-arm64 binary
install-termux: linux-arm64
	@mkdir -p $$HOME/bin
	cp dist/$(BINARY)-linux-arm64 $$HOME/bin/$(BINARY)
	chmod +x $$HOME/bin/$(BINARY)
	@echo "Installed to ~/bin/cx — ensure ~/bin is in PATH"
	@echo "Add to ~/.bashrc: export PATH=\"\$$HOME/bin:\$$PATH\""
