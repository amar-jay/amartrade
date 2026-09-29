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
	exclusiveSelector
}

// PartnerSelector selects exactly one partner economy or economy group.
type PartnerSelector struct {
	exclusiveSelector
}

// GoodsSelector selects exactly one HS product or product group.
type GoodsSelector struct {
	exclusiveSelector
}

type exclusiveSelector struct {
	kind selectorKind
	code string
}

// ServiceSelector selects one EBOPS service or aggregate services. S00 is a
// response-only total and is rejected when a selector is serialized.
type ServiceSelector struct{ code ServiceCode }

func ReporterEconomy(code EconomyCode) (ReporterSelector, error) {
	selector, err := newExclusiveSelector(code, NewEconomyCode, selectorIndividual)
	return ReporterSelector{selector}, err
}

func ReporterEconomyGroup(code EconomyGroupCode) (ReporterSelector, error) {
	selector, err := newExclusiveSelector(code, NewEconomyGroupCode, selectorGroup)
	return ReporterSelector{selector}, err
}

func PartnerEconomy(code EconomyCode) (PartnerSelector, error) {
	selector, err := newExclusiveSelector(code, NewEconomyCode, selectorIndividual)
	return PartnerSelector{selector}, err
}

func PartnerEconomyGroup(code EconomyGroupCode) (PartnerSelector, error) {
	selector, err := newExclusiveSelector(code, NewEconomyGroupCode, selectorGroup)
	return PartnerSelector{selector}, err
}

func GoodsProduct(code HSProductCode) (GoodsSelector, error) {
	selector, err := newExclusiveSelector(code, NewHSProductCode, selectorIndividual)
	return GoodsSelector{selector}, err
}

func GoodsProductGroup(code ProductGroupCode) (GoodsSelector, error) {
	selector, err := newExclusiveSelector(code, NewProductGroupCode, selectorGroup)
	return GoodsSelector{selector}, err
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

func newExclusiveSelector[C ~string](code C, validate func(string) (C, error), kind selectorKind) (exclusiveSelector, error) {
	validated, err := validate(string(code))
	if err != nil {
		return exclusiveSelector{}, err
	}
	return exclusiveSelector{kind: kind, code: string(validated)}, nil
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
