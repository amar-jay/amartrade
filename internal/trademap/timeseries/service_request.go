package timeseries

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/amar-jay/amartrade/internal/trademap/types"
)

// ServiceRequest describes one page of a services time-series query without
// exposing goods-only product, HS-level, indicator, or direct/mirror fields.
type ServiceRequest struct {
	Frequency    types.Frequency
	Dimension    types.Dimension
	Reporter     types.ReporterSelector
	Partner      types.PartnerSelector
	Service      types.ServiceSelector
	Periods      types.PeriodRange
	Flow         types.TradeFlow
	Currency     string
	ServiceLevel types.ServiceLevel
	Pagination   Pagination
	Sort         Sort
}

func (request ServiceRequest) prepare() (preparedQuery, error) {
	request = request.withDefaults()
	frequency, err := request.Frequency.PathSegment()
	if err != nil {
		return preparedQuery{}, err
	}
	if request.Frequency == types.Monthly {
		return preparedQuery{}, fmt.Errorf("trademap: services time series supports yearly and quarterly frequencies only")
	}
	if request.Dimension != types.ByService && request.Dimension != types.ByCountry && request.Dimension != types.ByPartner {
		return preparedQuery{}, fmt.Errorf("trademap: services time series requires byService, byCountry, or byPartner dimension")
	}
	if err := validateFrequencyPeriods(request.Frequency, request.Periods); err != nil {
		return preparedQuery{}, err
	}
	if request.Pagination.Number < 1 || request.Pagination.Size < 1 {
		return preparedQuery{}, fmt.Errorf("trademap: page number and size must be positive")
	}
	if !validCurrency(request.Currency) {
		return preparedQuery{}, fmt.Errorf("trademap: currency must contain exactly three uppercase letters")
	}
	if err := validateSort(request.Sort); err != nil {
		return preparedQuery{}, err
	}
	if request.Dimension == types.ByService {
		if request.ServiceLevel.String() == "" {
			return preparedQuery{}, fmt.Errorf("trademap: service level is required for byService requests")
		}
	} else if request.ServiceLevel.String() != "" {
		return preparedQuery{}, fmt.Errorf("trademap: service level is only valid for byService requests")
	}

	query := url.Values{}
	encoders := []interface{ EncodeQuery(url.Values) error }{
		request.Reporter, request.Partner, request.Service, request.Periods, request.Flow,
	}
	for _, encoder := range encoders {
		if err := encoder.EncodeQuery(query); err != nil {
			return preparedQuery{}, err
		}
	}
	if request.Dimension == types.ByService {
		if err := request.ServiceLevel.EncodeQuery(query); err != nil {
			return preparedQuery{}, err
		}
	}
	query.Set("currency", request.Currency)
	query.Set("page", strconv.Itoa(request.Pagination.Number))
	query.Set("pageSize", strconv.Itoa(request.Pagination.Size))
	if request.Sort.By != "" {
		query.Set("sortBy", request.Sort.By)
		if err := request.Sort.Direction.EncodeQuery(query); err != nil {
			return preparedQuery{}, err
		}
	}
	measurement := Measurement{Unit: types.ValueUnit, Currency: request.Currency}
	multiplier := valueScale
	measurement.Multiplier = &multiplier
	return preparedQuery{
		endpoint: "services/timeSeries/" + frequency + "/" + request.Dimension.String(),
		query:    query, normalized: request, measurement: measurement,
	}, nil
}

func (request ServiceRequest) withDefaults() ServiceRequest {
	if request.Pagination.Number == 0 {
		request.Pagination.Number = 1
	}
	if request.Pagination.Size == 0 {
		request.Pagination.Size = defaultPageSize
	}
	if request.Currency == "" {
		request.Currency = "USD"
	}
	return request
}

func (request ServiceRequest) withPage(page int) Query {
	request.Pagination.Number = page
	return request
}
