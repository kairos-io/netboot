# netboot/Makefile

THIS := $(abspath $(lastword $(MAKEFILE_LIST)))
HERE := $(patsubst %/,%,$(dir $(THIS)))

all:
	$(error Please request a specific thing, there is no default target)

.PHONY: update-ipxe
update-ipxe:
	# Build arm64 ipxe
	$(MAKE) CROSS_COMPILE=aarch64-linux-gnu- ARCH=arm64 -C third_party/ipxe/src bin-arm64-efi/ipxe.efi \
	EMBED=$(HERE)/boot.ipxe bin-arm64-efi/ipxe.efi
	# Build x86 ipxe. This helps with caching and avoids rebuilding on each run
	$(MAKE) -C third_party/ipxe/src bin-i386-efi/ipxe.efi
	$(MAKE) -C third_party/ipxe/src bin-x86_64-efi/ipxe.efi
	# Embed x86 artifacts with pixiecore
	$(MAKE) -C third_party/ipxe/src \
	EMBED=$(HERE)/boot.ipxe \
	bin/ipxe.pxe \
	bin/undionly.kpxe \
	bin-x86_64-efi/ipxe.efi \
	bin-i386-efi/ipxe.efi
	# Turn those generated binaries into go data
	mkdir -p data/
	cp third_party/ipxe/src/bin/ipxe.pxe data/ipxe.pxe
	cp third_party/ipxe/src/bin/undionly.kpxe data/undionly.kpxe
	cp third_party/ipxe/src/bin-arm64-efi/ipxe.efi data/arm64.ipxe.efi
	cp third_party/ipxe/src/bin-x86_64-efi/ipxe.efi data/amd64.ipxe.efi
	cp third_party/ipxe/src/bin-i386-efi/ipxe.efi data/i386.ipxe.efi
	cp third_party/ipxe/src/bin/ipxe.pxe data/
	go run github.com/go-bindata/go-bindata/go-bindata -prefix "data/" -o assets/bindata.go -pkg assets -nometadata -nomemcopy data/
	gofmt -s -w assets/bindata.go
	rm -rf data/
