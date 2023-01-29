package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/jylitalo/go2md/pkg"
)

// NewCommand returns root level command.
// Supports `--version`.
// Default is to generate markdown from current directory.
func NewCommand(out io.Writer, version string) *cobra.Command {
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			if flag, _ := cmd.Flags().GetBool("version"); flag {
				out.Write([]byte(fmt.Sprintf("go2md %s\n", version)))
				return nil
			}
			return pkg.Run(out, version)
		},
	}
	cmd.Flags().BoolP("version", "v", false, "print go2md version")
	return cmd
}
