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
	for _, want := range []string{"projects", "pull", "push"} {
		if !names[want] {
			t.Errorf("%s missing from the api module; got %v", want, names)
		}
	}
}
