package main

import (
	"fmt"
	"os"

	"github.com/rkuska/carn/internal/app"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		if _, err := fmt.Fprintln(os.Stdout, app.VersionInfo()); err != nil {
			os.Exit(1)
		}
		return
	}

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
