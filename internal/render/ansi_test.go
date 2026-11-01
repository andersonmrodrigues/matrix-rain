package render

import (
	"bytes"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/andersonmrodrigues/matrix-rain/internal/rain"
)

var (
	csi = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
	cup = regexp.MustCompile(`\x1b\[(\d+);(\d+)H`)
)

func TestFrameInBounds(t *testing.T) {
	tests := []struct {
		name  string
		w, h  int
		ticks int
	}{
		{"typical", 80, 24, 60},
		{"wide", 200, 10, 30},
		{"tall", 5, 60, 100},
		{"single cell", 1, 1, 20},
		{"empty width", 0, 5, 5},
		{"empty height", 5, 0, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := rain.New(tt.w, tt.h, rand.New(rand.NewPCG(9, 9)))
			for i := 0; i < tt.ticks; i++ {
				e.Tick()
			}
			var buf bytes.Buffer
			if err := Frame(&buf, e); err != nil {
				t.Fatal(err)
			}
			out := buf.String()

			// The only escapes allowed are cursor moves and colours: no terminal queries.
			for _, seq := range csi.FindAllString(out, -1) {
				if !cup.MatchString(seq) && !strings.HasSuffix(seq, "m") {
					t.Errorf("unexpected escape %q", seq)
				}
			}

			moves := cup.FindAllStringSubmatchIndex(out, -1)
			if len(moves) != tt.h {
				t.Fatalf("%d cursor moves, want %d", len(moves), tt.h)
			}
			for i, m := range moves {
				row, _ := strconv.Atoi(out[m[2]:m[3]])
				col, _ := strconv.Atoi(out[m[4]:m[5]])
				if row < 1 || row > tt.h || col != 1 {
					t.Errorf("cursor move to %d;%d outside %dx%d", row, col, tt.w, tt.h)
				}
				end := len(out)
				if i+1 < len(moves) {
					end = moves[i+1][0]
				}
				text := csi.ReplaceAllString(out[m[1]:end], "")
				if n := len([]rune(text)); n != tt.w {
					t.Errorf("row %d has %d cells, want %d", row, n, tt.w)
				}
			}
		})
	}
}

func TestFrameColours(t *testing.T) {
	e := rain.New(1, 10, rand.New(rand.NewPCG(1, 1)))
	for i := 0; i < 40; i++ {
		e.Tick()
		var buf bytes.Buffer
		if err := Frame(&buf, e); err != nil {
			t.Fatal(err)
		}
		heads := strings.Count(buf.String(), white)
		if heads > 1 {
			t.Fatalf("tick %d: %d white heads in one column", i, heads)
		}
	}
}
