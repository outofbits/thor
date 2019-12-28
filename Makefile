VERSION=`grep -oP 'const ApplicationVersion string = "\K[\d.-a-zA-Z]+' main.go`

build-docker-image:
	docker build . --build-arg THOR_VERSION=$(VERSION) --tag adalove/thor:$(VERSION)