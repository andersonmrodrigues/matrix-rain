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

	if panicHook != nil {
		panicHook()
	}

	keys := sess.Keys()
	tick := time.NewTicker(cfg.Frame)
	defer tick.Stop()

	for {
		select {
		case sig := <-sigs:
			if sig == syscall.SIGINT {
				return 130
			}
			return 143
		case k, ok := <-keys:
			if !ok || k == 'q' {
				return 0
			}
			if k == 0x03 { // Ctrl-C: raw mode turns off ISIG, so it arrives as a byte
				return 130
			}
		case <-tick.C:
			eng.Tick()
			if err := render.Frame(out, eng); err != nil {
				fmt.Fprintln(os.Stderr, "ccmatrix:", err)
				return 1
			}
		}
	}
}
