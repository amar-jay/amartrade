package reference

import "github.com/amar-jay/amartrade/internal/trademap/types"

// Availability is an inclusive period range advertised by Trade Map.
// Zero boundaries mean that the dataset is unavailable.
type Availability struct {
	FirstPeriod int
	LastPeriod  int
	Level       int
}

type Economy struct {
	Code              types.EconomyCode
	Label             string
	NES               bool
	TradeIndicators   bool
	YearlyGoods       Availability
	YearlyTariffLines Availability
	YearlyReexports   Availability
	YearlyServices    Availability
	QuarterlyGoods    Availability
	QuarterlyServices Availability
	MonthlyGoods      Availability
}

// EconomyMember always carries an economy code, never its containing group ID.
type EconomyMember struct {
	Code  types.EconomyCode
	Label string
}

type EconomyGroup struct {
	Code    types.EconomyGroupCode
	Label   string
	Type    string
	Note    string
	Members []EconomyMember
}

type HSProduct struct {
	Code      types.HSProductCode
	Label     string
	Revisions string
}

type ProductGroup struct {
	Code     types.ProductGroupCode
	Label    string
	Level    int
	Products []HSProduct
}

type EBOPSService struct {
	Code        types.ServiceCode
	DisplayCode string
	Label       string
	MaxLevel    string
}

type EconomyCatalog = Catalog[Economy, types.EconomyCode]
type EconomyGroupCatalog = Catalog[EconomyGroup, types.EconomyGroupCode]
type HSProductCatalog = Catalog[HSProduct, types.HSProductCode]
type ProductGroupCatalog = Catalog[ProductGroup, types.ProductGroupCode]
type EBOPSServiceCatalog = Catalog[EBOPSService, types.ServiceCode]
