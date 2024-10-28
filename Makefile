# netboot/Makefile

THIS := $(abspath $(lastword $(MAKEFILE_LIST)))
HERE := $(patsubst %/,%,$(dir $(THIS)))

GOCMD:=go
GOMODULECMD:=GO111MODULE=on go

# Local customizations to the above.
ifneq ($(wildcard Makefile.defaults),)
include Makefile.defaults
endif

all:
	$(error Please request a specific thing, there is no default target)

.PHONY: ci-prepare
ci-prepare:
	$(GOCMD) get -u github.com/estesp/manifest-tool

.PHONY: build
build:
	$(GOMODULECMD) install -v ./cmd/pixiecore

.PHONY: test
test:
	$(GOMODULECMD) test ./...
	$(GOMODULECMD) test -race ./...

.PHONY: lint
lint:
	$(GOMODULECMD) tool vet .

REGISTRY=pixiecore
TAG=dev
.PHONY: ci-push-images
ci-push-images:
	make -f Makefile.inc push GOARCH=amd64   TAG=$(TAG)-amd64   BINARY=pixiecore REGISTRY=$(REGISTRY)
	make -f Makefile.inc push GOARCH=arm     TAG=$(TAG)-arm     BINARY=pixiecore REGISTRY=$(REGISTRY)
	make -f Makefile.inc push GOARCH=arm64   TAG=$(TAG)-arm64   BINARY=pixiecore REGISTRY=$(REGISTRY)
	make -f Makefile.inc push GOARCH=ppc64le TAG=$(TAG)-ppc64le BINARY=pixiecore REGISTRY=$(REGISTRY)
	make -f Makefile.inc push GOARCH=s390x   TAG=$(TAG)-s390x   BINARY=pixiecore REGISTRY=$(REGISTRY)
	manifest-tool push from-args --platforms linux/amd64,linux/arm,linux/arm64,linux/ppc64le,linux/s390x --template $(REGISTRY)/pixiecore:$(TAG)-ARCH --target $(REGISTRY)/pixiecore:$(TAG)

.PHONY: ci-config
ci-config:
	(cd .circleci && go run gen-config.go >config.yml)

.PHONY: update-ipxe
update-ipxe:
	# Build arm64 ipxe
	$(MAKE) CROSS_COMPILE=aarch64-linux-gnu- ARCH=arm64 -C third_party/ipxe/src bin-arm64-efi/ipxe.efi \
	EMBED=$(HERE)/pixiecore/boot.ipxe bin-arm64-efi/ipxe.efi
	# Build x86 ipxe. This helps with caching and avoids rebuildingon each run
	$(MAKE) -C third_party/ipxe/src bin-i386-efi/ipxe.efi
	$(MAKE) -C third_party/ipxe/src bin-x86_64-efi/ipxe.efi
	# Embed x86 artifacts with pixiecore?
	$(MAKE) -C third_party/ipxe/src \
	EMBED=$(HERE)/pixiecore/boot.ipxe \
	bin/ipxe.pxe \
	bin/undionly.kpxe \
	bin-x86_64-efi/ipxe.efi \
	bin-i386-efi/ipxe.efi
	# Turn those generated binaries into go data
	go run github.com/go-bindata/go-bindata/go-bindata -o out/ipxe/bindata.go -pkg ipxe -nometadata -nomemcopy \
	third_party/ipxe/src/bin/ipxe.pxe \
	third_party/ipxe/src/bin/undionly.kpxe \
	third_party/ipxe/src/bin-arm64-efi/ipxe.efi \
	third_party/ipxe/src/bin-x86_64-efi/ipxe.efi \
	third_party/ipxe/src/bin-i386-efi/ipxe.efi
	gofmt -s -w out/ipxe/bindata.go
