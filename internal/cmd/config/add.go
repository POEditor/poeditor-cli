package config

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
	"github.com/POEditor/poeditor-cli/internal/repoconfig"
)

// NewCmdAdd returns `poeditor config add`.
func NewCmdAdd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a project or a file to " + repoconfig.FileName + ", step by step",
		Long: `Add a project or a file to ` + repoconfig.FileName + `, step by step.

Only the new entry is written: comments and everything else already in the
file are kept as they are.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, issues, err := load(f)
			if err != nil {
				return err
			}
			if repoconfig.HasErrors(issues) {
				printIssues(f, issues)
				return errors.New("fix " + cfg.Path() + " first (see `poeditor config show`)")
			}

			w, err := newWizard(cmd, f, cfg.Dir())
			if err != nil {
				return err
			}
			what, err := w.ask.Select("What do you want to add?", []string{"A file to an existing project", "Another project"}, 0)
			if err != nil {
				return err
			}

			if what == 1 {
				project, err := w.askProject(cfg)
				if err != nil {
					return err
				}
				if err := repoconfig.AppendProject(cfg.Path(), project); err != nil {
					return err
				}
				w.say("Added project %s (%s) to %s.", project.Label(), fileCount(len(project.Files)), cfg.Path())
				return validateWritten(f, cfg.Path())
			}

			index := 0
			if len(cfg.Projects) > 1 {
				labels := make([]string, len(cfg.Projects))
				for i, p := range cfg.Projects {
					labels[i] = p.Label()
				}
				if index, err = w.ask.Select("Which project?", labels, 0); err != nil {
					return err
				}
			}
			project := cfg.Projects[index]

			account, err := f.ResolveFor(project.Account)
			if err != nil {
				return err
			}
			codes, _, err := w.projectLanguages(f.APIClientWithToken(account.Token), project.ID)
			if err != nil {
				return err
			}
			taken := map[string]bool{}
			for _, p := range cfg.Projects {
				for _, file := range p.Files {
					taken[file.Path] = true
				}
			}
			file, err := w.askFile(codes, taken)
			if err != nil {
				return err
			}
			if err := repoconfig.AppendFile(cfg.Path(), index, file); err != nil {
				return err
			}
			w.say("Added %s to project %s in %s.", file.Path, project.Label(), cfg.Path())
			return validateWritten(f, cfg.Path())
		},
	}
	cmdutil.MarkAction(cmd)
	return cmd
}

// validateWritten re-reads the file after an edit and reports any problem.
func validateWritten(f *cmdutil.Factory, path string) error {
	cfg, err := repoconfig.Load(path)
	if err != nil {
		return err
	}
	if issues := cfg.Validate(); repoconfig.HasErrors(issues) {
		printIssues(f, issues)
		return problemsError(cfg, issues)
	}
	return nil
}
