// Command data-viewer runs the data-viewer desktop widget: it resolves the
// config file path (an optional CLI argument overrides the default XDG
// path, FR2), then wires the config watcher, process runner and GTK4 UI
// together (internal/app) and blocks until the user quits.
package main

import (
	"fmt"
	"os"

	"data-viewer/internal/app"
)

func main() {
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
		fmt.Fprintf(os.Stderr, "data-viewer: %v\n", err)
		return 1
	}

	return app.New(path).Run()
}
