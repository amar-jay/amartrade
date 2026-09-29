package trademap

// Availability is an inclusive period range advertised by Trade Map.
// Zero boundaries mean that the dataset is unavailable.
type Availability struct {
	FirstPeriod int
	LastPeriod  int
	Level       int
}

type Economy struct {
	Code              EconomyCode
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
	Code  EconomyCode
	Label string
}

type EconomyGroup struct {
	Code    EconomyGroupCode
	Label   string
	Type    string
	Note    string
	Members []EconomyMember
}

type HSProduct struct {
	Code      HSProductCode
	Label     string
	Revisions string
}

type ProductGroup struct {
	Code     ProductGroupCode
	Label    string
	Level    int
	Products []HSProduct
}

type EBOPSService struct {
	Code        ServiceCode
	DisplayCode string
	Label       string
	MaxLevel    string
}

type EconomyCatalog = Catalog[Economy, EconomyCode]
type EconomyGroupCatalog = Catalog[EconomyGroup, EconomyGroupCode]
type HSProductCatalog = Catalog[HSProduct, HSProductCode]
type ProductGroupCatalog = Catalog[ProductGroup, ProductGroupCode]
type EBOPSServiceCatalog = Catalog[EBOPSService, ServiceCode]
