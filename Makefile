SCRIPTDIR := $(shell pwd)
GOBUILDDIR := $(SCRIPTDIR)/.gobuild

ifndef DOCKERIMAGE
	DOCKERIMAGE := kube-dhcp:dev
endif

MANIFESTTOOL := $(GOPATH)/bin/manifest-tool

# Development build
.PHONY: build
build:
	docker build -f Dockerfile.build --build-arg=GOARCH=$(shell go env GOARCH) -t $(DOCKERIMAGE) .

# Build all images for all archs.
.PHONY: all
all: pkg/registry/kube_lease.pb.go
	@${MAKE} -B DOCKERIMAGE=$(DOCKERIMAGE) GOARCH=amd64 build-arch
	@${MAKE} -B DOCKERIMAGE=$(DOCKERIMAGE) GOARCH=arm build-arch
	@${MAKE} -B DOCKERIMAGE=$(DOCKERIMAGE) GOARCH=arm64 build-arch

.PHONY: build-arch 
build-arch:
	docker build -f Dockerfile.build --build-arg=GOARCH=$(GOARCH) -t $(DOCKERIMAGE)-$(GOARCH) .
	docker push $(DOCKERIMAGE)-$(GOARCH)

.PHONY: development-env
development-env:
	pulsar go path
	GOPATH=$(GOBUILDDIR) pulsar go flatten -V vendor 

$(MANIFESTTOOL):
	go get github.com/estesp/manifest-tool

.PHONY: push-manifest
push-manifest: $(MANIFESTTOOL)
	@$(MANIFESTTOOL) $(MANIFESTAUTH) push from-args \
    	--platforms linux/amd64,linux/arm,linux/arm64 \
    	--template $(DOCKERIMAGE)-ARCH \
    	--target $(DOCKERIMAGE)

.PHONY: release
release:
	@${MAKE} -B DOCKERIMAGE=pulcy/$(shell pulsar docker-tag) all push-manifest
