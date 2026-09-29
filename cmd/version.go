package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand(version, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build version information",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(
				command.OutOrStdout(),
				"amartrade %s (%s)\n",
				version,
				date,
			)
			return err
		},
	}
}
