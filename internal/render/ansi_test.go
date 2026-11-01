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
			if err := Frame(&buf, e, colourCodes["green"]); err != nil {
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
		if err := Frame(&buf, e, colourCodes["green"]); err != nil {
			t.Fatal(err)
		}
		heads := strings.Count(buf.String(), white)
		if heads > 1 {
			t.Fatalf("tick %d: %d white heads in one column", i, heads)
		}
	}
}

func TestColour(t *testing.T) {
	tests := []struct {
		name string
		want string
		ok   bool
	}{
		{"green", "\x1b[32m", true},
		{"red", "\x1b[31m", true},
		{"blue", "\x1b[34m", true},
		{"white", "\x1b[37m", true},
		{"yellow", "\x1b[33m", true},
		{"cyan", "\x1b[36m", true},
		{"magenta", "\x1b[35m", true},
		{"black", "\x1b[30m", true},
		{"purple", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Colour(tt.name)
			if got != tt.want || ok != tt.ok {
				t.Errorf("Colour(%q) = %q, %v", tt.name, got, ok)
			}
		})
	}
	if len(ColourNames) != len(colourCodes) {
		t.Errorf("%d names, %d codes", len(ColourNames), len(colourCodes))
	}
}

func TestFrameTrailColour(t *testing.T) {
	for _, name := range ColourNames {
		t.Run(name, func(t *testing.T) {
			code, _ := Colour(name)
			e := rain.New(1, 10, rand.New(rand.NewPCG(1, 1)))
			var buf bytes.Buffer
			for i := 0; i < 40; i++ {
				e.Tick()
				if err := Frame(&buf, e, code); err != nil {
					t.Fatal(err)
				}
			}
			out := buf.String()
			if !strings.Contains(out, code) {
				t.Errorf("no trail cell uses %q", code)
			}
			if !strings.Contains(out, white) {
				t.Error("no white head")
			}
			if strings.Contains(out, "\x1b[32m") && name != "green" {
				t.Error("default green leaked into trail")
			}
		})
	}
}
