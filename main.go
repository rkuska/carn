package main

import (
	"fmt"
	"os"

	"github.com/rkuska/cldsrch/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
