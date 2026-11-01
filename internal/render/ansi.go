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
	green = "\x1b[32m"
)

// Frame draws the whole grid: each row is positioned with an absolute cursor
// move and holds exactly width characters, so nothing lands out of bounds.
func Frame(out io.Writer, e *rain.Engine) error {
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
					buf.WriteString(green)
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
