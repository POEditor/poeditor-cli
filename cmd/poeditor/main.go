// Command poeditor is the POEditor CLI.
package main

import (
	"fmt"
	"os"

	"github.com/POEditor/poeditor-cli/internal/cmd/root"
	"github.com/POEditor/poeditor-cli/internal/cmdutil"
)

func main() {
	f := cmdutil.New()
	if err := root.NewCmdRoot(f).Execute(); err != nil {
		fmt.Fprintln(f.IO.ErrOut, "error:", err)
		os.Exit(1)
	}
}
