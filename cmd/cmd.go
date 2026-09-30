package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func Execute(version, date string) error {
	return NewRoot(version, date, os.Stdout, os.Stderr).Execute()
}

func NewRoot(version, date string, stdout, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{Use: "amartrade", Short: "Query and analyze international trade data", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.AddCommand(&cobra.Command{Use: "version", Short: "Print version information", Args: cobra.NoArgs, Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "amartrade %s (%s)\n", version, date)
	}})
	tm := NewTradeMapCmd()
	root.AddCommand(tm)
	return root
}
