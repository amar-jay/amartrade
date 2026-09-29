package trademap

import (
	"fmt"
	"net/url"
)

// Closed enum types use an unexported wire value. Callers can only use the
// values declared here; their zero values are deliberately invalid.

type TradeFlow struct{ wire string }

var (
	Imports      = TradeFlow{"I"}
	Exports      = TradeFlow{"E"}
	ReExports    = TradeFlow{"RE"}
	TradeBalance = TradeFlow{"TB"}
)

type Frequency struct{ wire string }

var (
	Yearly    = Frequency{"yearly"}
	Quarterly = Frequency{"quarterly"}
	Monthly   = Frequency{"monthly"}
)

type Dimension struct{ wire string }

var (
	ByProduct = Dimension{"byProduct"}
	ByService = Dimension{"byService"}
	ByCountry = Dimension{"byCountry"}
	ByPartner = Dimension{"byPartner"}
)

// Unit is the requested time-series measure sent through the indicator query
// parameter. Returned data-point unit strings remain opaque API data.
type Unit struct{ wire string }

var (
	ValueUnit       = Unit{"VAL"}
	QuantityUnit    = Unit{"QTY"}
	GrowthValueUnit = Unit{"GV"}
)

type OutputMode struct{ wire string }

var (
	JSONOutput  = OutputMode{"json"}
	CSVOutput   = OutputMode{"csv"}
	ExcelOutput = OutputMode{"excel"}
)

type PeriodGranularity struct{ wire string }

var (
	YearGranularity    = PeriodGranularity{"Y"}
	QuarterGranularity = PeriodGranularity{"Q"}
	MonthGranularity   = PeriodGranularity{"M"}
)

func (value TradeFlow) String() string         { return value.wire }
func (value Frequency) String() string         { return value.wire }
func (value Dimension) String() string         { return value.wire }
func (value Unit) String() string              { return value.wire }
func (value OutputMode) String() string        { return value.wire }
func (value PeriodGranularity) String() string { return value.wire }

func (value TradeFlow) EncodeQuery(query url.Values) error {
	return encodeEnum(query, "tradeFlow", value.wire, value.valid())
}

func (value Dimension) EncodeQuery(query url.Values) error {
	return encodeEnum(query, "output", value.wire, value.valid())
}

func (value Unit) EncodeQuery(query url.Values) error {
	return encodeEnum(query, "indicator", value.wire, value.valid())
}

func (value OutputMode) EncodeQuery(query url.Values) error {
	if query == nil {
		return fmt.Errorf("trademap: query values must not be nil")
	}
	if !value.valid() {
		return fmt.Errorf("trademap: output mode is not initialized")
	}
	if value == JSONOutput {
		query.Del("export")
	} else {
		query.Set("export", value.wire)
	}
	return nil
}

func (value PeriodGranularity) EncodeQuery(query url.Values) error {
	return encodeEnum(query, "dataType", value.wire, value.valid())
}

// PathSegment returns the validated frequency segment used in endpoint paths.
func (value Frequency) PathSegment() (string, error) {
	if !value.valid() {
		return "", fmt.Errorf("trademap: frequency is not initialized")
	}
	return value.wire, nil
}

func (value TradeFlow) valid() bool {
	return value == Imports || value == Exports || value == ReExports || value == TradeBalance
}
func (value Frequency) valid() bool { return value == Yearly || value == Quarterly || value == Monthly }
func (value Dimension) valid() bool {
	return value == ByProduct || value == ByService || value == ByCountry || value == ByPartner
}
func (value Unit) valid() bool {
	return value == ValueUnit || value == QuantityUnit || value == GrowthValueUnit
}
func (value OutputMode) valid() bool {
	return value == JSONOutput || value == CSVOutput || value == ExcelOutput
}
func (value PeriodGranularity) valid() bool {
	return value == YearGranularity || value == QuarterGranularity || value == MonthGranularity
}

func encodeEnum(query url.Values, key, wire string, valid bool) error {
	if query == nil {
		return fmt.Errorf("trademap: query values must not be nil")
	}
	if !valid {
		return fmt.Errorf("trademap: %s is not initialized", key)
	}
	query.Set(key, wire)
	return nil
}
