package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unicode/utf8"

	"golang.org/x/sys/unix"

	"github.com/andersonmrodrigues/matrix-rain/internal/config"
)

const (
	enterAlt = "\x1b[?1049h"
	leaveAlt = "\x1b[?1049l"
	mouseOn  = "\x1b[?1000h"
	mouseOff = "\x1b[?1006l\x1b[?1000l"
)

// TestHelperProcess is the child: it runs the real app on its stdin/stdout.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("CCMATRIX_HELPER") != "1" {
		t.Skip("helper for the pty tests")
	}
	if os.Getenv("CCMATRIX_HELPER_PANIC") == "1" {
		panicHook = func() { panic("helper panic") }
	}
	cfg, _ := config.Parse(strings.Fields(os.Getenv("CCMATRIX_HELPER_ARGS")), os.Stderr)
	os.Exit(Run(cfg, os.Stdin, os.Stdout))
}

// syncBuf collects pty output while the test goroutine polls it.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// session is a helper process attached to a fresh 80x24 pty.
type session struct {
	t      *testing.T
	master *os.File
	slave  *os.File
	cmd    *exec.Cmd
	out    *syncBuf
	read   chan struct{} // closed when the master reaches EOF/EIO
	exited chan error
	before *unix.Termios // termios before the helper starts
	after  *unix.Termios // termios right after exit, before the slave closes
}

func openPty(t *testing.T) (master, slave *os.File) {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("no /dev/ptmx: %v", err)
	}
	t.Cleanup(func() { master.Close() })
	fd := int(master.Fd())
	if err := unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	n, err := unix.IoctlGetInt(fd, unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	slave, err = os.OpenFile("/dev/pts/"+itoa(n), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { slave.Close() })
	if err := unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 24, Col: 80}); err != nil {
		t.Fatal(err)
	}
	return master, slave
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for ; n > 0; n /= 10 {
		d = append([]byte{byte('0' + n%10)}, d...)
	}
	return string(d)
}

// start runs the helper with stdin on the pty slave. stdout is the slave too
// unless stdout is non-nil.
func start(t *testing.T, stdout *os.File, env ...string) *session {
	t.Helper()
	master, slave := openPty(t)
	s := &session{t: t, master: master, slave: slave, out: &syncBuf{}, read: make(chan struct{}), exited: make(chan error, 1)}

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	s.cmd = exec.Command(exe, "-test.run=^TestHelperProcess$")
	s.cmd.Env = append(os.Environ(), append([]string{"CCMATRIX_HELPER=1"}, env...)...)
	s.cmd.Stdin, s.cmd.Stdout, s.cmd.Stderr = slave, slave, slave
	if stdout != nil {
		s.cmd.Stdout = stdout
	}
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := master.Read(buf)
			s.out.Write(buf[:n])
			if err != nil {
				close(s.read)
				return
			}
		}
	}()
	s.before = s.termios()
	if err := s.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.cmd.Process.Kill() })
	go func() { s.exited <- s.cmd.Wait() }()
	return s
}

func (s *session) termios() *unix.Termios {
	s.t.Helper()
	tio, err := unix.IoctlGetTermios(int(s.slave.Fd()), unix.TCGETS)
	if err != nil {
		s.t.Fatal(err)
	}
	return tio
}

func (s *session) waitFor(sub string) {
	s.t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if strings.Contains(s.out.String(), sub) {
			return
		}
	}
	s.t.Fatalf("timeout waiting for %q; output %q", sub, s.out.String())
}

// wait returns the exit code, failing if the process outlives the deadline.
func (s *session) wait() int {
	s.t.Helper()
	select {
	case <-s.exited:
	case <-time.After(5 * time.Second):
		s.t.Fatal("process did not exit")
	}
	s.after = s.termios()
	s.slave.Close() // lets the master drain and hit EIO
	<-s.read
	return s.cmd.ProcessState.ExitCode()
}

func TestRestoredOnExit(t *testing.T) {
	cases := []struct {
		name string
		stop func(s *session)
		env  []string
		code int
	}{
		{"q", func(s *session) { s.master.WriteString("q") }, nil, 0},
		{"ctrl-c byte", func(s *session) { s.master.WriteString("\x03") }, nil, 130},
		{"SIGINT", func(s *session) { s.cmd.Process.Signal(syscall.SIGINT) }, nil, 130},
		{"SIGTERM", func(s *session) { s.cmd.Process.Signal(syscall.SIGTERM) }, nil, 143},
		{"panic", func(s *session) {}, []string{"CCMATRIX_HELPER_PANIC=1"}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := start(t, nil, tc.env...)
			before := s.before
			if before.Lflag&(unix.ECHO|unix.ICANON) != unix.ECHO|unix.ICANON {
				t.Fatalf("pty starts without ECHO/ICANON: %#x", before.Lflag)
			}
			s.waitFor(enterAlt)
			if tc.env == nil && s.termios().Lflag&(unix.ECHO|unix.ICANON) != 0 {
				t.Error("raw mode not active while running")
			}
			tc.stop(s)
			if code := s.wait(); code != tc.code {
				t.Errorf("exit code %d, want %d", code, tc.code)
			}
			if after := s.after; *after != *before {
				t.Errorf("termios not restored:\n before %+v\n after  %+v", before, after)
			}
			out := s.out.String()
			for _, seq := range []string{"\x1b[0m", "\x1b[?7h", "\x1b[?25h", leaveAlt} {
				if !strings.Contains(out, seq) {
					t.Errorf("output lacks cleanup %q", seq)
				}
			}
			if strings.LastIndex(out, leaveAlt) < strings.LastIndex(out, enterAlt) {
				t.Error("alternate screen left before it was entered")
			}
		})
	}
}

func TestMouseReporting(t *testing.T) {
	cases := []struct {
		name string
		args string
		on   bool
	}{
		{"without -s", "", false},
		{"with -s", "-s", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := start(t, nil, "CCMATRIX_HELPER_ARGS="+tc.args)
			s.waitFor(enterAlt)
			if tc.on {
				s.waitFor(mouseOn)
			}
			s.master.WriteString("\x03")
			s.wait()
			out := s.out.String()
			if got := strings.Contains(out, mouseOn); got != tc.on {
				t.Errorf("mouse enable present = %v, want %v", got, tc.on)
			}
			if !strings.Contains(out, mouseOff) {
				t.Error("output lacks the mouse disable sequence")
			}
		})
	}
}

func TestScreensaverExitsOnInput(t *testing.T) {
	for _, in := range []string{" ", "a", "5", "\r", "\x1b[A", "\x1b[<0;10;5M"} {
		t.Run(fmt.Sprintf("%q", in), func(t *testing.T) {
			s := start(t, nil, "CCMATRIX_HELPER_ARGS=-s")
			s.waitFor(mouseOn)
			s.master.WriteString(in)
			if code := s.wait(); code != 0 {
				t.Errorf("exit code %d, want 0", code)
			}
			if !strings.Contains(s.out.String(), mouseOff) {
				t.Error("output lacks the mouse disable sequence")
			}
		})
	}
}

var (
	csiSeq = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
	cupSeq = regexp.MustCompile(`\x1b\[(\d+);(\d+)H`)
)

// frameAfterClear checks the output written after a screen clear: the first
// frame must move the cursor to rows 1..rows in order and draw at most cols
// cells on each. With no rows it must not move the cursor at all.
func frameAfterClear(t *testing.T, after string, cols, rows int) {
	t.Helper()
	moves := cupSeq.FindAllStringSubmatchIndex(after, -1)
	if rows == 0 {
		if len(moves) != 0 {
			t.Errorf("%dx%d: %d cursor moves, want none", cols, rows, len(moves))
		}
		return
	}
	for i, m := range moves {
		row, _ := strconv.Atoi(after[m[2]:m[3]])
		col, _ := strconv.Atoi(after[m[4]:m[5]])
		if want := i%rows + 1; row != want {
			t.Fatalf("%dx%d: move %d goes to row %d, want %d", cols, rows, i, row, want)
		}
		if col != 1 {
			t.Errorf("%dx%d: move %d goes to column %d, want 1", cols, rows, i, col)
		}
		end := len(after)
		if i+1 < len(moves) {
			end = moves[i+1][0]
		}
		if n := utf8.RuneCountInString(csiSeq.ReplaceAllString(after[m[1]:end], "")); n > cols {
			t.Errorf("%dx%d: row %d draws %d cells", cols, rows, row, n)
		}
	}
	if len(moves) < rows {
		t.Errorf("%dx%d: %d cursor moves, want at least %d", cols, rows, len(moves), rows)
	}
}

func TestResizeKeepsRunning(t *testing.T) {
	s := start(t, nil)
	s.waitFor(enterAlt)
	for _, sz := range []unix.Winsize{{Row: 40, Col: 120}, {Row: 10, Col: 20}, {Row: 1, Col: 1}, {Row: 0, Col: 0}, {Row: 24, Col: 80}} {
		cols, rows := int(sz.Col), int(sz.Row)
		before := len(s.out.String())
		if err := unix.IoctlSetWinsize(int(s.slave.Fd()), unix.TIOCSWINSZ, &sz); err != nil {
			t.Fatal(err)
		}
		s.cmd.Process.Signal(syscall.SIGWINCH)
		// Wait for the clear, then for a whole frame (rows moves) after it.
		var after string
		for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(10 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatalf("%dx%d: no clear and frame after resize; output %q", cols, rows, s.out.String()[before:])
			}
			out := s.out.String()[before:]
			i := strings.Index(out, "\x1b[2J")
			if i < 0 {
				continue
			}
			after = out[i+len("\x1b[2J"):]
			if rows == 0 {
				// An empty frame has no moves; let one be drawn.
				if strings.Count(after, "\x1b[0m") >= 2 {
					break
				}
			} else if len(cupSeq.FindAllString(after, -1)) >= rows {
				break
			}
		}
		select {
		case <-s.exited:
			t.Fatalf("process exited on resize to %dx%d", cols, rows)
		default:
		}
		frameAfterClear(t, after, cols, rows)
	}
	s.master.WriteString("q")
	if code := s.wait(); code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
	if after := s.after; *after != *s.before {
		t.Errorf("termios not restored:\n before %+v\n after  %+v", s.before, after)
	}
}

// A pending SIGWINCH must not make the following SIGTERM get dropped.
func TestSigtermAfterResize(t *testing.T) {
	s := start(t, nil)
	s.waitFor(enterAlt)
	s.cmd.Process.Signal(syscall.SIGWINCH)
	s.cmd.Process.Signal(syscall.SIGTERM)
	if code := s.wait(); code != 143 {
		t.Errorf("exit code %d, want 143", code)
	}
	if after := s.after; *after != *s.before {
		t.Errorf("termios not restored:\n before %+v\n after  %+v", s.before, after)
	}
}

func TestOtherKeysIgnored(t *testing.T) {
	s := start(t, nil)
	s.waitFor(enterAlt)
	s.master.WriteString("ax")
	s.master.WriteString("0159") // speed keys change speed, never exit
	select {
	case <-s.exited:
		t.Fatal("process exited on a / x")
	case <-time.After(300 * time.Millisecond):
	}
	if strings.Contains(s.out.String(), leaveAlt) {
		t.Error("terminal restored while still running")
	}
	s.master.WriteString("q")
	if code := s.wait(); code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
}

func TestStdoutNotTerminal(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	r.Close() // broken pipe: writes would raise SIGPIPE
	defer w.Close()

	s := start(t, w)
	before := s.before
	if code := s.wait(); code != 1 {
		t.Errorf("exit code %d, want 1 (clean error)", code)
	}
	if after := s.after; *after != *before {
		t.Errorf("termios changed: %+v -> %+v", before, after)
	}
	if !strings.Contains(s.out.String(), "stdout is not a terminal") {
		t.Errorf("stderr %q lacks the error", s.out.String())
	}
}
