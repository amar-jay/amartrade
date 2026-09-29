package types

import (
	"fmt"
	"net/url"
	"strconv"
)

// Period is a validated Trade Map year, quarter, or month. Its zero value is
// invalid and cannot be serialized.
type Period struct {
	granularity PeriodGranularity
	year        int
	part        int
}

// PeriodRange represents an inclusive API period range.
type PeriodRange struct {
	from Period
	to   Period
}

func NewYear(year int) (Period, error) {
	if err := validateYear(year); err != nil {
		return Period{}, err
	}
	return Period{granularity: YearGranularity, year: year}, nil
}

func NewQuarter(year, quarter int) (Period, error) {
	if err := validateYear(year); err != nil {
		return Period{}, err
	}
	if quarter < 1 || quarter > 4 {
		return Period{}, fmt.Errorf("trademap: quarter must be between 1 and 4: %d", quarter)
	}
	return Period{granularity: QuarterGranularity, year: year, part: quarter}, nil
}

func NewMonth(year, month int) (Period, error) {
	if err := validateYear(year); err != nil {
		return Period{}, err
	}
	if month < 1 || month > 12 {
		return Period{}, fmt.Errorf("trademap: month must be between 1 and 12: %d", month)
	}
	return Period{granularity: MonthGranularity, year: year, part: month}, nil
}

// ParsePeriod parses the exact API encoding for the supplied granularity.
func ParsePeriod(granularity PeriodGranularity, value string) (Period, error) {
	if !granularity.valid() {
		return Period{}, fmt.Errorf("trademap: period granularity is not initialized")
	}
	wantLength := 4
	if granularity != YearGranularity {
		wantLength = 6
	}
	if len(value) != wantLength || !isDigits(value) {
		return Period{}, fmt.Errorf("trademap: invalid %s period %q", granularity, value)
	}
	year, _ := strconv.Atoi(value[:4])
	if granularity == YearGranularity {
		return NewYear(year)
	}
	part, _ := strconv.Atoi(value[4:])
	if granularity == QuarterGranularity {
		return NewQuarter(year, part)
	}
	return NewMonth(year, part)
}

func NewPeriodRange(from, to Period) (PeriodRange, error) {
	if !from.valid() || !to.valid() {
		return PeriodRange{}, fmt.Errorf("trademap: period range contains an invalid period")
	}
	if from.granularity != to.granularity {
		return PeriodRange{}, fmt.Errorf("trademap: period range granularities must match")
	}
	if from.String() > to.String() {
		return PeriodRange{}, fmt.Errorf("trademap: period range starts after it ends")
	}
	return PeriodRange{from: from, to: to}, nil
}

// EncodeQuery adds the inclusive periodFrom and periodTo values to query.
func (periods PeriodRange) EncodeQuery(query url.Values) error {
	if query == nil {
		return fmt.Errorf("trademap: query values must not be nil")
	}
	validated, err := NewPeriodRange(periods.from, periods.to)
	if err != nil {
		return err
	}
	query.Set("periodFrom", validated.from.String())
	query.Set("periodTo", validated.to.String())
	return nil
}

func (period Period) Granularity() PeriodGranularity { return period.granularity }

func (period Period) String() string {
	if !period.valid() {
		return ""
	}
	if period.granularity == YearGranularity {
		return fmt.Sprintf("%04d", period.year)
	}
	return fmt.Sprintf("%04d%02d", period.year, period.part)
}

func (period Period) valid() bool {
	if validateYear(period.year) != nil || !period.granularity.valid() {
		return false
	}
	switch period.granularity {
	case YearGranularity:
		return period.part == 0
	case QuarterGranularity:
		return period.part >= 1 && period.part <= 4
	case MonthGranularity:
		return period.part >= 1 && period.part <= 12
	default:
		return false
	}
}

func validateYear(year int) error {
	if year < 1 || year > 9999 {
		return fmt.Errorf("trademap: year must be between 1 and 9999: %d", year)
	}
	return nil
}
