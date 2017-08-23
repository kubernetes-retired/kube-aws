CALICO_BUILD?=calico/go-build
PACKAGE_NAME?=kubernetes-incubator/kube-aws
LOCAL_USER_ID?=$(shell id -u $$USER)

.PHONY: build
build:
	./build

vendor: glide.yaml
	rm -f glide.lock
	docker run --rm \
	    -v $(CURDIR):/go/src/github.com/$(PACKAGE_NAME):rw \
	    -e LOCAL_USER_ID=$(LOCAL_USER_ID) \
            -w /go/src/github.com/$(PACKAGE_NAME) \
	    $(CALICO_BUILD) glide install -strip-vendor
	

.PHONY: format
format:
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -d {} + | tee /dev/stderr)" || \
	test -z "$$(find . -path ./vendor -prune -type f -o -name '*.go' -exec gofmt -w {} + | tee /dev/stderr)"

.PHONY: test
test: build
	./make/test

.PHONY: test-with-cover
test-with-cover: build
	./make/test with-cover

.PHONY: docs-dependencies
docs-dependencies:
	if ! which gitbook; then npm install -g gitbook-cli; fi
	if ! which gh-pages; then npm install -g gh-pages; fi
	gitbook install

.PHONY: generate-docs
generate-docs: docs-dependencies
	gitbook build

.PHONY: serve-docs
serve-docs: docs-dependencies
	gitbook serve

.PHONY: publish-docs
publish-docs: generate-docs
	NODE_DEBUG=gh-pages gh-pages -d _book
