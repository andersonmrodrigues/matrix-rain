package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestMainHelper is the child: it runs the real main with the given flags.
func TestMainHelper(t *testing.T) {
	if os.Getenv("CCMATRIX_MAIN_HELPER") != "1" {
		t.Skip("helper for TestInvalidFlags")
	}
	os.Args = append([]string{"ccmatrix"}, strings.Fields(os.Getenv("CCMATRIX_ARGS"))...)
	main()
}

func TestInvalidFlags(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{"speed 11", "-u 11", "-u must be"},
		{"speed word", "-u fast", "invalid value"},
		{"colour purple", "-C purple", "supported: green, red, blue, white, yellow, cyan, magenta, black"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exe, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(exe, "-test.run=^TestMainHelper$")
			cmd.Env = append(os.Environ(), "CCMATRIX_MAIN_HELPER=1", "CCMATRIX_ARGS="+tt.args)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err = cmd.Run()
			if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() == 0 {
				t.Fatalf("err = %v, want non-zero exit", err)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout not empty: %q", stdout.String())
			}
			if strings.Contains(stderr.String(), "\x1b") {
				t.Errorf("stderr has escape sequences: %q", stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Errorf("stderr %q lacks %q", stderr.String(), tt.want)
			}
		})
	}
}
