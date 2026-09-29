package timeseries

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/amar-jay/amartrade/internal/trademap/types"
)

func TestAllServiceRoutesFromFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		frequency types.Frequency
		dimension types.Dimension
		fixture   string
	}{
		{"yearly by service", types.Yearly, types.ByService, "yearly_by_service.json"},
		{"yearly by country", types.Yearly, types.ByCountry, "yearly_by_country.json"},
		{"yearly by partner", types.Yearly, types.ByPartner, "yearly_by_partner.json"},
		{"quarterly by service", types.Quarterly, types.ByService, "quarterly_by_service.json"},
		{"quarterly by country", types.Quarterly, types.ByCountry, "quarterly_by_country.json"},
		{"quarterly by partner", types.Quarterly, types.ByPartner, "quarterly_by_partner.json"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := serviceFixtureRequest(t, test.frequency, test.dimension)
			getter := &serviceFixtureGetter{
				t: t, fixture: test.fixture,
				wantEndpoint: "services/timeSeries/" + test.frequency.String() + "/" + test.dimension.String(),
			}
			page, err := NewService(getter).Page(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			if page.Number != 1 || (len(page.Records) == 0 && len(page.AggregateRecords) == 0) {
				t.Fatalf("unexpected page: %#v", page)
			}
			if page.Measurement.Unit != types.ValueUnit || page.Measurement.Currency != "USD" || page.Measurement.Multiplier == nil || *page.Measurement.Multiplier != 1000 {
				t.Fatalf("service measurement metadata = %#v", page.Measurement)
			}
			if test.dimension == types.ByService {
				if len(page.AggregateRecords) != 1 || page.AggregateRecords[0].ProductCode != string(types.TotalServices) {
					t.Fatalf("aggregate service total not preserved: %#v", page.AggregateRecords)
				}
			}
		})
	}
}

func TestServiceAllRequestAndReturnedTotalAreDistinct(t *testing.T) {
	t.Parallel()
	request := serviceFixtureRequest(t, types.Yearly, types.ByService)
	prepared, err := request.prepare()
	if err != nil {
		t.Fatal(err)
	}
	if got := prepared.query.Get("service"); got != string(types.AllServices) {
		t.Fatalf("request service = %q, want ALL", got)
	}
	page, err := NewService(&serviceFixtureGetter{
		t: t, fixture: "yearly_by_service.json", wantEndpoint: "services/timeSeries/yearly/byService",
	}).Page(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if got := page.AggregateRecords[0].ProductCode; got != string(types.TotalServices) {
		t.Fatalf("returned aggregate code = %q, want S00", got)
	}
	if _, err := types.Service(types.TotalServices); err == nil {
		t.Fatal("response-only S00 was accepted as a request selector")
	}
}

func TestServiceRequestValidation(t *testing.T) {
	t.Parallel()
	valid := serviceFixtureRequest(t, types.Yearly, types.ByCountry)
	tests := []struct {
		name   string
		mutate func(*ServiceRequest)
	}{
		{"monthly frequency", func(request *ServiceRequest) { request.Frequency = types.Monthly }},
		{"goods dimension", func(request *ServiceRequest) { request.Dimension = types.ByProduct }},
		{"service level outside by-service", func(request *ServiceRequest) { request.ServiceLevel = types.ServiceLevel3 }},
		{"invalid currency", func(request *ServiceRequest) { request.Currency = "usd" }},
		{"invalid page", func(request *ServiceRequest) { request.Pagination.Number = -1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := valid
			test.mutate(&request)
			if _, err := request.prepare(); err == nil {
				t.Fatal("invalid service request returned no error")
			}
		})
	}
	byService := serviceFixtureRequest(t, types.Yearly, types.ByService)
	byService.ServiceLevel = types.ServiceLevel{}
	if _, err := byService.prepare(); err == nil {
		t.Fatal("by-service request without bpmLevel returned no error")
	}
}

func TestServiceRequestUsesSharedIterator(t *testing.T) {
	t.Parallel()
	request := serviceFixtureRequest(t, types.Yearly, types.ByPartner)
	getter := &singleServicePageGetter{}
	iterator, err := NewService(getter).NewIterator(request, Limits{MaxPages: 2, MaxRecords: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !iterator.Next(context.Background()) || iterator.Page().Records[0].ProductCode != "S00" {
		t.Fatalf("service iterator failed: %v", iterator.Err())
	}
	if iterator.Next(context.Background()) || iterator.Err() != nil {
		t.Fatalf("service iterator did not stop cleanly: %v", iterator.Err())
	}
}

type serviceFixtureGetter struct {
	t            *testing.T
	fixture      string
	wantEndpoint string
}

func (getter *serviceFixtureGetter) GetJSON(_ context.Context, endpoint string, query url.Values, result any) error {
	getter.t.Helper()
	if endpoint != getter.wantEndpoint {
		getter.t.Fatalf("endpoint = %q, want %q", endpoint, getter.wantEndpoint)
	}
	for key, want := range map[string]string{
		"service": "ALL", "tradeFlow": "E", "partner": "000", "currency": "USD", "page": "1", "pageSize": "2",
	} {
		if got := query.Get(key); got != want {
			getter.t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	for _, goodsOnly := range []string{"product", "productGrp", "hsLevel", "indicator", "directMirror"} {
		if query.Has(goodsOnly) {
			getter.t.Errorf("goods-only parameter %q leaked into service request: %v", goodsOnly, query)
		}
	}
	if stringsHasSuffix(endpoint, "/byService") && query.Get("bpmLevel") != "3" {
		getter.t.Errorf("bpmLevel = %q, want 3", query.Get("bpmLevel"))
	}
	data, err := os.ReadFile(filepath.Join("testdata", "services", getter.fixture))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, result)
}

type singleServicePageGetter struct{}

func (*singleServicePageGetter) GetJSON(_ context.Context, _ string, _ url.Values, result any) error {
	return json.Unmarshal([]byte(`{"nbRecords":1,"page":1,"nbRecordPerPage":1,"nbPages":1,"records":[{"reporterCd":"792","partnerCd":"000","productCd":"S00","data":[{"period":2023,"value":1}]}]}`), result)
}

func serviceFixtureRequest(t *testing.T, frequency types.Frequency, dimension types.Dimension) ServiceRequest {
	t.Helper()
	reporter, err := types.ReporterEconomy("792")
	if dimension == types.ByCountry {
		reporter, err = types.ReporterEconomy(types.WorldEconomy)
	}
	if err != nil {
		t.Fatal(err)
	}
	partner, err := types.PartnerEconomy(types.WorldEconomy)
	if err != nil {
		t.Fatal(err)
	}
	service := types.AggregateServices()
	request := ServiceRequest{
		Frequency: frequency, Dimension: dimension, Reporter: reporter, Partner: partner,
		Service: service, Periods: servicePeriodsFor(t, frequency), Flow: types.Exports,
		Pagination: Pagination{Number: 1, Size: 2},
	}
	if dimension == types.ByService {
		request.ServiceLevel = types.ServiceLevel3
	}
	return request
}

func servicePeriodsFor(t *testing.T, frequency types.Frequency) types.PeriodRange {
	t.Helper()
	var from, to types.Period
	var err error
	if frequency == types.Yearly {
		from, err = types.NewYear(2022)
		to, _ = types.NewYear(2023)
	} else {
		from, err = types.NewQuarter(2022, 1)
		to, _ = types.NewQuarter(2023, 1)
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
