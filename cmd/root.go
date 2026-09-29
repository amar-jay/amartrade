package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCommand builds the CLI command tree. Keeping construction in a
// function makes commands straightforward to exercise in tests and agents.
func NewRootCommand(version, date string) *cobra.Command {
	root := &cobra.Command{
		Use:           "amartrade",
		Short:         "Query and analyze trade data",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.Version = version
	root.SetVersionTemplate("{{with .Name}}{{printf \"%s \" .}}{{end}}{{printf \"%s\" .Version}}\n")
	root.AddCommand(newVersionCommand(version, date), newTradeMapCommand())

	return root
}

// Execute runs the CLI with the process standard streams.
func Execute(version, date string) error {
	return execute(NewRootCommand(version, date), os.Stdin, os.Stdout, os.Stderr)
}

func execute(command *cobra.Command, stdin io.Reader, stdout, stderr io.Writer) error {
	command.SetIn(stdin)
	command.SetOut(stdout)
	command.SetErr(stderr)
	return command.Execute()
}
