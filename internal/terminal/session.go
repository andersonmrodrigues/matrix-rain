// Package terminal owns raw mode, the alternate screen, the cursor, input and
// size. Restore undoes all of it and is safe to call more than once.
package terminal

import (
	"errors"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/term"
)

const (
	// Alternate screen, hidden cursor, autowrap off, cleared.
	enter = "\x1b[?1049h\x1b[?25l\x1b[?7l\x1b[2J"
	// Mouse button reporting with SGR coordinates, used only by Open(mouse).
	mouseOn = "\x1b[?1000h\x1b[?1006h"
	// Mouse reporting off, colours reset, autowrap on, cursor visible, leave
	// alternate screen. Always written, so a mode left on is never leaked.
	leave = "\x1b[?1006l\x1b[?1000l\x1b[0m\x1b[?7h\x1b[?25h\x1b[?1049l"
)

// Session is an active full-screen terminal session.
type Session struct {
	in   *os.File
	out  io.Writer
	old  *term.State // nil when raw mode was never entered
	once sync.Once
}

// Open puts in into raw mode and switches out to the alternate screen with a
// hidden cursor, plus mouse reporting when mouse is set. It fails if in or out is not a terminal.
func Open(in, out *os.File, mouse bool) (*Session, error) {
	// A broken-pipe stdout would otherwise kill the process on SIGPIPE with
	// termios unrestored; ignored, the write fails with EPIPE instead.
	signal.Ignore(syscall.SIGPIPE)
	fd := int(in.Fd())
	if !term.IsTerminal(fd) {
		return nil, errors.New("stdin is not a terminal")
	}
	if !term.IsTerminal(int(out.Fd())) {
		return nil, errors.New("stdout is not a terminal")
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	s := &Session{in: in, out: out, old: old}
	seq := enter
	if mouse {
		seq += mouseOn
	}
	if _, err := io.WriteString(out, seq); err != nil {
		s.Restore()
		return nil, err
	}
	return s, nil
}

// Restore turns mouse reporting off, leaves the alternate screen, shows the
// cursor, resets colours and restores the saved termios. Only the first call
// does anything.
func (s *Session) Restore() {
	s.once.Do(func() {
		io.WriteString(s.out, leave)
		if s.old != nil {
			term.Restore(int(s.in.Fd()), s.old)
		}
	})
}

// Size returns the current terminal width and height in cells.
func (s *Session) Size() (w, h int, err error) {
	return term.GetSize(int(s.in.Fd()))
}

// Keys streams raw input bytes until the input is closed.
func (s *Session) Keys() <-chan byte {
	ch := make(chan byte, 16)
	go func() {
		defer close(ch)
		buf := make([]byte, 32)
		for {
			n, err := s.in.Read(buf)
			for _, b := range buf[:n] {
				ch <- b
			}
			if err != nil {
				return
			}
		}
	}()
	return ch
}
