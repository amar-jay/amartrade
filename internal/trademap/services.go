package trademap

import (
	"github.com/amar-jay/amartrade/internal/trademap/reference"
	"github.com/amar-jay/amartrade/internal/trademap/timeseries"
)

// ReferenceData returns the reference catalog API backed by this client.
func (client *Client) ReferenceData() *reference.Service {
	return reference.NewService(client)
}

// GoodsTimeSeries returns the goods time-series API backed by this client.
func (client *Client) GoodsTimeSeries() *timeseries.Service {
	return timeseries.NewService(client)
}

// ServiceTimeSeries returns the services time-series API backed by this client.
func (client *Client) ServiceTimeSeries() *timeseries.Service {
	return timeseries.NewService(client)
}
