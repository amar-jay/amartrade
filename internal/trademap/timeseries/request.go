package timeseries

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/amar-jay/amartrade/internal/trademap/types"
)

const (
	defaultPageSize = 100
	valueScale      = int64(1000)
)

// Request describes one page of a goods time-series query.
type Request struct {
	Frequency  types.Frequency
	Dimension  types.Dimension
	Reporter   types.ReporterSelector
	Partner    types.PartnerSelector
	Goods      types.GoodsSelector
	Periods    types.PeriodRange
	Flow       types.TradeFlow
	DataMode   types.DataMode
	Unit       types.Unit
	Currency   string
	HSLevel    types.HSLevel
	Pagination Pagination
	Sort       Sort
}

type Pagination struct {
	Number int
	Size   int
}

type Sort struct {
	By        string
	Direction types.SortDirection
}

// Query is implemented by validated goods and service time-series requests.
type Query interface {
	prepare() (preparedQuery, error)
	withPage(int) Query
}

type preparedQuery struct {
	endpoint    string
	query       url.Values
	normalized  Query
	measurement Measurement
}

func (request Request) prepare() (preparedQuery, error) {
	endpoint, query, normalized, err := request.endpointAndQuery()
	if err != nil {
		return preparedQuery{}, err
	}
	return preparedQuery{endpoint: endpoint, query: query, normalized: normalized, measurement: measurementFor(normalized)}, nil
}

func (request Request) withPage(page int) Query {
	request.Pagination.Number = page
	return request
}

func (request Request) endpointAndQuery() (string, url.Values, Request, error) {
	request = request.withDefaults()
	frequency, err := request.Frequency.PathSegment()
	if err != nil {
		return "", nil, Request{}, err
	}
	if !goodsDimension(request.Dimension) {
		return "", nil, Request{}, fmt.Errorf("trademap: goods time series requires byProduct, byCountry, or byPartner dimension")
	}
	if err := validateFrequencyPeriods(request.Frequency, request.Periods); err != nil {
		return "", nil, Request{}, err
	}
	if request.Pagination.Number < 1 || request.Pagination.Size < 1 {
		return "", nil, Request{}, fmt.Errorf("trademap: page number and size must be positive")
	}
	if request.Dimension == types.ByProduct {
		if request.HSLevel.String() == "" {
			return "", nil, Request{}, fmt.Errorf("trademap: HS level is required for byProduct requests")
		}
	} else if request.HSLevel.String() != "" {
		return "", nil, Request{}, fmt.Errorf("trademap: HS level is only valid for byProduct requests")
	}
	if err := validateSort(request.Sort); err != nil {
		return "", nil, Request{}, err
	}
	if request.Unit == types.ValueUnit {
		if !validCurrency(request.Currency) {
			return "", nil, Request{}, fmt.Errorf("trademap: currency must contain exactly three uppercase letters")
		}
	} else if request.Currency != "" {
		return "", nil, Request{}, fmt.Errorf("trademap: currency is only valid with the VAL unit")
	}

	query := url.Values{}
	encoders := []interface{ EncodeQuery(url.Values) error }{
		request.Reporter, request.Partner, request.Goods, request.Periods,
		request.Flow, request.DataMode, request.Unit,
	}
	for _, encoder := range encoders {
		if err := encoder.EncodeQuery(query); err != nil {
			return "", nil, Request{}, err
		}
	}
	if request.Dimension == types.ByProduct {
		if err := request.HSLevel.EncodeQuery(query); err != nil {
			return "", nil, Request{}, err
		}
	}
	if request.Currency != "" {
		query.Set("currency", request.Currency)
	}
	query.Set("page", strconv.Itoa(request.Pagination.Number))
	query.Set("pageSize", strconv.Itoa(request.Pagination.Size))
	if request.Sort.By != "" {
		query.Set("sortBy", request.Sort.By)
		if err := request.Sort.Direction.EncodeQuery(query); err != nil {
			return "", nil, Request{}, err
		}
	}
	return "goods/timeSeries/" + frequency + "/" + request.Dimension.String(), query, request, nil
}

func (request Request) withDefaults() Request {
	if request.Pagination.Number == 0 {
		request.Pagination.Number = 1
	}
	if request.Pagination.Size == 0 {
		request.Pagination.Size = defaultPageSize
	}
	if request.Unit == types.ValueUnit && request.Currency == "" {
		request.Currency = "USD"
	}
	return request
}

func goodsDimension(dimension types.Dimension) bool {
	return dimension == types.ByProduct || dimension == types.ByCountry || dimension == types.ByPartner
}

func validateFrequencyPeriods(frequency types.Frequency, periods types.PeriodRange) error {
	want := types.YearGranularity
	switch frequency {
	case types.Yearly:
		want = types.YearGranularity
	case types.Quarterly:
		want = types.QuarterGranularity
	case types.Monthly:
		want = types.MonthGranularity
	default:
		return fmt.Errorf("trademap: frequency is not initialized")
	}
	if periods.Granularity() != want {
		return fmt.Errorf("trademap: %s frequency requires %s periods", frequency, want)
	}
	return nil
}

func validateSort(sortValue Sort) error {
	if strings.TrimSpace(sortValue.By) == "" {
		if sortValue.Direction.String() != "" {
			return fmt.Errorf("trademap: sort direction requires sort field")
		}
		return nil
	}
	if sortValue.Direction.String() == "" {
		return fmt.Errorf("trademap: sort field requires sort direction")
	}
	return nil
}

func validCurrency(currency string) bool {
	if len(currency) != 3 {
		return false
	}
	for _, character := range currency {
		if character < 'A' || character > 'Z' {
			return false
		}
	}
	return true
}
