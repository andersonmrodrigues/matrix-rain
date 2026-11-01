package app

import "testing"

func TestHandleKey(t *testing.T) {
	cases := []struct {
		name  string
		k     byte
		speed int
		saver bool
		want  int
		exit  bool
		code  int
	}{
		{"digit 0", '0', 5, false, 0, false, 0},
		{"digit 9", '9', 5, false, 9, false, 0},
		{"digit 3", '3', 7, false, 3, false, 0},
		{"q", 'q', 4, false, 4, true, 0},
		{"ctrl-c", 0x03, 4, false, 4, true, 130},
		{"a", 'a', 4, false, 4, false, 0},
		{"x", 'x', 4, false, 4, false, 0},
		{"Q", 'Q', 4, false, 4, false, 0},
		{"space", ' ', 4, false, 4, false, 0},
		{"enter", '\r', 4, false, 4, false, 0},
		{"escape", 0x1b, 4, false, 4, false, 0},
		{"just below 0", '/', 4, false, 4, false, 0},
		{"just above 9", ':', 4, false, 4, false, 0},
		{"nul", 0, 4, false, 4, false, 0},
		{"saver space", ' ', 4, true, 4, true, 0},
		{"saver letter", 'a', 4, true, 4, true, 0},
		{"saver q", 'q', 4, true, 4, true, 0},
		{"saver digit", '5', 4, true, 4, true, 0},
		{"saver enter", '\r', 4, true, 4, true, 0},
		{"saver escape", 0x1b, 4, true, 4, true, 0},
		{"saver mouse report byte", '<', 4, true, 4, true, 0},
		{"saver nul", 0, 4, true, 4, true, 0},
		{"saver ctrl-c", 0x03, 4, true, 4, true, 130},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			speed, exit, code := handleKey(tc.k, tc.speed, tc.saver)
			if speed != tc.want || exit != tc.exit || code != tc.code {
				t.Errorf("handleKey(%q, %d, %v) = %d, %v, %d; want %d, %v, %d",
					tc.k, tc.speed, tc.saver, speed, exit, code, tc.want, tc.exit, tc.code)
			}
		})
	}
}
