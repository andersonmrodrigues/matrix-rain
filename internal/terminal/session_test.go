package terminal

import (
	"bytes"
	"strings"
	"testing"
)

func TestRestoreSequence(t *testing.T) {
	var out bytes.Buffer
	s := &Session{out: &out}
	s.Restore()
	s.Restore() // second call must be a no-op

	want := []string{"\x1b[0m", "\x1b[?25h", "\x1b[?1049l"}
	got := out.String()
	for _, w := range want {
		if strings.Count(got, w) != 1 {
			t.Errorf("restore output %q: want exactly one %q", got, w)
		}
	}
}

func TestEnterHidesCursorOnAltScreen(t *testing.T) {
	for _, seq := range []string{"\x1b[?1049h", "\x1b[?25l"} {
		if !strings.Contains(enter, seq) {
			t.Errorf("enter lacks %q", seq)
		}
	}
}
