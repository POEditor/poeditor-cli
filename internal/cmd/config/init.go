package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

// NewCmdInit returns `poeditor config init`.
func NewCmdInit(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create " + repoconfig.FileName + " here, step by step",
		Long: `Create ` + repoconfig.FileName + ` in the current directory, step by step.

The wizard asks for the account, the project (from your project list), a short
name for it, its source language and its files. For each file it offers the
{lang} patterns it finds in this directory (locales/{lang}.json,
values-{lang}/strings.xml, …), guesses the format, and asks for the tags to
pull and the tags to put on pushed terms. Add more projects later with
` + "`poeditor config add`" + `.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := f.WorkingDir()
			if err != nil {
				return err
			}
			path := filepath.Join(dir, repoconfig.FileName)
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s already exists: use `poeditor config add` to add a project or a file", path)
			} else if !errors.Is(err, fs.ErrNotExist) {
				return err
			}

			w, err := newWizard(cmd, f, dir)
			if err != nil {
				return err
			}
			project, err := w.askProject(nil)
			if err != nil {
				return err
			}

			cfg := repoconfig.New(path, []repoconfig.Project{project})
			if issues := cfg.Validate(); repoconfig.HasErrors(issues) {
				printIssues(f, issues)
				return errors.New("the answers do not make a valid config; nothing was written")
			}
			if err := cfg.Create(); err != nil {
				return err
			}
			w.say("Wrote %s (project %s, %s).", path, project.Label(), fileCount(len(project.Files)))
			w.say("Next: `poeditor config check` to check it against POEditor.")
			return nil
		},
	}
	cmdutil.MarkAction(cmd)
	return cmd
}
