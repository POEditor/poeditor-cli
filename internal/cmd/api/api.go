// Package api is the API module: every command that talks to POEditor API v2.
// The workflow commands, pull and push, sit at the top level of the CLI;
// every endpoint is `poeditor api <group> <action>`. Add a group of
// dedicated commands here, as internal/cmd/api/<group>/.
package api

import (
	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmd/api/contributors"
	"github.com/POEditor/poeditor-cli/internal/cmd/api/languages"
	"github.com/POEditor/poeditor-cli/internal/cmd/api/projects"
	"github.com/POEditor/poeditor-cli/internal/cmd/api/pull"
	"github.com/POEditor/poeditor-cli/internal/cmd/api/push"
	"github.com/POEditor/poeditor-cli/internal/cmd/api/terms"
	"github.com/POEditor/poeditor-cli/internal/cmd/api/translations"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// Module is the API module.
var Module = cmdutil.Module{
	Name:  "api",
	Short: "POEditor API v2 commands",
	Commands: func(f *cmdutil.Factory) []*cobra.Command {
		return []*cobra.Command{
			pull.NewCmdPull(f),
			push.NewCmdPush(f),
			newCmdAPI(f),
		}
	},
}

// newCmdAPI returns `poeditor api`: one command per API v2 endpoint, from
// the group packages.
func newCmdAPI(f *cmdutil.Factory) *cobra.Command {
	cmd := cmdutil.NewGroup("api <group> <action>", "Call any POEditor API v2 endpoint")
	cmd.Long = `Every POEditor API v2 endpoint, as <group> <action>: the API's
projects/list is ` + "`poeditor api projects list`" + `, terms/add_comment is
` + "`poeditor api terms add-comment`" + `.

To transfer the files of .poeditor.yml, use ` + "`poeditor pull`" + ` and ` + "`poeditor push`" + `.`
	cmd.Example = `  poeditor api projects list
  poeditor api terms list --project web --language fr
  poeditor api contributors list --project 7717`
	must(cmdutil.AddUnique(cmd, contributors.NewCmdContributors(f), languages.NewCmdLanguages(f), projects.NewCmdProjects(f), terms.NewCmdTerms(f), translations.NewCmdTranslations(f)))
	return cmd
}

// must panics on a command-name clash, like the root does: a programming
// error the test suite catches.
func must(err error) {
	if err != nil {
		panic(err)
	}
}
