# em-data-viewer

A desktop widget for Ubuntu / GNOME / Wayland, built with Go and gotk4
(GTK4). It runs shell commands defined in a YAML config file and displays
their stdout in a single undecorated window with multiple sections. The
config file is watched, and style and command changes apply live without
restarting.

## Features

- **Widget Behavior**
  - Config-driven sections: interval mode (replace on each run) and
    stream mode (append lines, bounded by `max_lines`)
  - Pango markup rendering of command output, with plain-text fallback
    for invalid markup
  - Live config reload: style and command changes apply without a
    restart; invalid configs keep the last good config and show an error
    banner
  - Drag anywhere on the window to move it; right-click opens a context
    menu (Quit, keep-above toggle)
  - Keep-above (always on top), config-driven with default on
    (X11/XWayland; inert on native Wayland)
  - Deferred startup presentation: the window appears once, already at
    its fitted size, instead of flashing oversized

- **Layout**
  - Dynamic window height that follows content, capped at the monitor
    height
  - Greedy natural-width row packing: sections pack densely by their
    actual widths and wrap to new rows as needed
  - Forced row breaks (`type: break` in config) to force a new row at a
    chosen position
  - Floating divider rules between rows and between sections in a row

- **Output Rendering**
  - Default output font size 11pt (config-overridable)
  - Trailing newlines stripped from all output paths; interior blank
    lines preserved

- **Desktop Integration**
  - Application icon rendered from a single SVG source at build time,
    with hicolor theme icons and a `.desktop` entry installed by the deb
    package
  - Icon embedded into the binary (GResource) so it also appears without
    installing the deb
  - Application ID matched to the `.desktop` filename for Wayland dock
    icon resolution

## Configuration

Config file location: `$XDG_CONFIG_HOME/em-data-viewer/config.yaml`,
falling back to `~/.config/em-data-viewer/config.yaml`. A CLI positional
argument overrides this path. See `config.yaml.example` for a sample
configuration.

## Build

```sh
make build   # build the binary into build/
make test    # go test ./... (with build-time resources generated first)
make dpkg    # build a .deb package into build/
```

Building requires `rsvg-convert` and `glib-compile-resources` on `PATH`.

## License

MIT — see [LICENSE](LICENSE).
