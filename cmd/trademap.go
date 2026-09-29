package cmd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/amar-jay/amartrade/internal/trademap"
	"github.com/spf13/cobra"
)

func newTradeMapCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "trademap",
		Short: "Query the ITC Trade Map public data API",
		Long: `Query reference catalogs and goods or services time series from the
ITC Trade Map API. Data is written to stdout. Use --format json for
structured output.`,
	}
	flags := command.PersistentFlags()
	flags.String("base-url", "", "Trade Map API root (default https://www.trademap.org/api/)")
	flags.Duration("timeout", trademap.DefaultTimeout, "maximum time for one command, including retries")
	flags.Int("max-retries", trademap.DefaultMaxRetries, "retries after the first request for transient failures")
	flags.String("format", "table", "output format: table, json, or jsonl")
	command.AddCommand(newReferenceCommand(), newGoodsCommand(), newServicesCommand())
	return command
}

func tradeMapClient(command *cobra.Command) (*trademap.Client, error) {
	baseURL, err := command.Flags().GetString("base-url")
	if err != nil {
		return nil, err
	}
	timeout, err := command.Flags().GetDuration("timeout")
	if err != nil {
		return nil, err
	}
	retries, err := command.Flags().GetInt("max-retries")
	if err != nil {
		return nil, err
	}
	format, err := command.Flags().GetString("format")
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive")
	}
	switch format {
	case "table", "json", "jsonl":
	default:
		return nil, fmt.Errorf("format must be table, json, or jsonl")
	}
	options := []trademap.Option{
		trademap.WithHTTPClient(&http.Client{Timeout: timeout}),
		trademap.WithMaxRetries(retries),
	}
	if baseURL != "" {
		options = append(options, trademap.WithBaseURL(baseURL))
	}
	return trademap.NewClient(options...)
}

func commandContext(command *cobra.Command) (context.Context, context.CancelFunc) {
	parent := command.Context()
	if parent == nil {
		parent = context.Background()
	}
	timeout, err := command.Flags().GetDuration("timeout")
	if err != nil || timeout <= 0 {
		timeout = trademap.DefaultTimeout
	}
	return context.WithTimeout(parent, timeout)
}
