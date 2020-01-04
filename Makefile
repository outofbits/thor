VERSION=`grep -oP 'const ApplicationVersion string = "\K[\d.-a-zA-Z]+' main.go`

build-cross-platform: clean-build-dir build-linux-amd64 build-linux-arm build-linux-arm64

clean-build-dir:
	rm -rf build/

build-linux-amd64:
	env GOOS=linux GOARCH=amd64 go build -o build/thor .
	tar cfvz "build/thor-${VERSION}-linux-amd64.tar.gz" -C build thor 
	rm build/thor

build-linux-arm:
	env GOOS=linux GOARCH=arm go build -o build/thor .
	tar cfvz "build/thor-${VERSION}-linux-arm.tar.gz" -C build  thor 
	rm build/thor

build-linux-arm64:
	env GOOS=linux GOARCH=arm64 go build -o build/thor .
	tar cfvz "build/thor-${VERSION}-linux-arm64.tar.gz" -C build thor 
	rm build/thor

build-docker-image:
	docker build . --build-arg THOR_VERSION=$(VERSION) --tag adalove/thor:$(VERSION)
