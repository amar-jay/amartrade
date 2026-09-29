package types

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

// DataMode selects direct reporter data, mirror partner-reported data, or a
// provider-selected mixture of both.
type DataMode struct{ wire string }

var (
	DirectData = DataMode{"D"}
	MirrorData = DataMode{"M"}
	MixedData  = DataMode{"X"}
)

// HSLevel is the product depth returned by a by-product request.
type HSLevel struct{ wire string }

var (
	HS2  = HSLevel{"2"}
	HS4  = HSLevel{"4"}
	HS6  = HSLevel{"6"}
	HS10 = HSLevel{"10"}
)

// ServiceLevel is the EBOPS code length requested through bpmLevel.
type ServiceLevel struct{ wire string }

var (
	ServiceLevel3  = ServiceLevel{"3"}
	ServiceLevel6  = ServiceLevel{"6"}
	ServiceLevel9  = ServiceLevel{"9"}
	ServiceLevel12 = ServiceLevel{"12"}
	ServiceLevel15 = ServiceLevel{"15"}
)

type SortDirection struct{ wire string }

var (
	SortAscending  = SortDirection{"asc"}
	SortDescending = SortDirection{"desc"}
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
func (value DataMode) String() string          { return value.wire }
func (value HSLevel) String() string           { return value.wire }
func (value SortDirection) String() string     { return value.wire }
func (value ServiceLevel) String() string      { return value.wire }

func (value TradeFlow) EncodeQuery(query url.Values) error {
	return encodeEnum(query, "tradeFlow", value.wire, value.valid())
}

func (value Dimension) EncodeQuery(query url.Values) error {
	return encodeEnum(query, "output", value.wire, value.valid())
}

func (value Unit) EncodeQuery(query url.Values) error {
	return encodeEnum(query, "indicator", value.wire, value.valid())
}

func (value DataMode) EncodeQuery(query url.Values) error {
	return encodeEnum(query, "directMirror", value.wire, value.valid())
}

func (value HSLevel) EncodeQuery(query url.Values) error {
	return encodeEnum(query, "hsLevel", value.wire, value.valid())
}

func (value ServiceLevel) EncodeQuery(query url.Values) error {
	return encodeEnum(query, "bpmLevel", value.wire, value.valid())
}

func (value SortDirection) EncodeQuery(query url.Values) error {
	return encodeEnum(query, "sortDir", value.wire, value.valid())
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
func (value DataMode) valid() bool {
	return value == DirectData || value == MirrorData || value == MixedData
}
func (value HSLevel) valid() bool {
	return value == HS2 || value == HS4 || value == HS6 || value == HS10
}
func (value SortDirection) valid() bool {
	return value == SortAscending || value == SortDescending
}
func (value ServiceLevel) valid() bool {
	return value == ServiceLevel3 || value == ServiceLevel6 || value == ServiceLevel9 || value == ServiceLevel12 || value == ServiceLevel15
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
