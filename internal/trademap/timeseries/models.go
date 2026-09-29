package timeseries

import (
	"encoding/json"

	"github.com/amar-jay/amartrade/internal/trademap/types"
)

// Measurement describes how every value on a page should be interpreted.
type Measurement struct {
	Unit       types.Unit
	Currency   string
	Multiplier *int64
}

type Page struct {
	TotalRecords     int
	Number           int
	RecordsPerPage   int
	TotalPages       int
	ReferenceYear    *int
	Records          []Record
	AggregateRecords []Record
	Sources          []Source
	Measurement      Measurement
	Unknown          map[string]json.RawMessage
}

type Record struct {
	ReporterCode            string
	PartnerCode             string
	ProductCode             string
	ReporterLabel           *string
	PartnerLabel            *string
	ProductLabel            *string
	CanNavigateProductBelow bool
	CanNavigateCountry      bool
	Data                    []Value
	Unknown                 map[string]json.RawMessage
}

// Value preserves provider unit and flag strings and carries request-level
// scaling metadata beside the numeric value.
type Value struct {
	Period     int
	Amount     json.Number
	Unit       *string
	Flag       *string
	Measure    types.Unit
	Currency   string
	Multiplier *int64
	Unknown    map[string]json.RawMessage
}

// Source preserves the complete source value because the provider emits
// multiple undocumented source shapes.
type Source struct{ Raw json.RawMessage }

func (source *Source) UnmarshalJSON(data []byte) error {
	source.Raw = append(source.Raw[:0], data...)
	return nil
}

func (page *Page) UnmarshalJSON(data []byte) error {
	type wirePage struct {
		TotalRecords     int      `json:"nbRecords"`
		Number           int      `json:"page"`
		RecordsPerPage   int      `json:"nbRecordPerPage"`
		TotalPages       int      `json:"nbPages"`
		ReferenceYear    *int     `json:"refYear"`
		Records          []Record `json:"records"`
		AggregateRecords []Record `json:"aggregateRecords"`
		Sources          []Source `json:"sources"`
	}
	var wire wirePage
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	page.TotalRecords = wire.TotalRecords
	page.Number = wire.Number
	page.RecordsPerPage = wire.RecordsPerPage
	page.TotalPages = wire.TotalPages
	page.ReferenceYear = wire.ReferenceYear
	page.Records = wire.Records
	page.AggregateRecords = wire.AggregateRecords
	page.Sources = wire.Sources
	page.Unknown = unknownFields(data, "nbRecords", "page", "nbRecordPerPage", "nbPages", "refYear", "records", "aggregateRecords", "sources")
	return nil
}

func (record *Record) UnmarshalJSON(data []byte) error {
	type wireRecord struct {
		ReporterCode            string  `json:"reporterCd"`
		PartnerCode             string  `json:"partnerCd"`
		ProductCode             string  `json:"productCd"`
		ReporterLabel           *string `json:"reporterLabel"`
		PartnerLabel            *string `json:"partnerLabel"`
		ProductLabel            *string `json:"productLabel"`
		CanNavigateProductBelow bool    `json:"canNavigateProductBelow"`
		CanNavigateCountry      bool    `json:"canNavigateCountry"`
		Data                    []Value `json:"data"`
	}
	var wire wireRecord
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	record.ReporterCode = wire.ReporterCode
	record.PartnerCode = wire.PartnerCode
	record.ProductCode = wire.ProductCode
	record.ReporterLabel = wire.ReporterLabel
	record.PartnerLabel = wire.PartnerLabel
	record.ProductLabel = wire.ProductLabel
	record.CanNavigateProductBelow = wire.CanNavigateProductBelow
	record.CanNavigateCountry = wire.CanNavigateCountry
	record.Data = wire.Data
	record.Unknown = unknownFields(data, "reporterCd", "partnerCd", "productCd", "reporterLabel", "partnerLabel", "productLabel", "canNavigateProductBelow", "canNavigateCountry", "data")
	return nil
}

func (value *Value) UnmarshalJSON(data []byte) error {
	type wireValue struct {
		Period int         `json:"period"`
		Amount json.Number `json:"value"`
		Unit   *string     `json:"unit"`
		Flag   *string     `json:"flag"`
	}
	var wire wireValue
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	value.Period, value.Amount, value.Unit, value.Flag = wire.Period, wire.Amount, wire.Unit, wire.Flag
	value.Unknown = unknownFields(data, "period", "value", "unit", "flag")
	return nil
}

func unknownFields(data []byte, known ...string) map[string]json.RawMessage {
	fields := map[string]json.RawMessage{}
	if json.Unmarshal(data, &fields) != nil {
		return nil
	}
	for _, field := range known {
		delete(fields, field)
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}
