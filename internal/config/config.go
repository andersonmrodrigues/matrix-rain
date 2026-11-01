// Package config parses command-line flags without touching the terminal.
package config

import (
	"flag"
	"fmt"
	"io"
	"time"
)

// Config holds the validated runtime settings.
type Config struct {
	// Frame is the delay between two animation frames.
	Frame time.Duration
}

// Parse validates args (without the program name). Usage and errors are
// written to out. Step 1 has no flags yet; positional arguments are rejected.
func Parse(args []string, out io.Writer) (Config, error) {
	fs := flag.NewFlagSet("ccmatrix", flag.ContinueOnError)
	fs.SetOutput(out)
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	return Config{Frame: 50 * time.Millisecond}, nil
}
