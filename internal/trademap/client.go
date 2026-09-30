package trademap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{baseURL: defaultBaseURL, http: httpClient}
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("Trade Map request: %w", err)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			err = json.NewDecoder(resp.Body).Decode(out)
			resp.Body.Close()
			if err != nil {
				return fmt.Errorf("decode Trade Map response: %w", err)
			}
			return nil
		}
		resp.Body.Close()
		last = fmt.Errorf("Trade Map returned HTTP %d", resp.StatusCode)
		if resp.StatusCode != 429 && resp.StatusCode != 502 && resp.StatusCode != 503 && resp.StatusCode != 504 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 250 * time.Millisecond):
		}
	}
	return last
}

func parseYears(raw string) (int, int, error) {
	parts := strings.Split(raw, ":")
	if len(parts) > 2 {
		return 0, 0, fmt.Errorf("invalid --years %q", raw)
	}
	from, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid --years %q", raw)
	}
	to := from
	if len(parts) == 2 {
		to, err = strconv.Atoi(parts[1])
	}
	if err != nil || from < 1900 || to < from || to > 9999 {
		return 0, 0, fmt.Errorf("invalid --years %q", raw)
	}
	return from, to, nil
}

func (c *Client) Goods(ctx context.Context, opts GoodsOptions) (Result, error) {
	fromYear, toYear, err := parseYears(opts.Years)
	if err != nil {
		return Result{}, err
	}
	by := strings.ToLower(opts.By)
	if by != "country" && by != "partner" && by != "product" {
		return Result{}, fmt.Errorf("--by must be country, partner, or product")
	}
	if opts.Format != "json" && opts.Format != "jsonl" && opts.Format != "csv" {
		return Result{}, fmt.Errorf("--format must be json, jsonl, or csv")
	}
	if opts.Raw && opts.Format != "json" {
		return Result{}, fmt.Errorf("--raw requires --format json")
	}
	if by == "product" && opts.HSLevel != 2 && opts.HSLevel != 4 && opts.HSLevel != 6 && opts.HSLevel != 10 {
		return Result{}, fmt.Errorf("--hs-level must be 2, 4, 6, or 10")
	}
	dm := map[string]string{"direct": "D", "mirror": "M", "mixed": "X", "D": "D", "M": "M", "X": "X"}[opts.DirectMirror]
	if dm == "" {
		return Result{}, fmt.Errorf("--data must be direct, mirror, or mixed")
	}

	var countries []country
	var countryGroups []countryGroup
	var products []product
	var productGroups []productGroup
	if err = c.get(ctx, "/countries", nil, &countries); err != nil {
		return Result{}, err
	}
	if err = c.get(ctx, "/countries/groups/generic", url.Values{"loadMembers": {"true"}}, &countryGroups); err != nil {
		return Result{}, err
	}
	if err = c.get(ctx, "/products/HS", nil, &products); err != nil {
		return Result{}, err
	}
	if err = c.get(ctx, "/products/groups/generic", url.Values{"loadProducts": {"false"}}, &productGroups); err != nil {
		return Result{}, err
	}
	reporter, err := resolveEconomy(opts.From, countries, countryGroups)
	if err != nil {
		return Result{}, err
	}
	partner, err := resolveEconomy(opts.To, countries, countryGroups)
	if err != nil {
		return Result{}, err
	}
	prod, err := resolveProduct(opts.Product, products, productGroups)
	if err != nil {
		return Result{}, err
	}

	q := url.Values{"tradeFlow": {opts.Flow}, "periodFrom": {strconv.Itoa(fromYear)}, "periodTo": {strconv.Itoa(toYear)}, "directMirror": {dm}, "indicator": {"VAL"}, "currency": {"USD"}, "pageSize": {"500"}}
	addSelector(q, "country", "countryGrp", reporter)
	addSelector(q, "partner", "partnerGrp", partner)
	addSelector(q, "product", "productGrp", prod)
	if by == "product" {
		q.Set("hsLevel", strconv.Itoa(opts.HSLevel))
	}
	query := Query{Flow: opts.Flow, By: by, Reporter: reporter, Partner: partner, Product: prod, PeriodFrom: fromYear, PeriodTo: toYear, DirectMirror: dm, Indicator: "VAL", Currency: "USD", ValueScale: "units"}
	if by == "product" {
		query.HSLevel = opts.HSLevel
	}
	result := Result{Query: query, FetchedAt: time.Now().UTC()}
	labels := makeLabels(countries, countryGroups, products, productGroups)
	for pageNo := 1; ; pageNo++ {
		q.Set("page", strconv.Itoa(pageNo))
		var p page
		if err = c.get(ctx, "/goods/timeSeries/yearly/by"+upperFirst(by), q, &p); err != nil {
			return Result{}, err
		}
		for _, wr := range p.Records {
			result.Records = append(result.Records, normalizeRecord(wr, false, labels))
		}
		for _, wr := range p.AggregateRecords {
			result.Records = append(result.Records, normalizeRecord(wr, true, labels))
		}
		if pageNo == 1 {
			result.Sources = p.Sources
		}
		if p.NumPages == 0 || pageNo >= p.NumPages {
			break
		}
	}
	result.Records = removeDuplicateAggregates(result.Records)
	return result, nil
}

func removeDuplicateAggregates(records []JSONRecord) []JSONRecord {
	ordinary := make(map[string]struct{}, len(records))
	for _, record := range records {
		if !record.Aggregate {
			ordinary[recordIdentity(record)] = struct{}{}
		}
	}
	filtered := records[:0]
	for _, record := range records {
		if record.Aggregate {
			if _, duplicate := ordinary[recordIdentity(record)]; duplicate {
				continue
			}
		}
		filtered = append(filtered, record)
	}
	return filtered
}

func recordIdentity(record JSONRecord) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\x00%s\x00%s", record.Reporter.Code, record.Partner.Code, record.Product.Code)
	for _, point := range record.Data {
		fmt.Fprintf(&b, "\x00%d:", point.Period)
		if point.Value != nil {
			fmt.Fprintf(&b, "%d", *point.Value)
		}
		if point.Unit != nil {
			b.WriteString(":" + *point.Unit)
		}
		if point.Flag != nil {
			b.WriteString(":" + *point.Flag)
		}
	}
	return b.String()
}

func upperFirst(s string) string { return strings.ToUpper(s[:1]) + s[1:] }
func addSelector(q url.Values, individual, group string, s Selector) {
	if s.Kind == "group" {
		q.Set(group, s.Code)
	} else {
		q.Set(individual, s.Code)
	}
}

type labelSets struct{ economies, groups, products, productGroups map[string]string }

func makeLabels(cs []country, gs []countryGroup, ps []product, pgs []productGroup) labelSets {
	l := labelSets{map[string]string{}, map[string]string{}, map[string]string{}, map[string]string{}}
	for _, v := range cs {
		l.economies[v.Code] = v.Label
	}
	for _, v := range gs {
		l.groups[strconv.Itoa(v.ID)] = v.Label
	}
	for _, v := range ps {
		l.products[v.Code] = v.Label
	}
	for _, v := range pgs {
		l.productGroups[strconv.Itoa(v.ID)] = v.Label
	}
	l.products["TOTAL"] = "All products"
	return l
}
func normalizeRecord(w wireRecord, aggregate bool, l labelSets) JSONRecord {
	for i := range w.Data {
		if w.Data[i].Value != nil {
			v := *w.Data[i].Value * 1000
			w.Data[i].Value = &v
		}
	}
	rLabel, pLabel, productLabel := w.ReporterLabel, w.PartnerLabel, w.ProductLabel
	if rLabel == "" {
		rLabel = l.economies[w.ReporterCode]
		if rLabel == "" {
			rLabel = l.groups[w.ReporterCode]
		}
	}
	if pLabel == "" {
		pLabel = l.economies[w.PartnerCode]
		if pLabel == "" {
			pLabel = l.groups[w.PartnerCode]
		}
	}
	if productLabel == "" {
		productLabel = l.products[w.ProductCode]
		if productLabel == "" {
			productLabel = l.productGroups[w.ProductCode]
		}
	}
	rKind, pKind, productKind := "economy", "economy", "product"
	if _, ok := l.groups[w.ReporterCode]; ok {
		rKind = "group"
	}
	if _, ok := l.groups[w.PartnerCode]; ok {
		pKind = "group"
	}
	if _, ok := l.productGroups[w.ProductCode]; ok {
		productKind = "group"
	}
	return JSONRecord{Reporter: Selector{Kind: rKind, Code: w.ReporterCode, Label: rLabel}, Partner: Selector{Kind: pKind, Code: w.PartnerCode, Label: pLabel}, Product: Selector{Kind: productKind, Code: w.ProductCode, Label: productLabel}, Aggregate: aggregate, Data: w.Data}
}
