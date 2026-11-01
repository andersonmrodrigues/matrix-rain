// Command ccmatrix is a cmatrix clone: full-screen falling green text.
package main

import (
	"flag"
	"os"

	"github.com/andersonmrodrigues/matrix-rain/internal/app"
	"github.com/andersonmrodrigues/matrix-rain/internal/config"
)

func main() {
	cfg, err := config.Parse(os.Args[1:], os.Stderr)
	if err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		os.Exit(2)
	}
	// app.Run restores the terminal before returning, so exiting here is safe.
	os.Exit(app.Run(cfg, os.Stdin, os.Stdout))
}
