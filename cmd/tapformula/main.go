package main

import (
	"flag"
	"os"

	"github.com/rkuska/carn/internal/release"
)

func main() {
	var formula release.HomebrewSourceFormula

	flag.StringVar(&formula.URL, "url", "", "source archive url")
	flag.StringVar(&formula.SHA256, "sha256", "", "source archive sha256")
	flag.Parse()

	content, err := release.RenderHomebrewSourceFormula(formula)
	if err != nil {
		if _, writeErr := os.Stderr.WriteString(err.Error() + "\n"); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}

	if _, err := os.Stdout.WriteString(content); err != nil {
		os.Exit(1)
	}
}
