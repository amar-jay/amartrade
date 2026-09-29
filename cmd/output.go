package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/amar-jay/amartrade/internal/trademap/timeseries"
)

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func writeJSONL(writer io.Writer, values []any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	for _, value := range values {
		if err := encoder.Encode(value); err != nil {
			return err
		}
	}
	return nil
}

type tableRow struct {
	Code  string
	Label string
	Extra string
}

func writeCodeTable(writer io.Writer, rows []tableRow) error {
	table := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "CODE\tLABEL\tDETAIL"); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\n", row.Code, row.Label, row.Extra); err != nil {
			return err
		}
	}
	return table.Flush()
}

func writeTimeSeries(writer io.Writer, format string, pages []*timeseries.Page) error {
	switch format {
	case "json":
		return writeJSON(writer, map[string]any{"pages": pages})
	case "jsonl":
		lines := make([]any, 0)
		for _, page := range pages {
			for _, record := range page.Records {
				lines = append(lines, recordLine(page, record, false))
			}
			for _, record := range page.AggregateRecords {
				lines = append(lines, recordLine(page, record, true))
			}
		}
		return writeJSONL(writer, lines)
	default:
		return writeTimeSeriesTable(writer, pages)
	}
}

func recordLine(page *timeseries.Page, record timeseries.Record, aggregate bool) map[string]any {
	return map[string]any{
		"page": page.Number, "aggregate": aggregate,
		"reporter": record.ReporterCode, "partner": record.PartnerCode, "product": record.ProductCode,
		"reporterLabel": record.ReporterLabel, "partnerLabel": record.PartnerLabel, "productLabel": record.ProductLabel,
		"values": record.Data,
	}
}

func writeTimeSeriesTable(writer io.Writer, pages []*timeseries.Page) error {
	periods := map[int]struct{}{}
	ordered := make([]int, 0)
	for _, page := range pages {
		for _, record := range append(append([]timeseries.Record{}, page.Records...), page.AggregateRecords...) {
			for _, value := range record.Data {
				if _, seen := periods[value.Period]; seen {
					continue
				}
				periods[value.Period] = struct{}{}
				ordered = append(ordered, value.Period)
			}
		}
	}
	table := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	header := "REPORTER\tPARTNER\tPRODUCT\t" + strings.Join(periodHeaders(ordered), "\t")
	if _, err := fmt.Fprintln(table, header); err != nil {
		return err
	}
	for _, page := range pages {
		if err := writeRecordRows(table, page.Records, ordered); err != nil {
			return err
		}
		if err := writeRecordRows(table, page.AggregateRecords, ordered); err != nil {
			return err
		}
	}
	return table.Flush()
}

func periodHeaders(periods []int) []string {
	headers := make([]string, len(periods))
	for index, period := range periods {
		headers[index] = strconv.Itoa(period)
	}
	return headers
}

func writeRecordRows(table io.Writer, records []timeseries.Record, periods []int) error {
	for _, record := range records {
		amounts := map[int]string{}
		for _, value := range record.Data {
			amounts[value.Period] = value.Amount.String()
		}
		cells := make([]string, 0, 3+len(periods))
		cells = append(cells, record.ReporterCode, record.PartnerCode, record.ProductCode)
		for _, period := range periods {
			cells = append(cells, amounts[period])
		}
		if _, err := fmt.Fprintln(table, strings.Join(cells, "\t")); err != nil {
			return err
		}
	}
	return nil
}
