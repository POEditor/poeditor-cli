package iostreams_test

import (
	"testing"

	"github.com/POEditor/poeditor-cli/internal/iostreams"
)

func TestSystem_forceTTY(t *testing.T) {
	// Under `go test` stdout is never a terminal, so only the override can make it one.
	tests := []struct {
		value string
		want  bool
	}{{"", false}, {"0", false}, {"1", true}, {"true", true}}
	for _, tt := range tests {
		ios := iostreams.System(func(k string) string {
			if k == iostreams.ForceTTYEnv {
				return tt.value
			}
			return ""
		})
		if got := ios.IsStdoutTTY(); got != tt.want {
			t.Errorf("%s=%q: IsStdoutTTY = %v, want %v", iostreams.ForceTTYEnv, tt.value, got, tt.want)
		}
	}
}

func TestTest_defaultsToPipe(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	if ios.IsStdoutTTY() {
		t.Fatal("test streams should start as a pipe")
	}
	ios.SetStdoutTTY(true)
	if !ios.IsStdoutTTY() {
		t.Fatal("SetStdoutTTY(true) ignored")
	}
}

func TestSystem_terminalWidth(t *testing.T) {
	// Under `go test` stdout is not a terminal, so its size can't be read:
	// a forced TTY falls back to $COLUMNS, then to DefaultWidth.
	tests := []struct {
		name string
		env  map[string]string
		want int
	}{
		{"not a terminal: no width", map[string]string{}, 0},
		{"forced TTY uses $COLUMNS", map[string]string{iostreams.ForceTTYEnv: "1", "COLUMNS": "132"}, 132},
		{"forced TTY, bad $COLUMNS", map[string]string{iostreams.ForceTTYEnv: "1", "COLUMNS": "wide"}, iostreams.DefaultWidth},
		{"forced TTY, no $COLUMNS", map[string]string{iostreams.ForceTTYEnv: "1"}, iostreams.DefaultWidth},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios := iostreams.System(func(k string) string { return tt.env[k] })
			if got := ios.TerminalWidth(); got != tt.want {
				t.Errorf("width = %d, want %d", got, tt.want)
			}
		})
	}
}
