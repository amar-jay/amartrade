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

func parsePeriods(frequency types.Frequency, from, to string) (types.PeriodRange, error) {
	granularity := types.YearGranularity
	switch frequency {
	case types.Quarterly:
		granularity = types.QuarterGranularity
	case types.Monthly:
		granularity = types.MonthGranularity
	}
	start, err := types.ParsePeriod(granularity, strings.TrimSpace(from))
	if err != nil {
		return types.PeriodRange{}, fmt.Errorf("from: %w", err)
	}
	end, err := types.ParsePeriod(granularity, strings.TrimSpace(to))
	if err != nil {
		return types.PeriodRange{}, fmt.Errorf("to: %w", err)
	}
	return types.NewPeriodRange(start, end)
}

func exclusiveCode(command *cobra.Command, individual, group string) (string, bool, error) {
	one, err := command.Flags().GetString(individual)
	if err != nil {
		return "", false, err
	}
	grouped, err := command.Flags().GetString(group)
	if err != nil {
		return "", false, err
	}
	one, grouped = strings.TrimSpace(one), strings.TrimSpace(grouped)
	switch {
	case one != "" && grouped != "":
		return "", false, fmt.Errorf("--%s and --%s are mutually exclusive", individual, group)
	case one != "":
		return one, false, nil
	case grouped != "":
		return grouped, true, nil
	default:
		return "", false, fmt.Errorf("one of --%s or --%s is required", individual, group)
	}
}

func reporterSelector(command *cobra.Command) (types.ReporterSelector, error) {
	code, group, err := exclusiveCode(command, "reporter", "reporter-group")
	if err != nil {
		return types.ReporterSelector{}, err
	}
	if group {
		parsed, err := types.NewEconomyGroupCode(code)
		if err != nil {
			return types.ReporterSelector{}, err
		}
		return types.ReporterEconomyGroup(parsed)
	}
	parsed, err := types.NewEconomyCode(code)
	if err != nil {
		return types.ReporterSelector{}, err
	}
	return types.ReporterEconomy(parsed)
}

func partnerSelector(command *cobra.Command) (types.PartnerSelector, error) {
	code, group, err := exclusiveCode(command, "partner", "partner-group")
	if err != nil {
		return types.PartnerSelector{}, err
	}
	if group {
		parsed, err := types.NewEconomyGroupCode(code)
		if err != nil {
			return types.PartnerSelector{}, err
		}
		return types.PartnerEconomyGroup(parsed)
	}
	parsed, err := types.NewEconomyCode(code)
	if err != nil {
		return types.PartnerSelector{}, err
	}
	return types.PartnerEconomy(parsed)
}

func goodsSelector(command *cobra.Command) (types.GoodsSelector, error) {
	code, group, err := exclusiveCode(command, "product", "product-group")
	if err != nil {
		return types.GoodsSelector{}, err
	}
	if group {
		parsed, err := types.NewProductGroupCode(code)
		if err != nil {
			return types.GoodsSelector{}, err
		}
		return types.GoodsProductGroup(parsed)
	}
	parsed, err := types.NewHSProductCode(code)
	if err != nil {
		return types.GoodsSelector{}, err
	}
	return types.GoodsProduct(parsed)
}

func serviceSelector(command *cobra.Command) (types.ServiceSelector, error) {
	aggregate, err := command.Flags().GetBool("all-services")
	if err != nil {
		return types.ServiceSelector{}, err
	}
	code, err := command.Flags().GetString("service")
	if err != nil {
		return types.ServiceSelector{}, err
	}
	code = strings.TrimSpace(code)
	if aggregate && code != "" {
		return types.ServiceSelector{}, fmt.Errorf("--service and --all-services are mutually exclusive")
	}
	if aggregate {
		return types.AggregateServices(), nil
	}
	if code == "" {
		return types.ServiceSelector{}, fmt.Errorf("one of --service or --all-services is required")
	}
	parsed, err := types.NewServiceCode(code)
	if err != nil {
		return types.ServiceSelector{}, err
	}
	return types.Service(parsed)
}
