// Command em-data-viewer runs the em-data-viewer desktop widget: it resolves the
// config file path (an optional CLI argument overrides the default XDG
// path, FR2), then wires the config watcher, process runner and GTK4 UI
// together (internal/app) and blocks until the user quits.
package main

import (
	"fmt"
	"os"

	"em-data-viewer/internal/app"
)

func main() {
	// GSK's GPU renderers (vulkan/ngl) drop thin horizontal glyph strokes at
	// small sizes on this class of setup (e.g. the top bar of 5/7 at 12pt),
	// while the cairo renderer draws them correctly. The widget redraws at
	// ~1Hz, so software rendering costs nothing; an explicit GSK_RENDERER
	// from the environment still wins. The override also propagates to
	// runner-spawned child commands (they inherit the process environment) —
	// accepted trade-off: a configured command that launches a GTK app just
	// falls back to software rendering.
	if os.Getenv("GSK_RENDERER") == "" {
		os.Setenv("GSK_RENDERER", "cairo")
	}
	os.Exit(run(os.Args[1:]))
}

// run resolves the config path from an optional positional argument and
// enters the App's main loop, returning the process exit code.
func run(args []string) int {
	var explicit string
	if len(args) > 0 {
		explicit = args[0]
	}

	path, err := app.ResolveConfigPath(explicit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "em-data-viewer: %v\n", err)
		return 1
	}

	return app.New(path).Run()
}
