APP_NAME  := em-data-viewer
VERSION   ?= 0.1.0
ARCH      := $(shell dpkg --print-architecture)
BUILD_DIR := build
PKG_ROOT  := $(BUILD_DIR)/$(APP_NAME)_$(VERSION)_$(ARCH)
DEB_FILE  := $(BUILD_DIR)/$(APP_NAME)_$(VERSION)_$(ARCH).deb

.PHONY: all build dpkg clean

all: build

build:
	mkdir -p $(BUILD_DIR)
	CC=clang go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

dpkg: build
	rm -rf $(PKG_ROOT)
	mkdir -p $(PKG_ROOT)/DEBIAN $(PKG_ROOT)/usr/bin $(PKG_ROOT)/usr/share/doc/$(APP_NAME)
	cp $(BUILD_DIR)/$(APP_NAME) $(PKG_ROOT)/usr/bin/
	cp config.yaml.example $(PKG_ROOT)/usr/share/doc/$(APP_NAME)/
	printf 'Package: %s\n' '$(APP_NAME)' > $(PKG_ROOT)/DEBIAN/control
	printf 'Version: %s\n' '$(VERSION)' >> $(PKG_ROOT)/DEBIAN/control
	printf 'Architecture: %s\n' '$(ARCH)' >> $(PKG_ROOT)/DEBIAN/control
	printf 'Maintainer: %s\n' "$$(git config user.name) <$$(git config user.email)>" >> $(PKG_ROOT)/DEBIAN/control
	printf 'Depends: libgtk-4-1\n' >> $(PKG_ROOT)/DEBIAN/control
	printf 'Description: Data viewer application\n' >> $(PKG_ROOT)/DEBIAN/control
	dpkg-deb --build --root-owner-group $(PKG_ROOT) $(DEB_FILE)

clean:
	rm -rf $(BUILD_DIR)
