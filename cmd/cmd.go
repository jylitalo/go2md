package cmd

import (
	"fmt"
	"io"
	"os"

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
			if flag, _ := cmd.Flags().GetString("output"); flag != "" {
				fout, err := os.Create(flag)
				if err != nil {
					log.WithFields(log.Fields{"err": err, "filename": flag}).Fatal("failed to create file")
				}
				defer fout.Close()
				out = fout
			}
			return pkg.Run(out, version)
		},
	}
	cmd.Flags().BoolP("version", "v", false, "print go2md version")
	cmd.Flags().StringP("output", "o", "", "write output to file")
	return cmd
}
