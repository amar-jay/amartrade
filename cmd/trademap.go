package cmd
import (
	"github.com/amar-jay/amartrade/internal/trademap"
	"github.com/spf13/cobra"
)

func NewTradeMapCmd() *cobra.Command {
	tm := &cobra.Command{Use: "trademap", Short: "Query ITC Trade Map", Args: cobra.NoArgs}
	tm.AddCommand(newSearchCmd())
	goods := &cobra.Command{Use: "goods", Short: "Query merchandise trade", Args: cobra.NoArgs}
	goods.AddCommand(newGoodsFlowCmd("exports", "E"), newGoodsFlowCmd("imports", "I"))
	tm.AddCommand(goods)
	return tm
}

func newSearchCmd() *cobra.Command {
	var opts trademap.SearchOptions
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Trade Map economies, groups, and HS products",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Query = args[0]
			results, err := trademap.NewClient(nil).Search(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return trademap.WriteSearch(cmd.OutOrStdout(), results, opts.Format)
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.Type, "type", "all", "result type: all, economy, economy-group, product, or product-group")
	f.IntVar(&opts.Limit, "limit", 20, "maximum number of results; 0 means unlimited")
	f.StringVar(&opts.Format, "format", "jsonl", "output format: json, jsonl, or csv")
	f.BoolVar(&opts.Children, "children", false, "include descendants of an exact HS code")
	return cmd
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
	f.StringVar(&opts.Format, "format", "jsonl", "output format: json, jsonl, or csv")
	f.BoolVar(&opts.Raw, "raw", false, "include the detailed JSON envelope and provider metadata")
	f.IntVar(&opts.HSLevel, "hs-level", 2, "HS depth for --by product: 2, 4, 6, or 10")
	f.StringVar(&opts.DirectMirror, "data", "direct", "data source: direct, mirror, or mixed")
	_ = cmd.MarkFlagRequired("years")
	return cmd
}