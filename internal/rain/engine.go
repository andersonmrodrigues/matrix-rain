// Package rain holds the falling-text state. It knows nothing about terminals.
package rain

import "math/rand/v2"

// Kind says how a cell is drawn.
type Kind uint8

const (
	Blank Kind = iota
	Trail
	Head
)

// Cell is one character position on the grid.
type Cell struct {
	Ch   rune
	Kind Kind
}

const (
	minTrail = 3 // trail rows behind the head
	maxTrail = 7
	maxSpeed = 4 // a column moves one row every 1..maxSpeed ticks
)

var glyphs = []rune("abcdefghijklmnopqrstuvwxyz0123456789@#$%&*+=<>?")

// Katakana is the half-width katakana range U+FF66..U+FF9D, one cell wide each.
var Katakana = func() []rune {
	var r []rune
	for c := '\uff66'; c <= '\uff9d'; c++ {
		r = append(r, c)
	}
	return r
}()

type column struct {
	head  int // row of the head; negative while still above the screen
	trail int // trail rows behind the head
	speed int // ticks per step
	ticks int // ticks since last step
}

// Engine is a grid of falling columns driven by an injected RNG.
type Engine struct {
	w, h   int
	glyphs []rune
	rng    *rand.Rand
	cols   []column
	cells  []Cell
}

// New builds an engine of w columns by h rows. Every column gets its own
// random start, speed and trail length.
func New(w, h int, rng *rand.Rand) *Engine {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	e := &Engine{w: w, h: h, glyphs: glyphs, rng: rng, cols: make([]column, w), cells: make([]Cell, w*h)}
	for x := range e.cols {
		e.cols[x] = e.newColumn()
		// Spread starts over the screen height so columns do not move in lockstep.
		e.cols[x].head = -e.rng.IntN(max(h, 1))
	}
	return e
}

// SetGlyphs replaces the characters heads are drawn from. It does not change
// how many random numbers a tick uses.
func (e *Engine) SetGlyphs(g []rune) { e.glyphs = g }

// Size returns the grid width and height.
func (e *Engine) Size() (w, h int) { return e.w, e.h }

// At returns the cell at column x, row y. Out of range returns a blank cell.
func (e *Engine) At(x, y int) Cell {
	if x < 0 || x >= e.w || y < 0 || y >= e.h {
		return Cell{}
	}
	return e.cells[y*e.w+x]
}

func (e *Engine) set(x, y int, c Cell) {
	if y >= 0 && y < e.h {
		e.cells[y*e.w+x] = c
	}
}

func (e *Engine) newColumn() column {
	return column{
		head:  -1 - e.rng.IntN(max(e.h, 1)),
		trail: minTrail + e.rng.IntN(maxTrail-minTrail+1),
		speed: 1 + e.rng.IntN(maxSpeed),
	}
}

// Tick advances every column that is due: the old head turns into trail, a new
// head is drawn, the row that fell out of the trail is erased, and a column
// whose trail has left the screen is recycled above it.
func (e *Engine) Tick() {
	for x := range e.cols {
		c := &e.cols[x]
		c.ticks++
		if c.ticks < c.speed {
			continue
		}
		c.ticks = 0
		c.head++
		if old := e.At(x, c.head-1); old.Kind == Head {
			e.set(x, c.head-1, Cell{Ch: old.Ch, Kind: Trail})
		}
		if c.head >= 0 && c.head < e.h {
			e.set(x, c.head, Cell{Ch: e.glyphs[e.rng.IntN(len(e.glyphs))], Kind: Head})
		}
		e.set(x, c.head-c.trail-1, Cell{})
		if c.head-c.trail >= e.h {
			*c = e.newColumn()
		}
	}
}
