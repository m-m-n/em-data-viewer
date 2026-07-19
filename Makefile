APP_NAME     := em-data-viewer
VERSION      ?= 0.1.0
ARCH         := $(shell dpkg --print-architecture)
BUILD_DIR    := build
PKG_ROOT     := $(BUILD_DIR)/$(APP_NAME)_$(VERSION)_$(ARCH)
DEB_FILE     := $(BUILD_DIR)/$(APP_NAME)_$(VERSION)_$(ARCH).deb
ICON_SRC     := assets/icon.svg
ICON_DIR     := $(BUILD_DIR)/icons
ICON_SIZES   := 48 64 128 256
EMBED_DIR    := $(BUILD_DIR)/embed-icons
EMBED_SIZES  := 16 32 48 64 128 256
GRESOURCE_XML := assets/icon.gresource.xml
GRESOURCE_OUT := internal/ui/icon.gresource

.PHONY: all build dpkg icons resources test clean

all: build

build: resources
	mkdir -p $(BUILD_DIR)
	CC=clang go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

# resources renders the six icon sizes go:embed needs and compiles them
# into internal/ui/icon.gresource (task0001 FR1). Both build-time tools are
# checked up front so a missing one fails fast, before any partial output.
resources:
	@command -v rsvg-convert >/dev/null 2>&1 || { echo "resources: rsvg-convert not found on PATH" >&2; exit 1; }
	@command -v glib-compile-resources >/dev/null 2>&1 || { echo "resources: glib-compile-resources not found on PATH" >&2; exit 1; }
	mkdir -p $(EMBED_DIR)
	for s in $(EMBED_SIZES); do \
		rsvg-convert -w $$s -h $$s -o $(EMBED_DIR)/$(APP_NAME)-$$s.png $(ICON_SRC); \
	done
	glib-compile-resources --sourcedir=$(EMBED_DIR) --target=$(GRESOURCE_OUT) $(GRESOURCE_XML)

# test guarantees the embedded resource bundle exists before go:embed is
# compiled (task0001 FR2): a fresh checkout has no internal/ui/icon.gresource.
test: resources
	go test ./...

icons:
	@command -v rsvg-convert >/dev/null 2>&1 || { echo "icons: rsvg-convert not found on PATH" >&2; exit 1; }
	mkdir -p $(ICON_DIR)
	for s in $(ICON_SIZES); do \
		rsvg-convert -w $$s -h $$s -o $(ICON_DIR)/$(APP_NAME)-$$s.png $(ICON_SRC); \
	done

dpkg: build icons
	rm -rf $(PKG_ROOT)
	mkdir -p $(PKG_ROOT)/DEBIAN $(PKG_ROOT)/usr/bin $(PKG_ROOT)/usr/share/doc/$(APP_NAME)
	cp $(BUILD_DIR)/$(APP_NAME) $(PKG_ROOT)/usr/bin/
	cp config.yaml.example $(PKG_ROOT)/usr/share/doc/$(APP_NAME)/
	for s in $(ICON_SIZES); do \
		mkdir -p $(PKG_ROOT)/usr/share/icons/hicolor/$${s}x$${s}/apps; \
		cp $(ICON_DIR)/$(APP_NAME)-$$s.png $(PKG_ROOT)/usr/share/icons/hicolor/$${s}x$${s}/apps/$(APP_NAME).png; \
	done
	mkdir -p $(PKG_ROOT)/usr/share/icons/hicolor/scalable/apps
	cp $(ICON_SRC) $(PKG_ROOT)/usr/share/icons/hicolor/scalable/apps/$(APP_NAME).svg
	mkdir -p $(PKG_ROOT)/usr/share/applications
	printf '[Desktop Entry]\n' > $(PKG_ROOT)/usr/share/applications/$(APP_NAME).desktop
	printf 'Name=eM Data Viewer\n' >> $(PKG_ROOT)/usr/share/applications/$(APP_NAME).desktop
	printf 'Comment=Data viewer application\n' >> $(PKG_ROOT)/usr/share/applications/$(APP_NAME).desktop
	printf 'Exec=%s\n' '$(APP_NAME)' >> $(PKG_ROOT)/usr/share/applications/$(APP_NAME).desktop
	printf 'Icon=%s\n' '$(APP_NAME)' >> $(PKG_ROOT)/usr/share/applications/$(APP_NAME).desktop
	printf 'Terminal=false\n' >> $(PKG_ROOT)/usr/share/applications/$(APP_NAME).desktop
	printf 'Type=Application\n' >> $(PKG_ROOT)/usr/share/applications/$(APP_NAME).desktop
	printf 'Categories=Utility;\n' >> $(PKG_ROOT)/usr/share/applications/$(APP_NAME).desktop
	printf 'StartupWMClass=%s\n' '$(APP_NAME)' >> $(PKG_ROOT)/usr/share/applications/$(APP_NAME).desktop
	chmod 644 $(PKG_ROOT)/usr/share/applications/$(APP_NAME).desktop
	printf '#!/bin/sh\n' > $(PKG_ROOT)/DEBIAN/postinst
	printf 'set -e\n' >> $(PKG_ROOT)/DEBIAN/postinst
	printf 'command -v gtk-update-icon-cache >/dev/null 2>&1 && gtk-update-icon-cache -f -t /usr/share/icons/hicolor || true\n' >> $(PKG_ROOT)/DEBIAN/postinst
	printf 'command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database /usr/share/applications || true\n' >> $(PKG_ROOT)/DEBIAN/postinst
	printf 'exit 0\n' >> $(PKG_ROOT)/DEBIAN/postinst
	chmod 755 $(PKG_ROOT)/DEBIAN/postinst
	printf '#!/bin/sh\n' > $(PKG_ROOT)/DEBIAN/postrm
	printf 'set -e\n' >> $(PKG_ROOT)/DEBIAN/postrm
	printf 'command -v gtk-update-icon-cache >/dev/null 2>&1 && gtk-update-icon-cache -f -t /usr/share/icons/hicolor || true\n' >> $(PKG_ROOT)/DEBIAN/postrm
	printf 'command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database /usr/share/applications || true\n' >> $(PKG_ROOT)/DEBIAN/postrm
	printf 'exit 0\n' >> $(PKG_ROOT)/DEBIAN/postrm
	chmod 755 $(PKG_ROOT)/DEBIAN/postrm
	printf 'Package: %s\n' '$(APP_NAME)' > $(PKG_ROOT)/DEBIAN/control
	printf 'Version: %s\n' '$(VERSION)' >> $(PKG_ROOT)/DEBIAN/control
	printf 'Architecture: %s\n' '$(ARCH)' >> $(PKG_ROOT)/DEBIAN/control
	printf 'Maintainer: %s\n' "$$(git config user.name) <$$(git config user.email)>" >> $(PKG_ROOT)/DEBIAN/control
	printf 'Depends: libgtk-4-1\n' >> $(PKG_ROOT)/DEBIAN/control
	printf 'Description: Data viewer application\n' >> $(PKG_ROOT)/DEBIAN/control
	dpkg-deb --build --root-owner-group $(PKG_ROOT) $(DEB_FILE)

clean:
	rm -rf $(BUILD_DIR)
