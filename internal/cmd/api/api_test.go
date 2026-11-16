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
	// only the workflow commands sit at the top level; every endpoint is
	// under `poeditor api`
	if len(names) != 3 || !names["pull"] || !names["push"] || !names["api"] {
		t.Errorf("api module commands = %v, want pull, push and api", names)
	}
}
