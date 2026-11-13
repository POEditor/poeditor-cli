package config

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

// NewCmdShow returns `poeditor config show`.
func NewCmdShow(f *cmdutil.Factory) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the projects and files of " + repoconfig.FileName + " and the account each one uses",
		Long: `Show the projects and files of ` + repoconfig.FileName + ` and the account each one uses.

The file is searched for in the current directory and then in each parent
directory. Problems in it are listed on stderr; errors make the command fail.
Nothing is sent to the API: use ` + "`poeditor config check`" + ` for that.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, issues, err := load(f)
			if err != nil {
				return err
			}

			if asJSON {
				if issues == nil {
					issues = []repoconfig.Issue{}
				}
				if err := cmdutil.PrintJSON(f.IO, showJSON{File: cfg.Path(), Projects: cfg.Projects, Issues: issues}); err != nil {
					return err
				}
			} else {
				defaultAccount := ""
				if userCfg, err := f.Config(); err == nil {
					defaultAccount = userCfg.DefaultAccount
				}
				table := cmdutil.NewTable(f.IO, "PROJECT", "ID", "ACCOUNT", "FORMAT", "PATH")
				for _, p := range cfg.Projects {
					account := accountLabel(f.Account, p.Account, defaultAccount)
					for _, file := range p.Files {
						table.AddRow(p.Label(), strconv.Itoa(p.ID), account, file.Format, file.Path)
					}
				}
				if err := table.Render(); err != nil {
					return err
				}
				printIssues(f, issues)
			}

			if repoconfig.HasErrors(issues) {
				return problemsError(cfg, issues)
			}
			return nil
		},
	}

	cmdutil.AddJSONFlag(cmd, &asJSON)
	return cmd
}

type showJSON struct {
	File     string               `json:"file"`
	Projects []repoconfig.Project `json:"projects"`
	Issues   []repoconfig.Issue   `json:"issues"`
}

// accountLabel is the account a project runs as, as shown to people.
func accountLabel(flag, project, defaultAccount string) string {
	switch {
	case flag != "":
		return flag + " (--account)"
	case project != "":
		return project
	case defaultAccount != "":
		return defaultAccount + " (default)"
	default:
		return "(default)"
	}
}
