all:
	go build .

LDFLAGS := -X main.Version=$(VERSION)
release:
	@echo "Checking that VERSION was defined in the calling environment"
	@test -n "$(VERSION)"
	@echo "OK.  VERSION=$(VERSION)"
	GOOS=linux  GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o s3-linux-amd64
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o s3-darwin-amd64
	                         go build -ldflags="$(LDFLAGS)" -o s3
	./s3 --version
