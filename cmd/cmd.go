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
func NewCommand(writer io.WriteCloser, version string) *cobra.Command {
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			// parse flags
			if flag, _ := cmd.Flags().GetBool("version"); flag {
				_, _ = writer.Write([]byte(fmt.Sprintf("go2md %s\n", version)))
				return nil
			}
			dir, _ := cmd.Flags().GetString("directory")
			ignoreMain, _ := cmd.Flags().GetBool("ignore-main")
			output, _ := cmd.Flags().GetString("output")
			recursive, _ := cmd.Flags().GetBool("recursive")
			// execute
			outInput := pkg.OutputSettings{Default: writer, Directory: dir, Filename: output}
			if recursive {
				return pkg.RunDirTree(outInput, version, !ignoreMain)
			}
			return pkg.RunDirectory(outInput, version, !ignoreMain)
		},
	}
	cmd.Flags().StringP("directory", "d", ".", "root directory")
	cmd.Flags().StringP("output", "o", "", "write output to file")
	cmd.Flags().Bool("ignore-main", false, "ignore directory, if its main package")
	cmd.Flags().BoolP("recursive", "r", false, "go directories recursively")
	cmd.Flags().BoolP("version", "v", false, "print go2md version")
	return cmd
}
