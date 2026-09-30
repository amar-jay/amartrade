package trademap

import "time"

const defaultBaseURL = "https://www.trademap.org/api"

type GoodsOptions struct {
	Flow, By, From, To, Product, Years, Format, DirectMirror string
	HSLevel                                                  int
	Raw                                                      bool
}
type Selector struct {
	Kind  string `json:"kind"`
	Code  string `json:"code"`
	Label string `json:"label"`
}
type Query struct {
	Flow         string   `json:"flow"`
	By           string   `json:"by"`
	Reporter     Selector `json:"reporter"`
	Partner      Selector `json:"partner"`
	Product      Selector `json:"product"`
	PeriodFrom   int      `json:"period_from"`
	PeriodTo     int      `json:"period_to"`
	DirectMirror string   `json:"direct_mirror"`
	Indicator    string   `json:"indicator"`
	Currency     string   `json:"currency"`
	ValueScale   string   `json:"value_scale"`
	HSLevel      int      `json:"hs_level,omitempty"`
}
type DataPoint struct {
	Period int     `json:"period"`
	Value  *int64  `json:"value"`
	Unit   *string `json:"unit"`
	Flag   *string `json:"flag"`
}
type JSONRecord struct {
	Reporter  Selector    `json:"reporter"`
	Partner   Selector    `json:"partner"`
	Product   Selector    `json:"product"`
	Aggregate bool        `json:"aggregate"`
	Data      []DataPoint `json:"data"`
}
type Result struct {
	Query     Query            `json:"query"`
	Records   []JSONRecord     `json:"records"`
	Sources   []map[string]any `json:"sources"`
	FetchedAt time.Time        `json:"fetched_at"`
}
type country struct {
	Code  string `json:"countryCd"`
	Label string `json:"label"`
}
type countryGroup struct {
	ID      int       `json:"id"`
	Label   string    `json:"label"`
	Members []country `json:"members"`
}
type product struct {
	Code      string `json:"productCd"`
	Label     string `json:"label"`
	Revisions string `json:"revisions"`
}
type productGroup struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
	Level int    `json:"level"`
}
type wireRecord struct {
	ReporterCode  string      `json:"reporterCd"`
	PartnerCode   string      `json:"partnerCd"`
	ProductCode   string      `json:"productCd"`
	ReporterLabel string      `json:"reporterLabel"`
	PartnerLabel  string      `json:"partnerLabel"`
	ProductLabel  string      `json:"productLabel"`
	Data          []DataPoint `json:"data"`
}
type page struct {
	NumPages         int              `json:"nbPages"`
	Records          []wireRecord     `json:"records"`
	AggregateRecords []wireRecord     `json:"aggregateRecords"`
	Sources          []map[string]any `json:"sources"`
}
