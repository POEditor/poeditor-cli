package api_test

import (
	"testing"

	apicmd "github.com/POEditor/poeditor-cli/internal/cmd/api"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/iostreams"
)

func TestModuleCommands(t *testing.T) {
	ios, _, _, _ := iostreams.Test()

	cmds := apicmd.Module.Commands(&cmdutil.Factory{IO: ios})

	names := map[string]bool{}
	for _, cmd := range cmds {
		if names[cmd.Name()] {
			t.Errorf("group %q registered twice", cmd.Name())
		}
		names[cmd.Name()] = true
	}
	if !names["projects"] {
		t.Errorf("projects group missing; got %v", names)
	}
}
