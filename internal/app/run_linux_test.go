package app

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/andersonmrodrigues/matrix-rain/internal/config"
)

const (
	enterAlt = "\x1b[?1049h"
	leaveAlt = "\x1b[?1049l"
)

// TestHelperProcess is the child: it runs the real app on its stdin/stdout.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("CCMATRIX_HELPER") != "1" {
		t.Skip("helper for the pty tests")
	}
	if os.Getenv("CCMATRIX_HELPER_PANIC") == "1" {
		panicHook = func() { panic("helper panic") }
	}
	cfg, _ := config.Parse(nil, os.Stderr)
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
