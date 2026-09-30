package cmd

import (
	"fmt"
	"strings"

	"github.com/amar-jay/amartrade/internal/trademap/timeseries"
	"github.com/amar-jay/amartrade/internal/trademap/types"
	"github.com/spf13/cobra"
)

func outputFormat(command *cobra.Command) (string, error) {
	format, err := command.Flags().GetString("format")
	if err != nil {
		return "", err
	}
	switch format {
	case "table", "json", "jsonl":
		return format, nil
	default:
		return "", fmt.Errorf("format must be table, json, or jsonl")
	}
}

func parseFrequency(value string, allowMonthly bool) (types.Frequency, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "yearly", "year", "y":
		return types.Yearly, nil
	case "quarterly", "quarter", "q":
		return types.Quarterly, nil
	case "monthly", "month", "m":
		if !allowMonthly {
			return types.Frequency{}, fmt.Errorf("services time series does not support monthly frequency")
		}
		return types.Monthly, nil
	default:
		return types.Frequency{}, fmt.Errorf("frequency must be yearly, quarterly, or monthly")
	}
}

func parseGoodsDimension(value string) (types.Dimension, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "byproduct", "by-product", "product":
		return types.ByProduct, nil
	case "bycountry", "by-country", "country":
		return types.ByCountry, nil
	case "bypartner", "by-partner", "partner":
		return types.ByPartner, nil
	default:
		return types.Dimension{}, fmt.Errorf("dimension must be byProduct, byCountry, or byPartner")
	}
}

func parseServiceDimension(value string) (types.Dimension, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "byservice", "by-service", "service":
		return types.ByService, nil
	case "bycountry", "by-country", "country":
		return types.ByCountry, nil
	case "bypartner", "by-partner", "partner":
		return types.ByPartner, nil
	default:
		return types.Dimension{}, fmt.Errorf("dimension must be byService, byCountry, or byPartner")
	}
}

func parseFlow(value string) (types.TradeFlow, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "I", "IMPORT", "IMPORTS":
		return types.Imports, nil
	case "E", "EXPORT", "EXPORTS":
		return types.Exports, nil
	case "RE", "REEXPORT", "REEXPORTS", "RE-EXPORTS":
		return types.ReExports, nil
	case "TB", "BALANCE":
		return types.TradeBalance, nil
	default:
		return types.TradeFlow{}, fmt.Errorf("flow must be I, E, RE, or TB")
	}
}

func parseUnit(value string) (types.Unit, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "VAL", "VALUE":
		return types.ValueUnit, nil
	case "QTY", "QUANTITY":
		return types.QuantityUnit, nil
	case "GV", "GROWTH":
		return types.GrowthValueUnit, nil
	default:
		return types.Unit{}, fmt.Errorf("unit must be VAL, QTY, or GV")
	}
}

func parseDataMode(value string) (types.DataMode, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "D", "DIRECT":
		return types.DirectData, nil
	case "M", "MIRROR":
		return types.MirrorData, nil
	case "X", "MIXED":
		return types.MixedData, nil
	default:
		return types.DataMode{}, fmt.Errorf("data mode must be D, M, or X")
	}
}

func parseHSLevel(value string) (types.HSLevel, error) {
	switch strings.TrimSpace(value) {
	case "":
		return types.HSLevel{}, nil
	case "2":
		return types.HS2, nil
	case "4":
		return types.HS4, nil
	case "6":
		return types.HS6, nil
	case "10":
		return types.HS10, nil
	default:
		return types.HSLevel{}, fmt.Errorf("HS level must be 2, 4, 6, or 10")
	}
}

func parseServiceLevel(value string) (types.ServiceLevel, error) {
	switch strings.TrimSpace(value) {
	case "":
		return types.ServiceLevel{}, nil
	case "3":
		return types.ServiceLevel3, nil
	case "6":
		return types.ServiceLevel6, nil
	case "9":
		return types.ServiceLevel9, nil
	case "12":
		return types.ServiceLevel12, nil
	case "15":
		return types.ServiceLevel15, nil
	default:
		return types.ServiceLevel{}, fmt.Errorf("service level must be 3, 6, 9, 12, or 15")
	}
}

func parseSort(by, direction string) (timeseries.Sort, error) {
	by = strings.TrimSpace(by)
	direction = strings.TrimSpace(direction)
	if by == "" && direction == "" {
		return timeseries.Sort{}, nil
	}
	if by == "" || direction == "" {
		return timeseries.Sort{}, fmt.Errorf("sort-by and sort-dir must be set together")
	}
	switch strings.ToLower(direction) {
	case "asc", "ascending":
		return timeseries.Sort{By: by, Direction: types.SortAscending}, nil
	case "desc", "descending":
		return timeseries.Sort{By: by, Direction: types.SortDescending}, nil
	default:
		return timeseries.Sort{}, fmt.Errorf("sort-dir must be asc or desc")
	}
}

func missingFlagError(message string) error { return fmt.Errorf("%s", message) }
