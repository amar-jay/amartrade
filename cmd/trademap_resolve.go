package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/amar-jay/amartrade/internal/trademap/reference"
	"github.com/amar-jay/amartrade/internal/trademap/types"
	"github.com/spf13/cobra"
)

// resolver lazily loads reference catalogs only when a human-friendly value
// cannot be used directly. Exact API codes never trigger a network call, so
// scripted queries such as --reporter 276 stay offline and fast.
type resolver struct {
	service *reference.Service
	ctx     context.Context

	economies      *reference.EconomyCatalog
	economyGroups  *reference.EconomyGroupCatalog
	products       *reference.HSProductCatalog
	productGroups  *reference.ProductGroupCatalog
	services       *reference.EBOPSServiceCatalog
	loaded         map[string]bool
}

func newResolver(ctx context.Context, service *reference.Service) *resolver {
	return &resolver{service: service, ctx: ctx, loaded: map[string]bool{}}
}

func (r *resolver) economiesCatalog() (*reference.EconomyCatalog, error) {
	if r.economies == nil {
		catalog, err := r.service.Economies(r.ctx)
		if err != nil {
			return nil, err
		}
		r.economies = catalog
	}
	return r.economies, nil
}

func (r *resolver) economyGroupsCatalog() (*reference.EconomyGroupCatalog, error) {
	if r.economyGroups == nil {
		catalog, err := r.service.EconomyGroups(r.ctx)
		if err != nil {
			return nil, err
		}
		r.economyGroups = catalog
	}
	return r.economyGroups, nil
}

func (r *resolver) productsCatalog() (*reference.HSProductCatalog, error) {
	if r.products == nil {
		catalog, err := r.service.HSProducts(r.ctx)
		if err != nil {
			return nil, err
		}
		r.products = catalog
	}
	return r.products, nil
}

func (r *resolver) productGroupsCatalog() (*reference.ProductGroupCatalog, error) {
	if r.productGroups == nil {
		catalog, err := r.service.ProductGroups(r.ctx)
		if err != nil {
			return nil, err
		}
		r.productGroups = catalog
	}
	return r.productGroups, nil
}

func (r *resolver) servicesCatalog() (*reference.EBOPSServiceCatalog, error) {
	if r.services == nil {
		catalog, err := r.service.EBOPSServices(r.ctx)
		if err != nil {
			return nil, err
		}
		r.services = catalog
	}
	return r.services, nil
}

func explain(command *cobra.Command, format string, args ...any) {
	fmt.Fprintf(command.ErrOrStderr(), format+"\n", args...)
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

// resolveEconomy maps one human input to either an economy or an economy-group
// selector. It returns the selector kind ("economy" or "group"), the code that
// was sent, and a display label for stderr diagnostics.
func (r *resolver) resolveEconomy(input string) (kind, code, label string, reporter types.ReporterSelector, partner types.PartnerSelector, err error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", "", "", types.ReporterSelector{}, types.PartnerSelector{}, fmt.Errorf("economy must not be empty; try a name like %q or a code like 276", "Germany")
	}
	lowered := strings.ToLower(trimmed)
	switch lowered {
	case "world", "000", "wld", "*":
		rep, _ := types.ReporterEconomy(types.WorldEconomy)
		par, _ := types.PartnerEconomy(types.WorldEconomy)
		return "economy", "000", "World", rep, par, nil
	}

	// Fast path: exact three-digit code needs no catalog lookup.
	if candidate, cerr := types.NewEconomyCode(trimmed); cerr == nil {
		rep, _ := types.ReporterEconomy(candidate)
		par, _ := types.PartnerEconomy(candidate)
		return "economy", string(candidate), string(candidate), rep, par, nil
	}
	// Short digits such as "4" mean Afghanistan (004). Pad without a lookup.
	if isDigits(trimmed) && len(trimmed) < 3 {
		padded := strings.Repeat("0", 3-len(trimmed)) + trimmed
		if candidate, cerr := types.NewEconomyCode(padded); cerr == nil {
			rep, _ := types.ReporterEconomy(candidate)
			par, _ := types.PartnerEconomy(candidate)
			return "economy", string(candidate), string(candidate), rep, par, nil
		}
	}

	// Catalog path: exact names first, then numeric group IDs, then fuzzy help.
	economies, err := r.economiesCatalog()
	if err != nil {
		return "", "", "", types.ReporterSelector{}, types.PartnerSelector{}, err
	}
	groups, err := r.economyGroupsCatalog()
	if err != nil {
		return "", "", "", types.ReporterSelector{}, types.PartnerSelector{}, err
	}
	if item, err := economies.ByName(trimmed); err == nil {
		rep, _ := types.ReporterEconomy(item.Code)
		par, _ := types.PartnerEconomy(item.Code)
		return "economy", string(item.Code), item.Label, rep, par, nil
	} else if ambiguous, ok := err.(*reference.AmbiguousMatchError); ok {
		return "", "", "", types.ReporterSelector{}, types.PartnerSelector{}, fmt.Errorf("%s; be more specific", ambiguous.Error())
	}
	if item, err := groups.ByName(trimmed); err == nil {
		rep, _ := types.ReporterEconomyGroup(item.Code)
		par, _ := types.PartnerEconomyGroup(item.Code)
		return "group", string(item.Code), item.Label, rep, par, nil
	} else if ambiguous, ok := err.(*reference.AmbiguousMatchError); ok {
		return "", "", "", types.ReporterSelector{}, types.PartnerSelector{}, fmt.Errorf("%s; be more specific", ambiguous.Error())
	}
	// Bare group IDs such as 42 or 6757. "0" alone is World, handled above.
	if isDigits(trimmed) && strings.Trim(trimmed, "0") != "" {
		ungrouped := strings.TrimLeft(trimmed, "0")
		if item, err := groups.ByCode(types.EconomyGroupCode(ungrouped)); err == nil {
			rep, _ := types.ReporterEconomyGroup(item.Code)
			par, _ := types.PartnerEconomyGroup(item.Code)
			return "group", string(item.Code), item.Label, rep, par, nil
		}
		// Three-digit-looking groups with leading zeros never exist, but a
		// padded economy may: already handled above.
	}
	return "", "", "", types.ReporterSelector{}, types.PartnerSelector{}, r.economyNotFound(trimmed)
}

func (r *resolver) economyNotFound(query string) error {
	var hints []string
	if r.economies != nil {
		if matches, err := r.economies.Search(query, 3); err == nil {
			for _, match := range matches {
				hints = append(hints, fmt.Sprintf("%s (%s)", match.Item.Label, string(match.Item.Code)))
			}
		}
	} else if economies, err := r.economiesCatalog(); err == nil {
		if matches, err := economies.Search(query, 3); err == nil {
			for _, match := range matches {
				hints = append(hints, fmt.Sprintf("%s (%s)", match.Item.Label, string(match.Item.Code)))
			}
		}
	}
	if r.economyGroups != nil {
		if matches, err := r.economyGroups.Search(query, 2); err == nil {
			for _, match := range matches {
				hints = append(hints, fmt.Sprintf("%s (group %s)", match.Item.Label, string(match.Item.Code)))
			}
		}
	} else if groups, err := r.economyGroupsCatalog(); err == nil {
		if matches, err := groups.Search(query, 2); err == nil {
			for _, match := range matches {
				hints = append(hints, fmt.Sprintf("%s (group %s)", match.Item.Label, string(match.Item.Code)))
			}
		}
	}
	if len(hints) == 0 {
		return fmt.Errorf("economy %q not found; list candidates with: amartrade trademap reference economies --search %q", query, query)
	}
	return fmt.Errorf("economy %q not found; did you mean: %s", query, strings.Join(hints, "; "))
}

// resolveEconomyForReporter is a thin wrapper that discards the partner half.
func (r *resolver) resolveReporter(input string) (types.ReporterSelector, string, error) {
	kind, code, label, reporter, _, err := r.resolveEconomy(input)
	if err != nil {
		return types.ReporterSelector{}, "", err
	}
	return reporter, fmt.Sprintf("%s %s (%s)", kind, code, label), nil
}

func (r *resolver) resolvePartner(input string) (types.PartnerSelector, string, error) {
	kind, code, label, _, partner, err := r.resolveEconomy(input)
	if err != nil {
		return types.PartnerSelector{}, "", err
	}
	return partner, fmt.Sprintf("%s %s (%s)", kind, code, label), nil
}

// resolveGoods maps one human input to an HS product or product-group
// selector. "all", "ALL", and "*" select every product.
func (r *resolver) resolveGoods(input string) (types.GoodsSelector, string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return types.GoodsSelector{}, "", fmt.Errorf("product must not be empty; use %q for every product", "all")
	}
	switch strings.ToLower(trimmed) {
	case "all", "*", "total", "all products":
		selector, _ := types.GoodsProduct(types.AllGoods)
		return selector, "product ALL (All products)", nil
	}
	upper := strings.ToUpper(trimmed)
	if code, err := types.NewHSProductCode(upper); err == nil {
		// Fast path: a well-formed HS code or ALL needs no lookup.
		selector, _ := types.GoodsProduct(code)
		return selector, fmt.Sprintf("product %s", string(code)), nil
	}
	// Numeric group IDs such as 18 need catalog context; fall through.
	products, err := r.productsCatalog()
	if err != nil {
		return types.GoodsSelector{}, "", err
	}
	groups, err := r.productGroupsCatalog()
	if err != nil {
		return types.GoodsSelector{}, "", err
	}
	if isDigits(trimmed) && strings.Trim(trimmed, "0") != "" {
		if item, err := groups.ByCode(types.ProductGroupCode(strings.TrimLeft(trimmed, "0"))); err == nil {
			selector, _ := types.GoodsProductGroup(item.Code)
			return selector, fmt.Sprintf("product group %s (%s)", string(item.Code), item.Label), nil
		}
		// Zero-padded HS codes such as 01 keep their zeros; try them before
		// giving up so "0101" still resolves even if case differed.
		if item, err := products.ByCode(types.HSProductCode(trimmed)); err == nil {
			selector, _ := types.GoodsProduct(item.Code)
			return selector, fmt.Sprintf("product %s (%s)", string(item.Code), item.Label), nil
		}
	}
	if item, err := products.ByName(trimmed); err == nil {
		selector, _ := types.GoodsProduct(item.Code)
		return selector, fmt.Sprintf("product %s (%s)", string(item.Code), item.Label), nil
	} else if ambiguous, ok := err.(*reference.AmbiguousMatchError); ok {
		return types.GoodsSelector{}, "", fmt.Errorf("%s; be more specific", ambiguous.Error())
	}
	if item, err := groups.ByName(trimmed); err == nil {
		selector, _ := types.GoodsProductGroup(item.Code)
		return selector, fmt.Sprintf("product group %s (%s)", string(item.Code), item.Label), nil
	} else if ambiguous, ok := err.(*reference.AmbiguousMatchError); ok {
		return types.GoodsSelector{}, "", fmt.Errorf("%s; be more specific", ambiguous.Error())
	}
	return types.GoodsSelector{}, "", r.goodsNotFound(trimmed)
}

func (r *resolver) goodsNotFound(query string) error {
	var hints []string
	if matches, err := r.products.Search(query, 3); err == nil {
		for _, match := range matches {
			hints = append(hints, fmt.Sprintf("%s (%s)", match.Item.Label, string(match.Item.Code)))
		}
	}
	if r.productGroups != nil {
		if matches, err := r.productGroups.Search(query, 2); err == nil {
			for _, match := range matches {
				hints = append(hints, fmt.Sprintf("%s (group %s)", match.Item.Label, string(match.Item.Code)))
			}
		}
	}
	if len(hints) == 0 {
		return fmt.Errorf("product %q not found; list candidates with: amartrade trademap reference products --search %q", query, query)
	}
	return fmt.Errorf("product %q not found; did you mean: %s", query, strings.Join(hints, "; "))
}

// resolveService maps one human input to an EBOPS service selector.
func (r *resolver) resolveService(input string, all bool) (types.ServiceSelector, string, error) {
	trimmed := strings.TrimSpace(input)
	if all {
		if trimmed != "" {
			return types.ServiceSelector{}, "", fmt.Errorf("--service and --all-services are mutually exclusive")
		}
		return types.AggregateServices(), "service ALL (All services)", nil
	}
	if trimmed == "" {
		return types.ServiceSelector{}, "", fmt.Errorf("one of --service or --all-services is required")
	}
	switch strings.ToLower(trimmed) {
	case "all", "*", "total", "all services", "s":
		return types.AggregateServices(), "service ALL (All services)", nil
	}
	upper := strings.ToUpper(strings.ReplaceAll(trimmed, " ", ""))
	// Display codes such as 3.1.1 are resolved through the catalog below.
	if !strings.Contains(upper, ".") {
		if selector, err := types.Service(types.ServiceCode(upper)); err == nil {
			return selector, fmt.Sprintf("service %s", upper), nil
		}
	}
	services, err := r.servicesCatalog()
	if err != nil {
		return types.ServiceSelector{}, "", err
	}
	if item, err := services.ByCode(types.ServiceCode(upper)); err == nil {
		selector, _ := types.Service(item.Code)
		return selector, fmt.Sprintf("service %s (%s)", string(item.Code), item.Label), nil
	}
	for _, item := range services.All() {
		if strings.EqualFold(strings.TrimSpace(item.DisplayCode), strings.TrimSpace(trimmed)) {
			selector, err := types.Service(item.Code)
			if err != nil {
				continue
			}
			return selector, fmt.Sprintf("service %s (%s)", string(item.Code), item.Label), nil
		}
	}
	if item, err := services.ByName(trimmed); err == nil {
		selector, err := types.Service(item.Code)
		if err != nil {
			return types.ServiceSelector{}, "", err
		}
		return selector, fmt.Sprintf("service %s (%s)", string(item.Code), item.Label), nil
	} else if ambiguous, ok := err.(*reference.AmbiguousMatchError); ok {
		return types.ServiceSelector{}, "", fmt.Errorf("%s; be more specific", ambiguous.Error())
	}
	var hints []string
	if matches, err := services.Search(trimmed, 4); err == nil {
		for _, match := range matches {
			hints = append(hints, fmt.Sprintf("%s (%s)", match.Item.Label, string(match.Item.Code)))
		}
	}
	if len(hints) == 0 {
		return types.ServiceSelector{}, "", fmt.Errorf("service %q not found; list candidates with: amartrade trademap reference services --search %q", trimmed, trimmed)
	}
	return types.ServiceSelector{}, "", fmt.Errorf("service %q not found; did you mean: %s", trimmed, strings.Join(hints, "; "))
}

// Friendly period parsing -------------------------------------------------

var (
	quarterFirstPattern  = regexp.MustCompile(`^(?i)\s*(\d{4})\s*[-/._\s]*q\s*([1-4])\s*$`)
	quarterSecondPattern = regexp.MustCompile(`^(?i)\s*q\s*([1-4])\s*[-/._\s]*(\d{4})\s*$`)
	yearDashNumPattern   = regexp.MustCompile(`^\s*(\d{4})\s*[-/._\s]+\s*(\d{1,2})\s*$`)
	numDashYearPattern   = regexp.MustCompile(`^\s*(\d{1,2})\s*[-/._\s]+\s*(\d{4})\s*$`)
	plainYearPattern     = regexp.MustCompile(`^\s*(\d{4})\s*$`)
	plainSixPattern      = regexp.MustCompile(`^\s*(\d{6})\s*$`)
)

var monthNames = map[string]int{
	"jan": 1, "january": 1, "feb": 2, "february": 2, "mar": 3, "march": 3,
	"apr": 4, "april": 4, "may": 5, "jun": 6, "june": 6,
	"jul": 7, "july": 7, "aug": 8, "august": 8, "sep": 9, "sept": 9,
	"september": 9, "oct": 10, "october": 10, "nov": 11, "november": 11,
	"dec": 12, "december": 12,
}

// parseFriendlyPeriod accepts everyday spellings and returns the validated
// Trade Map period for the requested granularity:
//
//	2024, 2024-Q1, Q1-2024, 2024Q1, 2024-01, 2024/7, Jan-2024, 202401, 2024
func parseFriendlyPeriod(granularity types.PeriodGranularity, input string) (types.Period, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return types.Period{}, fmt.Errorf("period must not be empty")
	}
	// Exact provider encoding always wins (YYYY, YYYYQQ, YYYYMM).
	if period, err := types.ParsePeriod(granularity, trimmed); err == nil {
		return period, nil
	}
	switch granularity {
	case types.YearGranularity:
		if match := plainYearPattern.FindStringSubmatch(trimmed); match != nil {
			year, _ := strconv.Atoi(match[1])
			return types.NewYear(year)
		}
		return types.Period{}, fmt.Errorf("invalid yearly period %q; use YYYY such as 2024", input)
	case types.QuarterGranularity:
		if match := quarterFirstPattern.FindStringSubmatch(trimmed); match != nil {
			year, _ := strconv.Atoi(match[1])
			quarter, _ := strconv.Atoi(match[2])
			return types.NewQuarter(year, quarter)
		}
		if match := quarterSecondPattern.FindStringSubmatch(trimmed); match != nil {
			quarter, _ := strconv.Atoi(match[1])
			year, _ := strconv.Atoi(match[2])
			return types.NewQuarter(year, quarter)
		}
		if match := yearDashNumPattern.FindStringSubmatch(trimmed); match != nil {
			year, _ := strconv.Atoi(match[1])
			part, _ := strconv.Atoi(match[2])
			if part >= 1 && part <= 4 {
				return types.NewQuarter(year, part)
			}
		}
		if match := plainSixPattern.FindStringSubmatch(trimmed); match != nil {
			// 202401 means Q1 2024 in Trade Map's quarterly encoding.
			return types.ParsePeriod(granularity, match[1])
		}
		return types.Period{}, fmt.Errorf("invalid quarterly period %q; use YYYYQQ (202401) or 2024-Q1", input)
	case types.MonthGranularity:
		if year, month, ok := splitYearMonth(trimmed); ok {
			return types.NewMonth(year, month)
		}
		if match := yearDashNumPattern.FindStringSubmatch(trimmed); match != nil {
			year, _ := strconv.Atoi(match[1])
			month, _ := strconv.Atoi(match[2])
			if month >= 1 && month <= 12 {
				return types.NewMonth(year, month)
			}
		}
		if match := numDashYearPattern.FindStringSubmatch(trimmed); match != nil {
			month, _ := strconv.Atoi(match[1])
			year, _ := strconv.Atoi(match[2])
			if month >= 1 && month <= 12 {
				return types.NewMonth(year, month)
			}
		}
		if match := plainSixPattern.FindStringSubmatch(trimmed); match != nil {
			return types.ParsePeriod(granularity, match[1])
		}
		return types.Period{}, fmt.Errorf("invalid monthly period %q; use YYYYMM (202409) or 2024-09", input)
	default:
		return types.Period{}, fmt.Errorf("period granularity is not initialized")
	}
}

// splitYearMonth parses "2024-09", "09/2024", "Jan-2024", "2024-Jan".
// It returns false when the input does not hold exactly a year and a month.
func splitYearMonth(input string) (int, int, bool) {
	fields := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(input)), func(r rune) bool {
		return r == '-' || r == '/' || r == '.' || r == '_' || r == ' ' || r == ','
	})
	if len(fields) != 2 {
		return 0, 0, false
	}
	if year, err := strconv.Atoi(fields[0]); err == nil {
		if month, ok := lookupMonth(fields[1]); ok {
			return year, month, true
		}
	}
	if year, err := strconv.Atoi(fields[1]); err == nil {
		if month, ok := lookupMonth(fields[0]); ok {
			return year, month, true
		}
	}
	return 0, 0, false
}

func lookupMonth(part string) (int, bool) {
	if month, err := strconv.Atoi(part); err == nil {
		if month >= 1 && month <= 12 {
			return month, true
		}
		return 0, false
	}
	month, ok := monthNames[strings.ToLower(part)]
	return month, ok
}

func parseFriendlyPeriodRange(frequency types.Frequency, from, to string) (types.PeriodRange, error) {
	granularity := types.YearGranularity
	switch frequency {
	case types.Quarterly:
		granularity = types.QuarterGranularity
	case types.Monthly:
		granularity = types.MonthGranularity
	}
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" {
		return types.PeriodRange{}, fmt.Errorf("--from is required (for example 2020)")
	}
	if to == "" {
		to = from
	}
	start, err := parseFriendlyPeriod(granularity, from)
	if err != nil {
		return types.PeriodRange{}, fmt.Errorf("--from: %w", err)
	}
	end, err := parseFriendlyPeriod(granularity, to)
	if err != nil {
		return types.PeriodRange{}, fmt.Errorf("--to: %w", err)
	}
	return types.NewPeriodRange(start, end)
}
