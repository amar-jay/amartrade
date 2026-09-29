package timeseries

import (
	"context"
	"net/url"

	"github.com/amar-jay/amartrade/internal/trademap/types"
)

type JSONGetter interface {
	GetJSON(context.Context, string, url.Values, any) error
}

type Service struct{ client JSONGetter }

func NewService(client JSONGetter) *Service { return &Service{client: client} }

// Page validates a goods or services request and fetches exactly one page.
func (service *Service) Page(ctx context.Context, request Query) (*Page, error) {
	prepared, err := request.prepare()
	if err != nil {
		return nil, err
	}
	var page Page
	if err := service.client.GetJSON(ctx, prepared.endpoint, prepared.query, &page); err != nil {
		return nil, err
	}
	page.Measurement = prepared.measurement
	attachMeasurement(page.Records, prepared.measurement)
	attachMeasurement(page.AggregateRecords, prepared.measurement)
	return &page, nil
}

func measurementFor(request Request) Measurement {
	measurement := Measurement{Unit: request.Unit, Currency: request.Currency}
	if request.Unit == types.ValueUnit {
		multiplier := valueScale
		measurement.Multiplier = &multiplier
	}
	return measurement
}

func attachMeasurement(records []Record, measurement Measurement) {
	for recordIndex := range records {
		for valueIndex := range records[recordIndex].Data {
			value := &records[recordIndex].Data[valueIndex]
			value.Measure = measurement.Unit
			value.Currency = measurement.Currency
			value.Multiplier = measurement.Multiplier
		}
	}
}
