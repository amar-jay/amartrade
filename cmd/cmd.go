package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/amar-jay/amartrade/internal/trademap"
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
	tm := &cobra.Command{Use: "trademap", Short: "Query ITC Trade Map", Args: cobra.NoArgs}
	goods := &cobra.Command{Use: "goods", Short: "Query merchandise trade", Args: cobra.NoArgs}
	goods.AddCommand(newGoodsFlowCmd("exports", "E"), newGoodsFlowCmd("imports", "I"))
	tm.AddCommand(goods)
	root.AddCommand(tm)
	return root
}

func newGoodsFlowCmd(name, flow string) *cobra.Command {
	var opts trademap.GoodsOptions
	cmd := &cobra.Command{Use: name, Short: "Fetch annual goods " + name, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		opts.Flow = flow
		result, err := trademap.NewClient(nil).Goods(cmd.Context(), opts)
		if err != nil {
			return err
		}
		return trademap.Write(cmd.OutOrStdout(), result, opts.Format, opts.Raw)
	}}
	f := cmd.Flags()
	f.StringVar(&opts.By, "by", "country", "expanded dimension: country, partner, or product")
	f.StringVar(&opts.From, "from", "WORLD", "reporter economy or group")
	f.StringVar(&opts.To, "to", "WORLD", "partner economy or group")
	f.StringVar(&opts.Product, "product", "ALL", "HS code, ALL, group label, or group:<id>")
	f.StringVar(&opts.Years, "years", "", "inclusive year or range, e.g. 2020:2024")
	f.StringVar(&opts.Format, "format", "json", "output format: json, jsonl, or csv")
	f.BoolVar(&opts.Raw, "raw", false, "include the detailed JSON envelope and provider metadata")
	f.IntVar(&opts.HSLevel, "hs-level", 2, "HS depth for --by product: 2, 4, 6, or 10")
	f.StringVar(&opts.DirectMirror, "data", "direct", "data source: direct, mirror, or mixed")
	_ = cmd.MarkFlagRequired("years")
	return cmd
}
