// Package render turns an engine frame into ANSI escape codes. It only writes
// to an io.Writer and never queries the terminal.
package render

import (
	"bytes"
	"fmt"
	"io"

	"github.com/andersonmrodrigues/matrix-rain/internal/rain"
)

const (
	reset = "\x1b[0m"
	white = "\x1b[97m"
)

// ColourNames lists the supported trail colours.
var ColourNames = []string{"green", "red", "blue", "white", "yellow", "cyan", "magenta", "black"}

var colourCodes = map[string]string{
	"black":   "\x1b[30m",
	"red":     "\x1b[31m",
	"green":   "\x1b[32m",
	"yellow":  "\x1b[33m",
	"blue":    "\x1b[34m",
	"magenta": "\x1b[35m",
	"cyan":    "\x1b[36m",
	"white":   "\x1b[37m",
}

// Colour returns the escape sequence for a trail colour name.
func Colour(name string) (string, bool) {
	code, ok := colourCodes[name]
	return code, ok
}

// Frame draws the whole grid: each row is positioned with an absolute cursor
// move and holds exactly width characters, so nothing lands out of bounds. The
// head is always white; trail is the escape sequence for the trail colour.
func Frame(out io.Writer, e *rain.Engine, trail string) error {
	w, h := e.Size()
	var buf bytes.Buffer
	for y := 0; y < h; y++ {
		fmt.Fprintf(&buf, "\x1b[%d;1H", y+1)
		cur := rain.Blank
		buf.WriteString(reset)
		for x := 0; x < w; x++ {
			c := e.At(x, y)
			if c.Kind != cur {
				switch c.Kind {
				case rain.Head:
					buf.WriteString(white)
				case rain.Trail:
					buf.WriteString(trail)
				default:
					buf.WriteString(reset)
				}
				cur = c.Kind
			}
			if c.Kind == rain.Blank {
				buf.WriteByte(' ')
			} else {
				buf.WriteRune(c.Ch)
			}
		}
	}
	buf.WriteString(reset)
	_, err := out.Write(buf.Bytes())
	return err
}
