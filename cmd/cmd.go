package cmd

import (
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
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
			dir, _ := cmd.Flags().GetString("directory")
			output, _ := cmd.Flags().GetString("output")
			recursive, _ := cmd.Flags().GetBool("recursive")
			if recursive {
				return pkg.RunDirTree(out, dir, output, version)
			}
			out, close, err := pkg.Output(out, dir, output)
			if err != nil {
				log.WithFields(log.Fields{"err": err, "filename": output}).Fatal("failed to create file")
			}
			if close != nil {
				defer close()
			}
			return pkg.Run(out, dir, version)
		},
	}
	cmd.Flags().StringP("directory", "d", ".", "root directory")
	cmd.Flags().StringP("output", "o", "", "write output to file")
	cmd.Flags().BoolP("recursive", "r", false, "go directories recursively")
	cmd.Flags().BoolP("version", "v", false, "print go2md version")
	return cmd
}
