package trademap

import (
	"errors"
	"net/url"
)

type selectorKind uint8

const (
	selectorInvalid selectorKind = iota
	selectorIndividual
	selectorGroup
)

// ReporterSelector selects exactly one reporter economy or economy group.
type ReporterSelector struct {
	kind selectorKind
	code string
}

// PartnerSelector selects exactly one partner economy or economy group.
type PartnerSelector struct {
	kind selectorKind
	code string
}

// GoodsSelector selects exactly one HS product or product group.
type GoodsSelector struct {
	kind selectorKind
	code string
}

// ServiceSelector selects one EBOPS service or aggregate services. S00 is a
// response-only total and is rejected when a selector is serialized.
type ServiceSelector struct{ code ServiceCode }

func ReporterEconomy(code EconomyCode) (ReporterSelector, error) {
	validated, err := NewEconomyCode(string(code))
	if err != nil {
		return ReporterSelector{}, err
	}
	return ReporterSelector{kind: selectorIndividual, code: string(validated)}, nil
}

func ReporterEconomyGroup(code EconomyGroupCode) (ReporterSelector, error) {
	validated, err := NewEconomyGroupCode(string(code))
	if err != nil {
		return ReporterSelector{}, err
	}
	return ReporterSelector{kind: selectorGroup, code: string(validated)}, nil
}

func PartnerEconomy(code EconomyCode) (PartnerSelector, error) {
	validated, err := NewEconomyCode(string(code))
	if err != nil {
		return PartnerSelector{}, err
	}
	return PartnerSelector{kind: selectorIndividual, code: string(validated)}, nil
}

func PartnerEconomyGroup(code EconomyGroupCode) (PartnerSelector, error) {
	validated, err := NewEconomyGroupCode(string(code))
	if err != nil {
		return PartnerSelector{}, err
	}
	return PartnerSelector{kind: selectorGroup, code: string(validated)}, nil
}

func GoodsProduct(code HSProductCode) (GoodsSelector, error) {
	validated, err := NewHSProductCode(string(code))
	if err != nil {
		return GoodsSelector{}, err
	}
	return GoodsSelector{kind: selectorIndividual, code: string(validated)}, nil
}

func GoodsProductGroup(code ProductGroupCode) (GoodsSelector, error) {
	validated, err := NewProductGroupCode(string(code))
	if err != nil {
		return GoodsSelector{}, err
	}
	return GoodsSelector{kind: selectorGroup, code: string(validated)}, nil
}

func Service(code ServiceCode) (ServiceSelector, error) {
	validated, err := NewServiceCode(string(code))
	if err != nil {
		return ServiceSelector{}, err
	}
	if validated == TotalServices {
		return ServiceSelector{}, errors.New("trademap: S00 is a returned total code and cannot be used in a service request")
	}
	return ServiceSelector{code: validated}, nil
}

func AggregateServices() ServiceSelector { return ServiceSelector{code: AllServices} }

// EncodeQuery adds the reporter selector to query.
func (selector ReporterSelector) EncodeQuery(query url.Values) error {
	return addExclusiveSelector(query, selector.kind, selector.code, "country", "countryGrp")
}

// EncodeQuery adds the partner selector to query.
func (selector PartnerSelector) EncodeQuery(query url.Values) error {
	return addExclusiveSelector(query, selector.kind, selector.code, "partner", "partnerGrp")
}

// EncodeQuery adds the goods selector to query.
func (selector GoodsSelector) EncodeQuery(query url.Values) error {
	return addExclusiveSelector(query, selector.kind, selector.code, "product", "productGrp")
}

// EncodeQuery adds the service selector to query.
func (selector ServiceSelector) EncodeQuery(query url.Values) error {
	if query == nil {
		return errors.New("trademap: query values must not be nil")
	}
	validated, err := Service(selector.code)
	if err != nil {
		return err
	}
	query.Set("service", string(validated.code))
	return nil
}

func addExclusiveSelector(query url.Values, kind selectorKind, code, individualKey, groupKey string) error {
	if query == nil {
		return errors.New("trademap: query values must not be nil")
	}
	switch kind {
	case selectorIndividual:
		query.Set(individualKey, code)
		query.Del(groupKey)
	case selectorGroup:
		query.Set(groupKey, code)
		query.Del(individualKey)
	default:
		return errors.New("trademap: selector is not initialized")
	}
	return nil
}
