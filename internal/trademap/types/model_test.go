package types

import (
	"net/url"
	"testing"
)

func TestCodeValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		valid bool
		build func() error
	}{
		{"world economy", true, codeError(NewEconomyCode("000"))},
		{"leading-zero economy", true, codeError(NewEconomyCode("004"))},
		{"empty economy", false, codeError(NewEconomyCode(""))},
		{"reserved economy character", false, codeError(NewEconomyCode("0&4"))},
		{"economy group", true, codeError(NewEconomyGroupCode("6757"))},
		{"padded economy group", false, codeError(NewEconomyGroupCode("042"))},
		{"empty economy group", false, codeError(NewEconomyGroupCode(""))},
		{"HS all", true, codeError(NewHSProductCode("ALL"))},
		{"leading-zero HS", true, codeError(NewHSProductCode("0101"))},
		{"ten-digit HS", true, codeError(NewHSProductCode("0101210000"))},
		{"empty HS", false, codeError(NewHSProductCode(""))},
		{"reserved HS character", false, codeError(NewHSProductCode("01/01"))},
		{"product group", true, codeError(NewProductGroupCode("18"))},
		{"padded product group", false, codeError(NewProductGroupCode("018"))},
		{"all services", true, codeError(NewServiceCode("ALL"))},
		{"returned service total", true, codeError(NewServiceCode("S00"))},
		{"EBOPS service", true, codeError(NewServiceCode("S03001001"))},
		{"empty service", false, codeError(NewServiceCode(""))},
		{"reserved service character", false, codeError(NewServiceCode("S03&01"))},
		{"indicator", true, codeError(NewIndicatorCode("GV5P"))},
		{"empty indicator", false, codeError(NewIndicatorCode(""))},
		{"reserved indicator character", false, codeError(NewIndicatorCode("VAL,BAL"))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := test.build()
			if test.valid && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestSelectorQuerySerialization(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		build func() (queryEncoder, error)
		want  string
	}{
		{"reporter world", func() (queryEncoder, error) { return ReporterEconomy(WorldEconomy) }, "country=000"},
		{"reporter group", func() (queryEncoder, error) { return ReporterEconomyGroup("42") }, "countryGrp=42"},
		{"partner leading zero", func() (queryEncoder, error) { return PartnerEconomy("004") }, "partner=004"},
		{"partner group", func() (queryEncoder, error) { return PartnerEconomyGroup("6757") }, "partnerGrp=6757"},
		{"all goods", func() (queryEncoder, error) { return GoodsProduct(AllGoods) }, "product=ALL"},
		{"leading-zero HS", func() (queryEncoder, error) { return GoodsProduct("0101") }, "product=0101"},
		{"product group", func() (queryEncoder, error) { return GoodsProductGroup("18") }, "productGrp=18"},
		{"all services", func() (queryEncoder, error) { return Service(AllServices) }, "service=ALL"},
		{"aggregate services", func() (queryEncoder, error) { return AggregateServices(), nil }, "service=ALL"},
		{"EBOPS service", func() (queryEncoder, error) { return Service("S03001001") }, "service=S03001001"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			selector, err := test.build()
			if err != nil {
				t.Fatal(err)
			}
			query := url.Values{}
			if err := selector.EncodeQuery(query); err != nil {
				t.Fatal(err)
			}
			if got := query.Encode(); got != test.want {
				t.Fatalf("query = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSelectorsCannotSerializeInvalidValues(t *testing.T) {
	t.Parallel()
	invalid := []queryEncoder{ReporterSelector{}, PartnerSelector{}, GoodsSelector{}, ServiceSelector{}}
	for _, selector := range invalid {
		if err := selector.EncodeQuery(url.Values{}); err == nil {
			t.Fatalf("%T zero value serialized without error", selector)
		}
	}
	if _, err := Service(TotalServices); err == nil {
		t.Fatal("S00 must not be accepted as a request selector")
	}
	if selector, err := ReporterEconomy(WorldEconomy); err != nil {
		t.Fatal(err)
	} else if err := selector.EncodeQuery(nil); err == nil {
		t.Fatal("nil query serialized without error")
	}
}

func TestSelectorRemovesConflictingKey(t *testing.T) {
	t.Parallel()
	selector, err := ReporterEconomyGroup("42")
	if err != nil {
		t.Fatal(err)
	}
	query := url.Values{"country": {"000"}}
	if err := selector.EncodeQuery(query); err != nil {
		t.Fatal(err)
	}
	if got := query.Encode(); got != "countryGrp=42" {
		t.Fatalf("query = %q", got)
	}
}

func TestEnumSerialization(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value queryEncoder
		want  string
	}{
		{"imports", Imports, "tradeFlow=I"},
		{"exports", Exports, "tradeFlow=E"},
		{"re-exports", ReExports, "tradeFlow=RE"},
		{"trade balance", TradeBalance, "tradeFlow=TB"},
		{"by product", ByProduct, "output=byProduct"},
		{"value unit", ValueUnit, "indicator=VAL"},
		{"quantity unit", QuantityUnit, "indicator=QTY"},
		{"growth-value unit", GrowthValueUnit, "indicator=GV"},
		{"CSV output", CSVOutput, "export=csv"},
		{"Excel output", ExcelOutput, "export=excel"},
		{"year granularity", YearGranularity, "dataType=Y"},
		{"quarter granularity", QuarterGranularity, "dataType=Q"},
		{"month granularity", MonthGranularity, "dataType=M"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := url.Values{}
			if err := test.value.EncodeQuery(query); err != nil {
				t.Fatal(err)
			}
			if got := query.Encode(); got != test.want {
				t.Fatalf("query = %q, want %q", got, test.want)
			}
		})
	}
	query := url.Values{"export": {"csv"}}
	if err := JSONOutput.EncodeQuery(query); err != nil {
		t.Fatal(err)
	}
	if got := query.Encode(); got != "" {
		t.Fatalf("JSON query = %q, want empty", got)
	}
	if _, err := (Frequency{}).PathSegment(); err == nil {
		t.Fatal("zero frequency returned a path segment")
	}
	if got, err := Monthly.PathSegment(); err != nil || got != "monthly" {
		t.Fatalf("monthly path = %q, %v", got, err)
	}
	invalid := []queryEncoder{TradeFlow{}, Dimension{}, Unit{}, OutputMode{}, PeriodGranularity{}}
	for _, value := range invalid {
		if err := value.EncodeQuery(url.Values{}); err == nil {
			t.Fatalf("%T zero value serialized without error", value)
		}
	}
}

func TestPeriods(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		granularity PeriodGranularity
		encoded     string
		valid       bool
	}{
		{"year", YearGranularity, "2025", true},
		{"quarter", QuarterGranularity, "202501", true},
		{"fourth quarter", QuarterGranularity, "202504", true},
		{"month", MonthGranularity, "202509", true},
		{"bad year width", YearGranularity, "025", false},
		{"zero year", YearGranularity, "0000", false},
		{"zero quarter", QuarterGranularity, "202500", false},
		{"fifth quarter", QuarterGranularity, "202505", false},
		{"thirteenth month", MonthGranularity, "202513", false},
		{"reserved period character", MonthGranularity, "2025&1", false},
		{"empty period", YearGranularity, "", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			period, err := ParsePeriod(test.granularity, test.encoded)
			if !test.valid {
				if err == nil {
					t.Fatalf("ParsePeriod(%q) returned no error", test.encoded)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := period.String(); got != test.encoded {
				t.Fatalf("period = %q, want %q", got, test.encoded)
			}
		})
	}
}

func TestPeriodRangeSerialization(t *testing.T) {
	t.Parallel()
	from, _ := NewMonth(2025, 1)
	to, _ := NewMonth(2025, 9)
	periods, err := NewPeriodRange(from, to)
	if err != nil {
		t.Fatal(err)
	}
	query := url.Values{}
	if err := periods.EncodeQuery(query); err != nil {
		t.Fatal(err)
	}
	if got := query.Encode(); got != "periodFrom=202501&periodTo=202509" {
		t.Fatalf("query = %q", got)
	}
	year, _ := NewYear(2025)
	if _, err := NewPeriodRange(from, year); err == nil {
		t.Fatal("mixed-granularity range returned no error")
	}
	if _, err := NewPeriodRange(to, from); err == nil {
		t.Fatal("descending range returned no error")
	}
}

type queryEncoder interface {
	EncodeQuery(url.Values) error
}

func codeError[T ~string](value T, err error) func() error {
	return func() error { return err }
}
