package projects

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

// Project is one entry of the projects/list result.
type Project struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Public  int    `json:"public"`
	Open    int    `json:"open"`
	Created string `json:"created"`
}

// NewCmdList returns `poeditor projects list` (API v2 projects/list).
func NewCmdList(f *cmdutil.Factory) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the projects your token can access",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := f.APIClient()
			if err != nil {
				return err
			}

			var result struct {
				Projects []Project `json:"projects"`
			}
			if err := client.Post(cmd.Context(), "projects/list", nil, &result); err != nil {
				return err
			}

			if asJSON {
				enc := json.NewEncoder(f.IO.Out)
				enc.SetIndent("", "  ")
				if result.Projects == nil {
					result.Projects = []Project{}
				}
				return enc.Encode(result.Projects)
			}

			if len(result.Projects) == 0 {
				fmt.Fprintln(f.IO.Out, "No projects found.")
				return nil
			}
			w := tabwriter.NewWriter(f.IO.Out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tPUBLIC\tOPEN\tCREATED")
			for _, p := range result.Projects {
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", p.ID, p.Name, yesNo(p.Public), yesNo(p.Open), p.Created)
			}
			return w.Flush()
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "Output the projects as JSON")
	return cmd
}

func yesNo(v int) string {
	if v != 0 {
		return "yes"
	}
	return "no"
}
