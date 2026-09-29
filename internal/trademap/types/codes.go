package types

import (
	"fmt"
	"strings"
)

// EconomyCode is a three-digit Trade Map economy code. Codes are strings so
// significant leading zeroes, including World (000), are preserved.
type EconomyCode string

// EconomyGroupCode is an opaque, unpadded Trade Map economy-group ID.
type EconomyGroupCode string

// HSProductCode is ALL or a 2, 4, 6, or 10-digit HS product code.
type HSProductCode string

// ProductGroupCode is an opaque, unpadded Trade Map product-group ID.
type ProductGroupCode string

// ServiceCode is ALL, S00, or an EBOPS service code.
type ServiceCode string

// IndicatorCode identifies a Trade Map indicator, such as VAL or GV5.
type IndicatorCode string

const (
	WorldEconomy EconomyCode   = "000"
	AllGoods     HSProductCode = "ALL"
	// AllServices is the request code for aggregate service queries.
	AllServices ServiceCode = "ALL"
	// TotalServices is the code returned by the API for aggregate services.
	TotalServices ServiceCode = "S00"
)

func NewEconomyCode(value string) (EconomyCode, error) {
	if !isDigits(value) || len(value) != 3 {
		return "", fmt.Errorf("trademap: economy code must contain exactly three digits: %q", value)
	}
	return EconomyCode(value), nil
}

func NewEconomyGroupCode(value string) (EconomyGroupCode, error) {
	if err := validateGroupCode("economy group", value); err != nil {
		return "", err
	}
	return EconomyGroupCode(value), nil
}

func NewHSProductCode(value string) (HSProductCode, error) {
	if value == string(AllGoods) {
		return AllGoods, nil
	}
	if !isDigits(value) || (len(value) != 2 && len(value) != 4 && len(value) != 6 && len(value) != 10) {
		return "", fmt.Errorf("trademap: HS product code must be ALL or contain 2, 4, 6, or 10 digits: %q", value)
	}
	return HSProductCode(value), nil
}

func NewProductGroupCode(value string) (ProductGroupCode, error) {
	if err := validateGroupCode("product group", value); err != nil {
		return "", err
	}
	return ProductGroupCode(value), nil
}

func NewServiceCode(value string) (ServiceCode, error) {
	if value == string(AllServices) || value == string(TotalServices) {
		return ServiceCode(value), nil
	}
	if !strings.HasPrefix(value, "S") || !isDigits(strings.TrimPrefix(value, "S")) {
		return "", fmt.Errorf("trademap: invalid EBOPS service code: %q", value)
	}
	digits := len(value) - 1
	if digits != 2 && digits != 5 && digits != 8 && digits != 11 && digits != 14 {
		return "", fmt.Errorf("trademap: EBOPS service code must contain S followed by 2, 5, 8, 11, or 14 digits: %q", value)
	}
	return ServiceCode(value), nil
}

func NewIndicatorCode(value string) (IndicatorCode, error) {
	if value == "" || len(value) > 16 {
		return "", fmt.Errorf("trademap: indicator code must contain 1 to 16 uppercase letters or digits: %q", value)
	}
	for _, character := range value {
		if (character < 'A' || character > 'Z') && (character < '0' || character > '9') {
			return "", fmt.Errorf("trademap: indicator code must contain only uppercase letters or digits: %q", value)
		}
	}
	return IndicatorCode(value), nil
}

func validateGroupCode(kind, value string) error {
	if !isDigits(value) || value == "0" || strings.HasPrefix(value, "0") {
		return fmt.Errorf("trademap: %s code must be an unpadded positive integer: %q", kind, value)
	}
	return nil
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
