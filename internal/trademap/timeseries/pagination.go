package timeseries

import (
	"context"
	"fmt"
)

var DefaultLimits = Limits{MaxPages: 100, MaxRecords: 100_000}

// Limits bounds automatic pagination independently of provider metadata.
type Limits struct {
	MaxPages   int
	MaxRecords int
}

type LimitError struct {
	Limit string
	Max   int
}

func (err *LimitError) Error() string {
	return fmt.Sprintf("trademap: time-series pagination exceeded %s limit of %d", err.Limit, err.Max)
}

type Iterator struct {
	service *Service
	request Request
	limits  Limits
	current *Page
	pages   int
	records int
	done    bool
	err     error
}

func (service *Service) NewIterator(request Request, limits Limits) (*Iterator, error) {
	if err := validateLimits(limits); err != nil {
		return nil, err
	}
	_, _, normalized, err := request.endpointAndQuery()
	if err != nil {
		return nil, err
	}
	return &Iterator{service: service, request: normalized, limits: limits}, nil
}

// Next fetches the next page. It returns false on completion or error.
func (iterator *Iterator) Next(ctx context.Context) bool {
	if iterator.done || iterator.err != nil {
		return false
	}
	if iterator.pages >= iterator.limits.MaxPages {
		iterator.err = &LimitError{Limit: "page", Max: iterator.limits.MaxPages}
		return false
	}
	page, err := iterator.service.Page(ctx, iterator.request)
	if err != nil {
		iterator.err = err
		return false
	}
	pageRecords := len(page.Records) + len(page.AggregateRecords)
	if iterator.records+pageRecords > iterator.limits.MaxRecords {
		iterator.err = &LimitError{Limit: "record", Max: iterator.limits.MaxRecords}
		return false
	}
	iterator.current = page
	iterator.pages++
	iterator.records += pageRecords

	if pageRecords == 0 || (page.TotalPages > 0 && page.Number >= page.TotalPages) {
		iterator.done = true
	} else {
		iterator.request.Pagination.Number++
	}
	return true
}

func (iterator *Iterator) Page() *Page { return iterator.current }
func (iterator *Iterator) Err() error  { return iterator.err }

// AllPages fetches every available page within explicit safety limits. On a
// limit or transport error it returns pages already fetched together with err.
func (service *Service) AllPages(ctx context.Context, request Request, limits Limits) ([]*Page, error) {
	iterator, err := service.NewIterator(request, limits)
	if err != nil {
		return nil, err
	}
	pages := make([]*Page, 0)
	for iterator.Next(ctx) {
		pages = append(pages, iterator.Page())
	}
	return pages, iterator.Err()
}

func validateLimits(limits Limits) error {
	if limits.MaxPages < 1 || limits.MaxRecords < 1 {
		return fmt.Errorf("trademap: maximum pages and records must be positive")
	}
	return nil
}
