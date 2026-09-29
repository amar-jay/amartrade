package timeseries

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/amar-jay/amartrade/internal/trademap/types"
)

func TestAllGoodsRoutesFromFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		frequency types.Frequency
		dimension types.Dimension
		fixture   string
	}{
		{"yearly by product", types.Yearly, types.ByProduct, "yearly_by_product.json"},
		{"yearly by country group", types.Yearly, types.ByCountry, "yearly_by_country.json"},
		{"yearly by partner", types.Yearly, types.ByPartner, "yearly_by_partner.json"},
		{"quarterly by product", types.Quarterly, types.ByProduct, "quarterly_by_product.json"},
		{"quarterly by country", types.Quarterly, types.ByCountry, "quarterly_by_country.json"},
		{"quarterly by partner", types.Quarterly, types.ByPartner, "quarterly_by_partner.json"},
		{"monthly by product", types.Monthly, types.ByProduct, "monthly_by_product.json"},
		{"monthly by country", types.Monthly, types.ByCountry, "monthly_by_country.json"},
		{"monthly by partner", types.Monthly, types.ByPartner, "monthly_by_partner.json"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := fixtureRequest(t, test.frequency, test.dimension)
			getter := &fixtureGetter{t: t, fixture: test.fixture, wantEndpoint: "goods/timeSeries/" + test.frequency.String() + "/" + test.dimension.String()}
			page, err := NewService(getter).Page(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			if page.Number != 1 || page.TotalPages < 1 || len(page.Records) == 0 {
				t.Fatalf("unexpected page: %#v", page)
			}
			value := page.Records[0].Data[0]
			if value.Measure != types.ValueUnit || value.Currency != "USD" || value.Multiplier == nil || *value.Multiplier != 1000 {
				t.Fatalf("measurement metadata not attached: %#v", value)
			}
			if test.fixture == "yearly_by_product.json" {
				if len(page.Sources) != 1 {
					t.Fatalf("sources missing: %#v", page)
				}
			}
			if test.fixture == "yearly_by_country.json" {
				if len(page.AggregateRecords) != 1 || page.AggregateRecords[0].ReporterCode != "42" || page.Records[0].ReporterCode == "42" {
					t.Fatalf("group result not preserved: %#v", page)
				}
			}
		})
	}
}

func TestUnknownResponseFieldsArePreserved(t *testing.T) {
	t.Parallel()
	var page Page
	body := `{"nbRecords":1,"page":1,"nbPages":1,"diagnostic":"kept","records":[{"reporterCd":"792","recordHint":true,"data":[{"period":2025,"value":1,"valueHint":"kept"}]}]}`
	if err := json.Unmarshal([]byte(body), &page); err != nil {
		t.Fatal(err)
	}
	if page.Unknown["diagnostic"] == nil || page.Records[0].Unknown["recordHint"] == nil || page.Records[0].Data[0].Unknown["valueHint"] == nil {
		t.Fatalf("unknown fields were discarded: %#v", page)
	}
}

func TestAllPagesAndIteratorLimits(t *testing.T) {
	t.Parallel()
	request := fixtureRequest(t, types.Yearly, types.ByCountry)
	getter := &pagedGetter{}
	service := NewService(getter)
	pages, err := service.AllPages(context.Background(), request, Limits{MaxPages: 3, MaxRecords: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 3 || pages[2].Number != 3 || getter.calls != 3 {
		t.Fatalf("pages = %d, calls = %d", len(pages), getter.calls)
	}

	getter = &pagedGetter{}
	pages, err = NewService(getter).AllPages(context.Background(), request, Limits{MaxPages: 2, MaxRecords: 10})
	var limitError *LimitError
	if !errors.As(err, &limitError) || limitError.Limit != "page" || len(pages) != 2 {
		t.Fatalf("pages = %d, error = %v", len(pages), err)
	}

	getter = &pagedGetter{}
	iterator, err := NewService(getter).NewIterator(request, Limits{MaxPages: 10, MaxRecords: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !iterator.Next(context.Background()) || iterator.Page().Number != 1 {
		t.Fatalf("first page failed: %v", iterator.Err())
	}
	if iterator.Next(context.Background()) || !errors.As(iterator.Err(), &limitError) || limitError.Limit != "record" {
		t.Fatalf("record limit error = %v", iterator.Err())
	}
}

func TestRequestValidation(t *testing.T) {
	t.Parallel()
	valid := fixtureRequest(t, types.Yearly, types.ByCountry)
	tests := []struct {
		name   string
		mutate func(*Request)
	}{
		{"frequency period mismatch", func(request *Request) { request.Frequency = types.Monthly }},
		{"service dimension", func(request *Request) { request.Dimension = types.ByService }},
		{"HS level outside by-product", func(request *Request) { request.HSLevel = types.HS6 }},
		{"invalid page", func(request *Request) { request.Pagination.Number = -1 }},
		{"sort without direction", func(request *Request) { request.Sort = Sort{By: "2025"} }},
		{"currency outside value", func(request *Request) { request.Unit, request.Currency = types.QuantityUnit, "USD" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := valid
			test.mutate(&request)
			if _, _, _, err := request.endpointAndQuery(); err == nil {
				t.Fatal("invalid request returned no error")
			}
		})
	}
	byProduct := fixtureRequest(t, types.Yearly, types.ByProduct)
	byProduct.HSLevel = types.HSLevel{}
	if _, _, _, err := byProduct.endpointAndQuery(); err == nil {
		t.Fatal("by-product request without HS level returned no error")
	}
}

func TestSortSerialization(t *testing.T) {
	t.Parallel()
	request := fixtureRequest(t, types.Yearly, types.ByCountry)
	request.Sort = Sort{By: "2025", Direction: types.SortDescending}
	_, query, _, err := request.endpointAndQuery()
	if err != nil {
		t.Fatal(err)
	}
	if query.Get("sortBy") != "2025" || query.Get("sortDir") != "desc" {
		t.Fatalf("sort query = %v", query)
	}
}

type fixtureGetter struct {
	t            *testing.T
	fixture      string
	wantEndpoint string
}

func (getter *fixtureGetter) GetJSON(_ context.Context, endpoint string, query url.Values, result any) error {
	getter.t.Helper()
	if endpoint != getter.wantEndpoint {
		getter.t.Fatalf("endpoint = %q, want %q", endpoint, getter.wantEndpoint)
	}
	for key, want := range map[string]string{
		"partner": "000", "product": "ALL", "tradeFlow": "E", "directMirror": "D",
		"indicator": "VAL", "currency": "USD", "page": "1", "pageSize": "2",
	} {
		if got := query.Get(key); got != want {
			getter.t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if query.Get("periodFrom") == "" || query.Get("periodTo") == "" {
		getter.t.Errorf("missing period query: %v", query)
	}
	if query.Get("country") == "" && query.Get("countryGrp") == "" {
		getter.t.Errorf("missing reporter query: %v", query)
	}
	if stringsHasSuffix(endpoint, "/byProduct") && query.Get("hsLevel") != "6" {
		getter.t.Errorf("hsLevel = %q", query.Get("hsLevel"))
	}
	data, err := os.ReadFile(filepath.Join("testdata", getter.fixture))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, result)
}

type pagedGetter struct{ calls int }

func (getter *pagedGetter) GetJSON(_ context.Context, _ string, query url.Values, result any) error {
	getter.calls++
	page, _ := strconv.Atoi(query.Get("page"))
	body := fmt.Sprintf(`{"nbRecords":3,"page":%d,"nbRecordPerPage":1,"nbPages":3,"records":[{"reporterCd":"%03d","partnerCd":"000","productCd":"ALL","data":[{"period":2025,"value":1}]}],"aggregateRecords":[],"sources":[]}`, page, page)
	return json.Unmarshal([]byte(body), result)
}

func fixtureRequest(t *testing.T, frequency types.Frequency, dimension types.Dimension) Request {
	t.Helper()
	reporter, err := types.ReporterEconomy("792")
	if frequency == types.Yearly && dimension == types.ByCountry {
		reporter, err = types.ReporterEconomyGroup("42")
	} else if dimension == types.ByCountry {
		reporter, err = types.ReporterEconomy(types.WorldEconomy)
	}
	if err != nil {
		t.Fatal(err)
	}
	partner, err := types.PartnerEconomy(types.WorldEconomy)
	if err != nil {
		t.Fatal(err)
	}
	goods, err := types.GoodsProduct(types.AllGoods)
	if err != nil {
		t.Fatal(err)
	}
	periods := periodsFor(t, frequency)
	request := Request{
		Frequency: frequency, Dimension: dimension, Reporter: reporter, Partner: partner, Goods: goods,
		Periods: periods, Flow: types.Exports, DataMode: types.DirectData, Unit: types.ValueUnit,
		Pagination: Pagination{Number: 1, Size: 2},
	}
	if dimension == types.ByProduct {
		request.HSLevel = types.HS6
	}
	return request
}

func periodsFor(t *testing.T, frequency types.Frequency) types.PeriodRange {
	t.Helper()
	var from, to types.Period
	var err error
	switch frequency {
	case types.Yearly:
		from, err = types.NewYear(2024)
		to, _ = types.NewYear(2025)
	case types.Quarterly:
		from, err = types.NewQuarter(2024, 1)
		to, _ = types.NewQuarter(2025, 1)
	case types.Monthly:
		from, err = types.NewMonth(2024, 1)
		to, _ = types.NewMonth(2025, 1)
	}
	if err != nil {
		t.Fatal(err)
	}
	periods, err := types.NewPeriodRange(from, to)
	if err != nil {
		t.Fatal(err)
	}
	return periods
}

func stringsHasSuffix(value, suffix string) bool {
	return len(value) >= len(suffix) && value[len(value)-len(suffix):] == suffix
}
