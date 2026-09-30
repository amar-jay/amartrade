package trademap

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestParseYears(t *testing.T) {
	tests := []struct {
		in       string
		from, to int
		fails    bool
	}{
		{"2024", 2024, 2024, false}, {"2020:2024", 2020, 2024, false}, {"2024:2020", 0, 0, true}, {"nope", 0, 0, true},
	}
	for _, tt := range tests {
		from, to, err := parseYears(tt.in)
		if (err != nil) != tt.fails || (!tt.fails && (from != tt.from || to != tt.to)) {
			t.Fatalf("parseYears(%q) = %d,%d,%v", tt.in, from, to, err)
		}
	}
}

func TestRawRequiresJSONFormat(t *testing.T) {
	_, err := NewClient(nil).Goods(context.Background(), GoodsOptions{Years: "2024", By: "country", Format: "jsonl", Raw: true})
	if err == nil || !strings.Contains(err.Error(), "--raw requires --format json") {
		t.Fatalf("expected raw format validation error, got %v", err)
	}
}

func TestResolveFriendlyAndNativeSelectors(t *testing.T) {
	countries := []country{{Code: "000", Label: "World"}, {Code: "276", Label: "Germany"}}
	groups := []countryGroup{{ID: 42, Label: "EU 27"}}
	for input, code := range map[string]string{"DEU": "276", "Germany": "276", "276": "276", "EU27": "42", "42": "42", "WORLD": "000"} {
		got, err := resolveEconomy(input, countries, groups)
		if err != nil || got.Code != code {
			t.Errorf("resolveEconomy(%q) = %#v, %v", input, got, err)
		}
	}
}

func TestProductGroupPrefixDisambiguatesHSCode(t *testing.T) {
	products := []product{{Code: "18", Label: "Cocoa"}}
	groups := []productGroup{{ID: 18, Label: "Vehicles"}}
	got, err := resolveProduct("group:18", products, groups)
	if err != nil || got.Kind != "group" || got.Label != "Vehicles" {
		t.Fatalf("resolveProduct(group:18) = %#v, %v", got, err)
	}
}

func TestGoodsPaginatesAndNormalizes(t *testing.T) {
	var dataQueries []url.Values
	bodies := map[string]string{
		"/countries":                `[{"countryCd":"000","label":"World"},{"countryCd":"276","label":"Germany"}]`,
		"/countries/groups/generic": `[{"id":42,"label":"EU 27","members":[]}]`,
		"/products/HS":              `[{"productCd":"ALL","label":"All products"}]`,
		"/products/groups/generic":  `[]`,
	}
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		path := strings.TrimPrefix(r.URL.Path, "/api")
		body := bodies[path]
		if path == "/goods/timeSeries/yearly/byCountry" {
			dataQueries = append(dataQueries, r.URL.Query())
			if r.URL.Query().Get("page") == "1" {
				body = `{"nbPages":2,"records":[{"reporterCd":"276","partnerCd":"000","productCd":"TOTAL","data":[{"period":2024,"value":12,"unit":null,"flag":"E"}]}],"aggregateRecords":[],"sources":[{"name":"ITC"}]}`
			} else {
				body = `{"nbPages":2,"records":[],"aggregateRecords":[{"reporterCd":"42","partnerCd":"000","productCd":"TOTAL","data":[{"period":2024,"value":20,"unit":null,"flag":null}]}]}`
			}
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	client := NewClient(&http.Client{Transport: transport})
	result, err := client.Goods(context.Background(), GoodsOptions{Flow: "E", By: "country", From: "EU27", To: "WORLD", Product: "ALL", Years: "2024", Format: "json", HSLevel: 2, DirectMirror: "direct"})
	if err != nil {
		t.Fatal(err)
	}
	if len(dataQueries) != 2 || dataQueries[0].Get("countryGrp") != "42" || dataQueries[0].Get("country") != "" {
		t.Fatalf("unexpected queries: %#v", dataQueries)
	}
	if len(result.Records) != 2 || result.Records[0].Data[0].Value == nil || *result.Records[0].Data[0].Value != 12000 {
		t.Fatalf("unexpected records: %#v", result.Records)
	}
	if !result.Records[1].Aggregate || result.Records[1].Reporter.Label != "EU 27" {
		t.Fatalf("aggregate not preserved: %#v", result.Records[1])
	}
	if len(result.Sources) != 1 || result.Records[0].Data[0].Flag == nil || *result.Records[0].Data[0].Flag != "E" {
		t.Fatalf("metadata not preserved: %#v", result)
	}
}

func TestWriteCSV(t *testing.T) {
	v := int64(123000)
	result := Result{Query: Query{PeriodFrom: 2024, PeriodTo: 2025, Currency: "USD", ValueScale: "units"}, Records: []JSONRecord{{Reporter: Selector{Code: "276"}, Partner: Selector{Code: "000"}, Product: Selector{Code: "TOTAL"}, Data: []DataPoint{{Period: 2024, Value: &v}}}}}
	var out bytes.Buffer
	if err := Write(&out, result, "csv", false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "REPORTER,PARTNER,PRODUCT,AGGREGATE,CURRENCY,VALUE_SCALE,2024,2025\nDEU,WORLD,TOTAL,false,USD,units,123000,") {
		t.Fatalf("unexpected CSV:\n%s", out.String())
	}
}

func TestDisplayCodeUsesGroupAlias(t *testing.T) {
	tests := []struct {
		selector Selector
		want     string
	}{
		{Selector{Kind: "group", Code: "24", Label: "ASEAN"}, "ASEAN"},
		{Selector{Kind: "group", Code: "42", Label: "EU 27"}, "EU27"},
		{Selector{Kind: "economy", Code: "276", Label: "Germany"}, "DEU"},
	}
	for _, tt := range tests {
		if got := displayCode(tt.selector); got != tt.want {
			t.Errorf("displayCode(%#v) = %q, want %q", tt.selector, got, tt.want)
		}
	}
}

func TestWriteCompactAndRawJSON(t *testing.T) {
	v := int64(123000)
	result := Result{Query: Query{PeriodFrom: 2024, PeriodTo: 2024, Currency: "USD", ValueScale: "units"}, Records: []JSONRecord{{Reporter: Selector{Code: "276", Label: "Germany"}, Partner: Selector{Code: "000", Label: "World"}, Product: Selector{Code: "TOTAL", Label: "All products"}, Data: []DataPoint{{Period: 2024, Value: &v}}}}}
	var compact bytes.Buffer
	if err := Write(&compact, result, "json", false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(compact.String(), `"reporter": "DEU"`) || !strings.Contains(compact.String(), `"2024": 123000`) || strings.Contains(compact.String(), `"query"`) {
		t.Fatalf("unexpected compact JSON:\n%s", compact.String())
	}
	var raw bytes.Buffer
	if err := Write(&raw, result, "json", true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw.String(), `"query"`) || !strings.Contains(raw.String(), `"label": "Germany"`) {
		t.Fatalf("unexpected raw JSON:\n%s", raw.String())
	}
}

func TestWriteJSONL(t *testing.T) {
	v1, v2 := int64(1000), int64(2000)
	result := Result{Query: Query{Currency: "USD", ValueScale: "units"}, Records: []JSONRecord{
		{Reporter: Selector{Code: "276"}, Partner: Selector{Code: "000"}, Product: Selector{Code: "TOTAL"}, Data: []DataPoint{{Period: 2024, Value: &v1}}},
		{Reporter: Selector{Code: "792"}, Partner: Selector{Code: "000"}, Product: Selector{Code: "TOTAL"}, Data: []DataPoint{{Period: 2024, Value: &v2}}},
	}}
	var out bytes.Buffer
	if err := Write(&out, result, "jsonl", false); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], `"reporter":"DEU"`) || !strings.Contains(lines[1], `"reporter":"TUR"`) {
		t.Fatalf("unexpected JSONL:\n%s", out.String())
	}
}

func TestRemoveDuplicateIndividualAggregate(t *testing.T) {
	v := int64(10)
	base := JSONRecord{Reporter: Selector{Code: "276"}, Partner: Selector{Code: "000"}, Product: Selector{Code: "TOTAL"}, Data: []DataPoint{{Period: 2024, Value: &v}}}
	aggregate := base
	aggregate.Aggregate = true
	groupAggregate := aggregate
	groupAggregate.Reporter.Code = "42"
	got := removeDuplicateAggregates([]JSONRecord{base, aggregate, groupAggregate})
	if len(got) != 2 || !got[1].Aggregate || got[1].Reporter.Code != "42" {
		t.Fatalf("unexpected filtered records: %#v", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
