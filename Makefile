APP_NAME     := em-data-viewer
# APP_ID is the GApplication ID (internal/ui applicationID) and MUST match
# the shipped .desktop basename below for Wayland dock icon resolution.
APP_ID       := io.github.m_m_n.EmDataViewer
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
# EXT_UUID is the GNOME Shell extension identity (task0002 FR5/FR6): it is
# simultaneously the source directory basename below and the metadata.json
# "uuid" field (see feature-docs/wayland-keep-above-extension/IMPLEMENTATION.md
# Shared Components — this literal is the single source of truth).
EXT_UUID          := keep-above@em-data-viewer.m-m-n.github.io
EXT_SRC_DIR       := contrib/gnome-shell-extension/$(EXT_UUID)
EXT_INSTALL_DIR   := $(HOME)/.local/share/gnome-shell/extensions/$(EXT_UUID)
GNOME_EXTENSIONS_CLI := gnome-extensions

.PHONY: all build dpkg icons resources test clean install-extension uninstall-extension

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
	printf '[Desktop Entry]\n' > $(PKG_ROOT)/usr/share/applications/$(APP_ID).desktop
	printf 'Name=eM Data Viewer\n' >> $(PKG_ROOT)/usr/share/applications/$(APP_ID).desktop
	printf 'Comment=Data viewer application\n' >> $(PKG_ROOT)/usr/share/applications/$(APP_ID).desktop
	printf 'Exec=%s\n' '$(APP_NAME)' >> $(PKG_ROOT)/usr/share/applications/$(APP_ID).desktop
	printf 'Icon=%s\n' '$(APP_NAME)' >> $(PKG_ROOT)/usr/share/applications/$(APP_ID).desktop
	printf 'Terminal=false\n' >> $(PKG_ROOT)/usr/share/applications/$(APP_ID).desktop
	printf 'Type=Application\n' >> $(PKG_ROOT)/usr/share/applications/$(APP_ID).desktop
	printf 'Categories=Utility;\n' >> $(PKG_ROOT)/usr/share/applications/$(APP_ID).desktop
	printf 'StartupWMClass=%s\n' '$(APP_NAME)' >> $(PKG_ROOT)/usr/share/applications/$(APP_ID).desktop
	chmod 644 $(PKG_ROOT)/usr/share/applications/$(APP_ID).desktop
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

# install-extension copies the bundled GNOME Shell extension (task0001) into
# the current user's per-user extensions directory and enables it (FR5).
# Fresh installs are detected before the copy overwrites anything: either the
# install directory does not exist yet, or the running Shell does not yet
# know the uuid. On a fresh install, `enable` may be a no-op (the Shell has
# not scanned the new directory) so its failure is tolerated and re-login
# guidance is printed instead. For a known install, an `enable` failure is a
# real error: it is not discarded and fails the target.
install-extension:
	@command -v $(GNOME_EXTENSIONS_CLI) >/dev/null 2>&1 || { echo "install-extension: $(GNOME_EXTENSIONS_CLI) not found on PATH" >&2; exit 1; }
	@fresh=0; \
	[ -d "$(EXT_INSTALL_DIR)" ] || fresh=1; \
	$(GNOME_EXTENSIONS_CLI) list | grep -Fxq "$(EXT_UUID)" || fresh=1; \
	rm -rf "$(EXT_INSTALL_DIR)" && \
	mkdir -p "$(EXT_INSTALL_DIR)" && \
	cp -r "$(EXT_SRC_DIR)/." "$(EXT_INSTALL_DIR)/" && \
	if [ "$$fresh" = "1" ]; then \
		{ $(GNOME_EXTENSIONS_CLI) enable "$(EXT_UUID)" >/dev/null 2>&1 || true; }; \
		echo "install-extension: fresh install of $(EXT_UUID) -- log out and back in (required on Wayland) for GNOME Shell to load it"; \
	else \
		$(GNOME_EXTENSIONS_CLI) enable "$(EXT_UUID)" || { echo "install-extension: failed to enable $(EXT_UUID)" >&2; exit 1; }; \
	fi

# uninstall-extension disables (failure tolerated: Shell absent, extension
# unknown, or already disabled) and removes the installed extension
# directory (FR6). Running it when the extension is not installed exits 0.
uninstall-extension:
	@if command -v $(GNOME_EXTENSIONS_CLI) >/dev/null 2>&1; then \
		$(GNOME_EXTENSIONS_CLI) disable "$(EXT_UUID)" >/dev/null 2>&1 || true; \
	fi
	rm -rf "$(EXT_INSTALL_DIR)"
