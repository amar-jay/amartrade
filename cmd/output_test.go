package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/amar-jay/amartrade/internal/trademap/timeseries"
	"github.com/amar-jay/amartrade/internal/trademap/types"
)

func TestWriteTimeSeriesTableParsesCodesAndDropsDuplicates(t *testing.T) {
	germany := "Germany"
	page := &timeseries.Page{
		Number: 1,
		Records: []timeseries.Record{{
			ReporterCode: "276", PartnerCode: "000", ProductCode: "TOTAL",
			ReporterLabel: &germany,
			Data: []timeseries.Value{
				{Period: 2020, Amount: json.Number("1379900278"), Measure: types.ValueUnit, Currency: "USD", Multiplier: int64Ptr(1000)},
				{Period: 2021, Amount: json.Number("1631098969"), Measure: types.ValueUnit, Currency: "USD", Multiplier: int64Ptr(1000)},
			},
		}},
		AggregateRecords: []timeseries.Record{{
			ReporterCode: "276", PartnerCode: "000", ProductCode: "TOTAL",
			Data: []timeseries.Value{
				{Period: 2020, Amount: json.Number("1379900278"), Measure: types.ValueUnit, Currency: "USD", Multiplier: int64Ptr(1000)},
			},
		}},
	}
	var stdout strings.Builder
	if err := writeTimeSeries(&stdout, "table", []*timeseries.Page{page}, nil); err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	if strings.Count(got, "Germany") != 1 {
		t.Fatalf("expected one labeled row, got:\n%s", got)
	}
	if strings.Contains(got, "276") {
		t.Fatalf("table still shows raw reporter code:\n%s", got)
	}
	if !strings.Contains(got, "World") || !strings.Contains(got, "All products") {
		t.Fatalf("partner/product not parsed:\n%s", got)
	}
	if !strings.Contains(got, "Values in thousands USD") {
		t.Fatalf("missing unit caption:\n%s", got)
	}
	if !strings.Contains(got, "1,379,900,278") {
		t.Fatalf("amount not grouped:\n%s", got)
	}
}

func int64Ptr(value int64) *int64 { return &value }
