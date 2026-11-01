package config

import (
	"io"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"no args", nil, false},
		{"unknown flag", []string{"-z"}, true},
		{"positional", []string{"foo"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Parse(tt.args, io.Discard)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && cfg.Frame != 50*time.Millisecond {
				t.Errorf("Frame = %v", cfg.Frame)
			}
		})
	}
}
