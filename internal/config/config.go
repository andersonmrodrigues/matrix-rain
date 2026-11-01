// Package config parses command-line flags without touching the terminal.
package config

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/andersonmrodrigues/matrix-rain/internal/render"
)

const (
	// DefaultSpeed is the -u value used when the flag is absent.
	DefaultSpeed = 4
	// MaxSpeed is the slowest -u value.
	MaxSpeed  = 10
	speedStep = 10 * time.Millisecond
)

// Config holds the validated runtime settings.
type Config struct {
	// Speed is the -u value, 0 (fastest) to MaxSpeed.
	Speed int
	// Colour is the trail colour name, one of render.ColourNames.
	Colour string
}

// Delay maps a speed to the pause between two frames: speed x 10ms, so 0 means
// no deliberate sleep.
func Delay(speed int) time.Duration { return time.Duration(speed) * speedStep }

// Parse validates args (without the program name). Usage and errors are
// written to out. Positional arguments are rejected.
func Parse(args []string, out io.Writer) (Config, error) {
	fs := flag.NewFlagSet("ccmatrix", flag.ContinueOnError)
	fs.SetOutput(out)
	speed := fs.Int("u", DefaultSpeed, "update delay `0-10` in 10ms steps, 0 is fastest")
	colour := fs.String("C", "green", "trail `colour`: "+strings.Join(render.ColourNames, ", "))
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if *speed < 0 || *speed > MaxSpeed {
		err := fmt.Errorf("-u must be an integer from 0 to %d, got %d", MaxSpeed, *speed)
		fmt.Fprintln(out, err)
		return Config{}, err
	}
	if _, ok := render.Colour(*colour); !ok {
		err := fmt.Errorf("unsupported colour %q; supported: %s", *colour, strings.Join(render.ColourNames, ", "))
		fmt.Fprintln(out, err)
		return Config{}, err
	}
	return Config{Speed: *speed, Colour: *colour}, nil
}
