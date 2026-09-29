package trademap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Economies fetches the authoritative Trade Map economy catalog.
func (client *Client) Economies(ctx context.Context) (*EconomyCatalog, error) {
	var wire []economyWire
	if err := client.GetJSON(ctx, "countries", nil, &wire); err != nil {
		return nil, err
	}
	items := make([]Economy, len(wire))
	for index, record := range wire {
		code, err := NewEconomyCode(record.Code)
		if err != nil {
			return nil, fmt.Errorf("trademap: decode economy at index %d: %w", index, err)
		}
		items[index] = Economy{
			Code: code, Label: record.Label, NES: record.NES, TradeIndicators: record.TradeIndicators,
			YearlyGoods: availability(record.YearlyGoods), YearlyTariffLines: availability(record.YearlyTariffLines),
			YearlyReexports: availability(record.YearlyReexports), YearlyServices: availability(record.YearlyServices),
			QuarterlyGoods: availability(record.QuarterlyGoods), QuarterlyServices: availability(record.QuarterlyServices),
			MonthlyGoods: availability(record.MonthlyGoods),
		}
	}
	return newCatalog("economy", items, func(item Economy) EconomyCode { return item.Code }, func(item Economy) string { return item.Label })
}

// EconomyGroups fetches generic economy groups with their literal membership.
func (client *Client) EconomyGroups(ctx context.Context) (*EconomyGroupCatalog, error) {
	var wire []economyGroupWire
	query := url.Values{"loadMembers": {"true"}}
	if err := client.GetJSON(ctx, "countries/groups/generic", query, &wire); err != nil {
		return nil, err
	}
	items := make([]EconomyGroup, len(wire))
	for index, record := range wire {
		code, err := NewEconomyGroupCode(record.ID.String())
		if err != nil {
			return nil, fmt.Errorf("trademap: decode economy group at index %d: %w", index, err)
		}
		members := make([]EconomyMember, len(record.Members))
		for memberIndex, member := range record.Members {
			memberCode, err := NewEconomyCode(member.Code)
			if err != nil {
				return nil, fmt.Errorf("trademap: decode member %d of economy group %s: %w", memberIndex, code, err)
			}
			members[memberIndex] = EconomyMember{Code: memberCode, Label: member.Label}
		}
		items[index] = EconomyGroup{Code: code, Label: record.Label, Type: record.Type, Note: record.Note, Members: members}
	}
	return newCatalog("economy group", items, func(item EconomyGroup) EconomyGroupCode { return item.Code }, func(item EconomyGroup) string { return item.Label })
}

// HSProducts fetches the HS product hierarchy. Country-specific NTL products
// are intentionally excluded from this general-purpose catalog.
func (client *Client) HSProducts(ctx context.Context) (*HSProductCatalog, error) {
	var wire []hsProductWire
	if err := client.GetJSON(ctx, "products/HS", nil, &wire); err != nil {
		return nil, err
	}
	items, err := decodeHSProducts(wire, "HS product")
	if err != nil {
		return nil, err
	}
	return newCatalog("HS product", items, func(item HSProduct) HSProductCode { return item.Code }, func(item HSProduct) string { return item.Label })
}

// ProductGroups fetches generic product groups and their products.
func (client *Client) ProductGroups(ctx context.Context) (*ProductGroupCatalog, error) {
	var wire []productGroupWire
	query := url.Values{"loadProducts": {"true"}}
	if err := client.GetJSON(ctx, "products/groups/generic", query, &wire); err != nil {
		return nil, err
	}
	items := make([]ProductGroup, len(wire))
	for index, record := range wire {
		code, err := NewProductGroupCode(record.ID.String())
		if err != nil {
			return nil, fmt.Errorf("trademap: decode product group at index %d: %w", index, err)
		}
		products, err := decodeHSProducts(record.Products, "product group "+string(code))
		if err != nil {
			return nil, err
		}
		items[index] = ProductGroup{Code: code, Label: record.Label, Level: record.Level, Products: products}
	}
	return newCatalog("product group", items, func(item ProductGroup) ProductGroupCode { return item.Code }, func(item ProductGroup) string { return item.Label })
}

// EBOPSServices fetches the EBOPS service hierarchy.
func (client *Client) EBOPSServices(ctx context.Context) (*EBOPSServiceCatalog, error) {
	var wire []serviceWire
	if err := client.GetJSON(ctx, "services/EBOPS", nil, &wire); err != nil {
		return nil, err
	}
	items := make([]EBOPSService, len(wire))
	for index, record := range wire {
		code, err := NewServiceCode(record.Code)
		if err != nil {
			return nil, fmt.Errorf("trademap: decode EBOPS service at index %d: %w", index, err)
		}
		items[index] = EBOPSService{Code: code, DisplayCode: record.DisplayCode, Label: record.Label, MaxLevel: record.MaxLevel}
	}
	return newCatalog("EBOPS service", items, func(item EBOPSService) ServiceCode { return item.Code }, func(item EBOPSService) string { return item.Label })
}

func decodeHSProducts(wire []hsProductWire, context string) ([]HSProduct, error) {
	items := make([]HSProduct, len(wire))
	for index, record := range wire {
		code, err := NewHSProductCode(record.Code)
		if err != nil {
			return nil, fmt.Errorf("trademap: decode %s at index %d: %w", context, index, err)
		}
		items[index] = HSProduct{Code: code, Label: record.Label, Revisions: record.Revisions}
	}
	return items, nil
}

func availability(value availabilityWire) Availability {
	return Availability{FirstPeriod: value.FirstPeriod, LastPeriod: value.LastPeriod, Level: value.Level}
}

type availabilityWire struct {
	FirstPeriod int `json:"firstPeriod"`
	LastPeriod  int `json:"lastPeriod"`
	Level       int `json:"level"`
}

type economyWire struct {
	Code              string           `json:"countryCd"`
	Label             string           `json:"label"`
	NES               bool             `json:"nes"`
	TradeIndicators   bool             `json:"ti"`
	YearlyGoods       availabilityWire `json:"yearly246"`
	YearlyTariffLines availabilityWire `json:"yearly10D"`
	YearlyReexports   availabilityWire `json:"yearlyReexport"`
	YearlyServices    availabilityWire `json:"yearlyServices"`
	QuarterlyGoods    availabilityWire `json:"quarterly"`
	QuarterlyServices availabilityWire `json:"quarterlyServices"`
	MonthlyGoods      availabilityWire `json:"monthly"`
}

type memberWire struct {
	Code  string `json:"countryCd"`
	Label string `json:"label"`
}

type economyGroupWire struct {
	ID      wireID       `json:"id"`
	Label   string       `json:"label"`
	Type    string       `json:"type"`
	Note    string       `json:"note"`
	Members []memberWire `json:"members"`
}

type hsProductWire struct {
	Code      string `json:"productCd"`
	Label     string `json:"label"`
	Revisions string `json:"revisions"`
}

type productGroupWire struct {
	ID       wireID          `json:"id"`
	Label    string          `json:"label"`
	Level    int             `json:"level"`
	Products []hsProductWire `json:"products"`
}

type serviceWire struct {
	Code        string `json:"productCd"`
	Label       string `json:"label"`
	DisplayCode string `json:"displayCd"`
	MaxLevel    string `json:"maxLevel"`
}

type wireID string

func (id *wireID) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return fmt.Errorf("ID must not be empty")
	}
	if trimmed[0] == '"' {
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return err
		}
		*id = wireID(value)
		return nil
	}
	value := string(trimmed)
	if strings.ContainsAny(value, ".eE+-") {
		return fmt.Errorf("ID must be a positive decimal integer: %q", value)
	}
	*id = wireID(value)
	return nil
}

func (id wireID) String() string { return string(id) }
