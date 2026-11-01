package config

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		want       Config
		wantErr    bool
		wantStderr string
	}{
		{"no args", nil, Config{Speed: 4, Colour: "green"}, false, ""},
		{"speed 0", []string{"-u", "0"}, Config{Speed: 0, Colour: "green"}, false, ""},
		{"speed 10", []string{"-u", "10"}, Config{Speed: 10, Colour: "green"}, false, ""},
		{"speed 11", []string{"-u", "11"}, Config{}, true, "-u must be"},
		{"speed negative", []string{"-u", "-1"}, Config{}, true, "-u must be"},
		{"speed word", []string{"-u", "fast"}, Config{}, true, "invalid value"},
		{"speed float", []string{"-u", "2.5"}, Config{}, true, "invalid value"},
		{"colour red", []string{"-C", "red"}, Config{Speed: 4, Colour: "red"}, false, ""},
		{"colour black", []string{"-C", "black"}, Config{Speed: 4, Colour: "black"}, false, ""},
		{"colour and speed", []string{"-C", "cyan", "-u", "2"}, Config{Speed: 2, Colour: "cyan"}, false, ""},
		{"colour purple", []string{"-C", "purple"}, Config{}, true, "green, red, blue, white, yellow, cyan, magenta, black"},
		{"colour uppercase", []string{"-C", "Red"}, Config{}, true, "supported:"},
		{"unknown flag", []string{"-z"}, Config{}, true, ""},
		{"positional", []string{"foo"}, Config{}, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			cfg, err := Parse(tt.args, &out)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if cfg != tt.want {
				t.Errorf("cfg = %+v, want %+v", cfg, tt.want)
			}
			if !strings.Contains(out.String(), tt.wantStderr) {
				t.Errorf("output %q lacks %q", out.String(), tt.wantStderr)
			}
			if strings.Contains(out.String(), "\x1b") {
				t.Errorf("output has escape sequences: %q", out.String())
			}
		})
	}
}

func TestDelay(t *testing.T) {
	tests := []struct {
		speed int
		want  time.Duration
	}{
		{0, 0},
		{1, 10 * time.Millisecond},
		{4, 40 * time.Millisecond},
		{10, 100 * time.Millisecond},
	}
	for _, tt := range tests {
		if got := Delay(tt.speed); got != tt.want {
			t.Errorf("Delay(%d) = %v, want %v", tt.speed, got, tt.want)
		}
	}
}
