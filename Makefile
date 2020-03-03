.PHONY: deps-update

deps-update:
	go mod tidy && \
	go mod vendor

gofmt:
	@echo "Running gofmt"
	gofmt -s -l `find . -path ./vendor -prune -o -type f -name '*.go' -print`

build-bin:
	go build --mod=vendor -o bin/network-metrics-daemon
	chmod +x bin/network-metrics-daemon

unittests:
	go test ./...
