package app

import "testing"

func TestHandleKey(t *testing.T) {
	cases := []struct {
		name  string
		k     byte
		speed int
		want  int
		exit  bool
		code  int
	}{
		{"digit 0", '0', 5, 0, false, 0},
		{"digit 9", '9', 5, 9, false, 0},
		{"digit 3", '3', 7, 3, false, 0},
		{"q", 'q', 4, 4, true, 0},
		{"ctrl-c", 0x03, 4, 4, true, 130},
		{"a", 'a', 4, 4, false, 0},
		{"x", 'x', 4, 4, false, 0},
		{"Q", 'Q', 4, 4, false, 0},
		{"just below 0", '/', 4, 4, false, 0},
		{"just above 9", ':', 4, 4, false, 0},
		{"nul", 0, 4, 4, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			speed, exit, code := handleKey(tc.k, tc.speed)
			if speed != tc.want || exit != tc.exit || code != tc.code {
				t.Errorf("handleKey(%q, %d) = %d, %v, %d; want %d, %v, %d",
					tc.k, tc.speed, speed, exit, code, tc.want, tc.exit, tc.code)
			}
		})
	}
}
