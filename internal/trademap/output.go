package trademap

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

type compactRecord struct {
	Reporter   string           `json:"reporter"`
	Partner    string           `json:"partner"`
	Product    string           `json:"product"`
	Aggregate  bool             `json:"aggregate"`
	Currency   string           `json:"currency"`
	ValueScale string           `json:"value_scale"`
	Values     map[string]int64 `json:"values"`
}

func Write(w io.Writer, result Result, format string, raw bool) error {
	enc := json.NewEncoder(w)
	if format == "json" {
		enc.SetIndent("", "  ")
		if raw {
			return enc.Encode(result)
		}
		return enc.Encode(compactRows(result))
	}
	if format == "jsonl" {
		for _, row := range compactRows(result) {
			if err := enc.Encode(row); err != nil {
				return fmt.Errorf("write JSONL: %w", err)
			}
		}
		return nil
	}
	cw := csv.NewWriter(w)
	header := []string{"REPORTER", "PARTNER", "PRODUCT", "AGGREGATE", "CURRENCY", "VALUE_SCALE"}
	for year := result.Query.PeriodFrom; year <= result.Query.PeriodTo; year++ {
		header = append(header, strconv.Itoa(year))
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, r := range result.Records {
		row := []string{displayCode(r.Reporter), displayCode(r.Partner), displayCode(r.Product), strconv.FormatBool(r.Aggregate), result.Query.Currency, result.Query.ValueScale}
		values := map[int]string{}
		for _, d := range r.Data {
			if d.Value != nil {
				values[d.Period] = strconv.FormatInt(*d.Value, 10)
			}
		}
		for year := result.Query.PeriodFrom; year <= result.Query.PeriodTo; year++ {
			row = append(row, values[year])
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("write CSV: %w", err)
	}
	return nil
}

func compactRows(result Result) []compactRecord {
	rows := make([]compactRecord, 0, len(result.Records))
	for _, r := range result.Records {
		row := compactRecord{Reporter: displayCode(r.Reporter), Partner: displayCode(r.Partner), Product: displayCode(r.Product), Aggregate: r.Aggregate, Currency: result.Query.Currency, ValueScale: result.Query.ValueScale, Values: map[string]int64{}}
		for _, d := range r.Data {
			if d.Value != nil {
				row.Values[strconv.Itoa(d.Period)] = *d.Value
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func displayCode(s Selector) string {
	if s.Kind == "group" && s.Label != "" {
		return normalized(s.Label)
	}
	if s.Code == "000" {
		return "WORLD"
	}
	if s.Code == "TOTAL" || s.Code == "ALL" {
		return "TOTAL"
	}
	for iso, numeric := range iso3ToNumeric {
		if numeric == s.Code && iso != "WORLD" {
			return iso
		}
	}
	return s.Code
}
