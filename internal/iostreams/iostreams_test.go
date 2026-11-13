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
