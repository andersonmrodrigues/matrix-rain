package rain

import (
	"math/rand/v2"
	"reflect"
	"testing"
)

func seeded(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, 1)) }

// colKinds returns the kinds of column x from top to bottom.
func colKinds(e *Engine, x int) []Kind {
	_, h := e.Size()
	ks := make([]Kind, h)
	for y := range ks {
		ks[y] = e.At(x, y).Kind
	}
	return ks
}

// single returns a 1x10 engine whose only column is set up by hand.
func single(c column) *Engine {
	e := New(1, 10, seeded(1))
	e.cols[0] = c
	return e
}

func TestTickAdvance(t *testing.T) {
	tests := []struct {
		name  string
		col   column
		ticks int
		want  []Kind
	}{
		{"head enters screen", column{head: -1, trail: 3, speed: 1}, 1,
			[]Kind{Head, Blank, Blank, Blank, Blank, Blank, Blank, Blank, Blank, Blank}},
		{"head moves, old head becomes trail", column{head: -1, trail: 3, speed: 1}, 3,
			[]Kind{Trail, Trail, Head, Blank, Blank, Blank, Blank, Blank, Blank, Blank}},
		{"slow column waits for its turn", column{head: -1, trail: 3, speed: 2}, 1,
			[]Kind{Blank, Blank, Blank, Blank, Blank, Blank, Blank, Blank, Blank, Blank}},
		{"slow column steps every 2 ticks", column{head: -1, trail: 3, speed: 2}, 4,
			[]Kind{Trail, Head, Blank, Blank, Blank, Blank, Blank, Blank, Blank, Blank}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := single(tt.col)
			for i := 0; i < tt.ticks; i++ {
				e.Tick()
			}
			if got := colKinds(e, 0); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("kinds = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTickErase(t *testing.T) {
	e := single(column{head: -1, trail: 3, speed: 1})
	for i := 0; i < 8; i++ { // head reaches row 7
		e.Tick()
	}
	// head at row 7 -> trail rows 4..6; everything above is erased.
	want := []Kind{Blank, Blank, Blank, Blank, Trail, Trail, Trail, Head, Blank, Blank}
	if got := colKinds(e, 0); !reflect.DeepEqual(got, want) {
		t.Errorf("kinds = %v, want %v", got, want)
	}
}

func TestTickRecycle(t *testing.T) {
	e := single(column{head: -1, trail: 3, speed: 1})
	// Head needs 10 ticks to reach row 9, then trail+1 more to erase it and recycle.
	for i := 0; i < 10+3+1; i++ {
		e.Tick()
	}
	for y, k := range colKinds(e, 0) {
		if k != Blank {
			t.Fatalf("row %d not erased before recycle: %v", y, k)
		}
	}
	if c := e.cols[0]; c.head >= 0 || c.trail < minTrail || c.trail > maxTrail || c.speed < 1 || c.speed > maxSpeed {
		t.Errorf("column not recycled above screen: %+v", c)
	}
	// A recycled column falls again.
	for i := 0; i < 30; i++ {
		e.Tick()
		for y := 0; y < 10; y++ {
			if e.At(0, y).Kind == Head {
				return
			}
		}
	}
	t.Error("recycled column never produced a head")
}

func TestSeededEngine(t *testing.T) {
	a, b := New(40, 20, seeded(42)), New(40, 20, seeded(42))
	for i := 0; i < 100; i++ {
		a.Tick()
		b.Tick()
	}
	if !reflect.DeepEqual(a.cells, b.cells) {
		t.Error("same seed produced different grids")
	}

	starts, speeds := map[int]bool{}, map[int]bool{}
	for _, c := range New(40, 20, seeded(7)).cols {
		starts[c.head] = true
		speeds[c.speed] = true
	}
	if len(starts) < 5 || len(speeds) < 2 {
		t.Errorf("columns not independent: %d starts, %d speeds", len(starts), len(speeds))
	}
}

func TestInvariants(t *testing.T) {
	tests := []struct{ w, h int }{{1, 1}, {5, 3}, {40, 20}, {0, 10}, {10, 0}}
	for _, tt := range tests {
		e := New(tt.w, tt.h, seeded(3))
		for i := 0; i < 200; i++ {
			e.Tick()
			for x := 0; x < tt.w; x++ {
				heads, trails := 0, 0
				for _, k := range colKinds(e, x) {
					switch k {
					case Head:
						heads++
					case Trail:
						trails++
					}
				}
				if heads > 1 || trails > maxTrail {
					t.Fatalf("%dx%d tick %d col %d: %d heads, %d trail", tt.w, tt.h, i, x, heads, trails)
				}
			}
		}
	}
}
