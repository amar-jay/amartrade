package cmd

import (
	"fmt"
	"strconv"

	"github.com/amar-jay/amartrade/internal/trademap/reference"
	"github.com/spf13/cobra"
)

func newReferenceCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "reference",
		Short: "List Trade Map reference catalogs",
	}
	command.AddCommand(
		newCatalogCommand("economies", "List economies", runEconomies),
		newCatalogCommand("economy-groups", "List economy groups", runEconomyGroups),
		newCatalogCommand("products", "List HS products", runProducts),
		newCatalogCommand("product-groups", "List product groups", runProductGroups),
		newCatalogCommand("services", "List EBOPS services", runServices),
	)
	return command
}

type catalogRunner func(*cobra.Command, *reference.Service) error

func newCatalogCommand(use, short string, run catalogRunner) *cobra.Command {
	command := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			client, err := tradeMapClient(command)
			if err != nil {
				return err
			}
			return run(command, client.ReferenceData())
		},
	}
	flags := command.Flags()
	flags.String("code", "", "return the record with this exact code")
	flags.String("name", "", "return the record with this exact label")
	flags.String("search", "", "return fuzzy matches for this query")
	flags.Int("limit", 20, "maximum fuzzy matches")
	command.MarkFlagsMutuallyExclusive("code", "name", "search")
	return command
}

func runEconomies(command *cobra.Command, service *reference.Service) error {
	ctx, cancel := commandContext(command)
	defer cancel()
	catalog, err := service.Economies(ctx)
	if err != nil {
		return err
	}
	return emitCatalog(command, catalog, func(item reference.Economy) tableRow {
		return tableRow{Code: string(item.Code), Label: item.Label, Extra: availabilityText(item.YearlyGoods)}
	})
}

func runEconomyGroups(command *cobra.Command, service *reference.Service) error {
	ctx, cancel := commandContext(command)
	defer cancel()
	catalog, err := service.EconomyGroups(ctx)
	if err != nil {
		return err
	}
	return emitCatalog(command, catalog, func(item reference.EconomyGroup) tableRow {
		return tableRow{Code: string(item.Code), Label: item.Label, Extra: strconv.Itoa(len(item.Members)) + " members"}
	})
}

func runProducts(command *cobra.Command, service *reference.Service) error {
	ctx, cancel := commandContext(command)
	defer cancel()
	catalog, err := service.HSProducts(ctx)
	if err != nil {
		return err
	}
	return emitCatalog(command, catalog, func(item reference.HSProduct) tableRow {
		return tableRow{Code: string(item.Code), Label: item.Label, Extra: item.Revisions}
	})
}

func runProductGroups(command *cobra.Command, service *reference.Service) error {
	ctx, cancel := commandContext(command)
	defer cancel()
	catalog, err := service.ProductGroups(ctx)
	if err != nil {
		return err
	}
	return emitCatalog(command, catalog, func(item reference.ProductGroup) tableRow {
		return tableRow{Code: string(item.Code), Label: item.Label, Extra: strconv.Itoa(len(item.Products)) + " products"}
	})
}

func runServices(command *cobra.Command, service *reference.Service) error {
	ctx, cancel := commandContext(command)
	defer cancel()
	catalog, err := service.EBOPSServices(ctx)
	if err != nil {
		return err
	}
	return emitCatalog(command, catalog, func(item reference.EBOPSService) tableRow {
		return tableRow{Code: string(item.Code), Label: item.Label, Extra: item.DisplayCode}
	})
}

func emitCatalog[T any, C ~string](command *cobra.Command, catalog *reference.Catalog[T, C], row func(T) tableRow) error {
	format, err := outputFormat(command)
	if err != nil {
		return err
	}
	code, _ := command.Flags().GetString("code")
	name, _ := command.Flags().GetString("name")
	query, _ := command.Flags().GetString("search")
	limit, _ := command.Flags().GetInt("limit")
	switch {
	case code != "":
		item, err := catalog.ByCode(C(code))
		if err != nil {
			return err
		}
		return writeSelection(command, format, []T{item}, row)
	case name != "":
		item, err := catalog.ByName(name)
		if err != nil {
			return err
		}
		return writeSelection(command, format, []T{item}, row)
	case query != "":
		matches, err := catalog.Search(query, limit)
		if err != nil {
			return err
		}
		if format == "table" {
			rows := make([]tableRow, len(matches))
			for index, match := range matches {
				rows[index] = row(match.Item)
				rows[index].Extra = fmt.Sprintf("%.2f %s", match.Score, rows[index].Extra)
			}
			return writeCodeTable(command.OutOrStdout(), rows)
		}
		items := make([]T, len(matches))
		for index, match := range matches {
			items[index] = match.Item
		}
		if format == "jsonl" {
			values := make([]any, len(matches))
			for index, match := range matches {
				values[index] = match
			}
			return writeJSONL(command.OutOrStdout(), values)
		}
		return writeJSON(command.OutOrStdout(), matches)
	default:
		return writeSelection(command, format, catalog.All(), row)
	}
}

func writeSelection[T any](command *cobra.Command, format string, items []T, row func(T) tableRow) error {
	switch format {
	case "json":
		return writeJSON(command.OutOrStdout(), items)
	case "jsonl":
		values := make([]any, len(items))
		for index, item := range items {
			values[index] = item
		}
		return writeJSONL(command.OutOrStdout(), values)
	default:
		rows := make([]tableRow, len(items))
		for index, item := range items {
			rows[index] = row(item)
		}
		return writeCodeTable(command.OutOrStdout(), rows)
	}
}

func availabilityText(value reference.Availability) string {
	if value.FirstPeriod == 0 && value.LastPeriod == 0 {
		return ""
	}
	return fmt.Sprintf("%d-%d", value.FirstPeriod, value.LastPeriod)
}
