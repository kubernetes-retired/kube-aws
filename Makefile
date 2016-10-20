.PHONY: build
build:
	./build

.PHONY: format
format:
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -d {} + | tee /dev/stderr)" || \
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -w {} + | tee /dev/stderr)"

.PHONY: test
test: build
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -d {} + | tee /dev/stderr)"
	go test -v $$(go list ./... | grep -v '/vendor/')
	go vet $$(go list ./... | grep -v '/vendor/')
