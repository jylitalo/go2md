package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jylitalo/go2md/pkg"
)

func NewCommand(version string) *cobra.Command {
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			if flag, _ := cmd.Flags().GetBool("version"); flag {
				fmt.Println("go2md", version)
				return nil
			}
			return pkg.Run()
		},
	}
	cmd.Flags().BoolP("version", "v", false, "print go2md version")
	return cmd
}
