// Package api is the API module: every command that talks to POEditor API v2.
// Its commands sit at the top level of the CLI (`poeditor projects list`).
// Add a new group here, as internal/cmd/api/<group>/.
package api

import (
	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmd/api/projects"
	"github.com/POEditor/poeditor-cli/internal/cmd/api/pull"
	"github.com/POEditor/poeditor-cli/internal/cmd/api/push"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// Module is the API module.
var Module = cmdutil.Module{
	Name:  "api",
	Short: "POEditor API v2 commands",
	Commands: func(f *cmdutil.Factory) []*cobra.Command {
		return []*cobra.Command{
			projects.NewCmdProjects(f),
			pull.NewCmdPull(f),
			push.NewCmdPush(f),
		}
	},
}
