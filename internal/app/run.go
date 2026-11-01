// Package app wires the engine, renderer and terminal into the event loop.
package app

import (
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andersonmrodrigues/matrix-rain/internal/config"
	"github.com/andersonmrodrigues/matrix-rain/internal/rain"
	"github.com/andersonmrodrigues/matrix-rain/internal/render"
	"github.com/andersonmrodrigues/matrix-rain/internal/terminal"
)

// panicHook is set only by tests, to prove Restore runs when Run panics.
var panicHook func()

// Run draws the rain until q, SIGINT or SIGTERM and returns the exit code.
// The terminal is restored before it returns, also when a panic unwinds it, so
// the caller can call os.Exit right after.
func Run(cfg config.Config, in, out *os.File) int {
	// Registered before the terminal is touched and stopped only after
	// Restore (defers run last-in first-out), so a signal that arrives while
	// the session is opening is queued instead of killing the process raw.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigs)

	sess, err := terminal.Open(in, out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ccmatrix:", err)
		return 1
	}
	defer sess.Restore()

	w, h, err := sess.Size()
	if err != nil {
		sess.Restore()
		fmt.Fprintln(os.Stderr, "ccmatrix:", err)
		return 1
	}
	eng := rain.New(w, h, rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)))

	if cfg.Katakana {
		eng.SetGlyphs(rain.Katakana)
	}

	if panicHook != nil {
		panicHook()
	}

	trail, _ := render.Colour(cfg.Colour)
	speed := cfg.Speed
	keys := sess.Keys()
	// A timer per frame instead of a ticker, so speed 0 has no sleep and a
	// speed key applies from the next frame.
	frame := time.After(config.Delay(speed))

	for {
		select {
		case sig := <-sigs:
			if sig == syscall.SIGINT {
				return 130
			}
			return 143
		case k, ok := <-keys:
			if !ok {
				return 0
			}
			var exit bool
			var code int
			if speed, exit, code = handleKey(k, speed); exit {
				return code
			}
		case <-frame:
			eng.Tick()
			if err := render.Frame(out, eng, trail, cfg.Rainbow); err != nil {
				fmt.Fprintln(os.Stderr, "ccmatrix:", err)
				return 1
			}
			frame = time.After(config.Delay(speed))
		}
	}
}

// handleKey applies one key byte: a digit sets the speed, q exits 0, Ctrl-C
// exits 130 (raw mode turns off ISIG, so it arrives as a byte) and any other
// byte changes nothing.
func handleKey(k byte, speed int) (newSpeed int, exit bool, code int) {
	switch {
	case k == 'q':
		return speed, true, 0
	case k == 0x03:
		return speed, true, 130
	case k >= '0' && k <= '9':
		return int(k - '0'), false, 0
	}
	return speed, false, 0
}
