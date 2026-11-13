// Package config is the config module: `poeditor config ...` manages the
// repo's .poeditor.yml (which projects, which files, pull/push options).
package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

// Module is the config module, attached as `poeditor config`.
var Module = cmdutil.Module{
	Name:  "config",
	Short: "Set up and check this repo's " + repoconfig.FileName,
	Commands: func(f *cmdutil.Factory) []*cobra.Command {
		return []*cobra.Command{
			NewCmdInit(f),
			NewCmdAdd(f),
			NewCmdShow(f),
			NewCmdCheck(f),
		}
	},
}

// load finds, parses and validates .poeditor.yml from the working directory.
func load(f *cmdutil.Factory) (*repoconfig.Config, []repoconfig.Issue, error) {
	dir, err := f.WorkingDir()
	if err != nil {
		return nil, nil, err
	}
	path, err := repoconfig.Find(dir)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := repoconfig.Load(path)
	if err != nil {
		return nil, nil, err
	}
	return cfg, cfg.Validate(), nil
}

// printIssues writes issues to stderr, one per line.
func printIssues(f *cmdutil.Factory, issues []repoconfig.Issue) {
	for _, issue := range issues {
		fmt.Fprintln(f.IO.ErrOut, issue.String())
	}
}

// problemsError is the command error when the config has errors.
func problemsError(cfg *repoconfig.Config, issues []repoconfig.Issue) error {
	n := 0
	for _, issue := range issues {
		if issue.Severity == repoconfig.Error {
			n++
		}
	}
	if n == 1 {
		return fmt.Errorf("1 problem in %s", cfg.Path())
	}
	return fmt.Errorf("%d problems in %s", n, cfg.Path())
}
