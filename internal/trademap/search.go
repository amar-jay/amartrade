package trademap

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

type SearchOptions struct {
	Query    string
	Type     string
	Limit    int
	Format   string
	Children bool
}

type HSNode struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type SearchResult struct {
	Type       string   `json:"type"`
	Selector   string   `json:"selector"`
	NativeCode string   `json:"native_code"`
	Label      string   `json:"label"`
	Level      int      `json:"level,omitempty"`
	Revisions  string   `json:"revisions,omitempty"`
	Hierarchy  []HSNode `json:"hierarchy,omitempty"`
	score      int
}

var searchTypes = map[string]bool{"all": true, "economy": true, "economy-group": true, "product": true, "product-group": true}

func (c *Client) Search(ctx context.Context, opts SearchOptions) ([]SearchResult, error) {
	kind := strings.ToLower(opts.Type)
	if !searchTypes[kind] {
		return nil, fmt.Errorf("--type must be all, economy, economy-group, product, or product-group")
	}
	if opts.Limit < 0 {
		return nil, fmt.Errorf("--limit must be zero or greater")
	}
	if opts.Format != "json" && opts.Format != "jsonl" && opts.Format != "csv" {
		return nil, fmt.Errorf("--format must be json, jsonl, or csv")
	}
	query := normalized(opts.Query)
	if query == "" {
		return nil, fmt.Errorf("search query must contain a letter or number")
	}
	results := make([]SearchResult, 0)
	if kind == "all" || kind == "economy" {
		var values []country
		if err := c.get(ctx, "/countries", nil, &values); err != nil {
			return nil, err
		}
		for _, value := range values {
			selector := displayCode(Selector{Kind: "economy", Code: value.Code, Label: value.Label})
			if score, ok := searchScore(query, value.Code, selector, value.Label); ok {
				results = append(results, SearchResult{Type: "economy", Selector: selector, NativeCode: value.Code, Label: value.Label, score: score})
			}
		}
	}
	if kind == "all" || kind == "economy-group" {
		var values []countryGroup
		if err := c.get(ctx, "/countries/groups/generic", url.Values{"loadMembers": {"false"}}, &values); err != nil {
			return nil, err
		}
		for _, value := range values {
			selector := normalized(value.Label)
			if score, ok := searchScore(query, strconv.Itoa(value.ID), selector, value.Label); ok {
				results = append(results, SearchResult{Type: "economy-group", Selector: selector, NativeCode: strconv.Itoa(value.ID), Label: value.Label, score: score})
			}
		}
	}
	if kind == "all" || kind == "product" {
		var values []product
		if err := c.get(ctx, "/products/HS", nil, &values); err != nil {
			return nil, err
		}
		byCode := make(map[string]product, len(values))
		exactCode := false
		for _, value := range values {
			byCode[value.Code] = value
			if normalized(value.Code) == query {
				exactCode = true
			}
		}
		for _, value := range values {
			if exactCode && !opts.Children && normalized(value.Code) != query {
				continue
			}
			level := len(value.Code)
			if value.Code == "ALL" {
				level = 0
			}
			if score, ok := searchScore(query, value.Code, value.Label); ok {
				results = append(results, SearchResult{Type: "product", Selector: value.Code, NativeCode: value.Code, Label: value.Label, Level: level, Revisions: value.Revisions, Hierarchy: hsHierarchy(value.Code, byCode), score: score})
			}
		}
	}
	if kind == "all" || kind == "product-group" {
		var values []productGroup
		if err := c.get(ctx, "/products/groups/generic", url.Values{"loadProducts": {"false"}}, &values); err != nil {
			return nil, err
		}
		for _, value := range values {
			native := strconv.Itoa(value.ID)
			if score, ok := searchScore(query, native, value.Label); ok {
				results = append(results, SearchResult{Type: "product-group", Selector: "group:" + native, NativeCode: native, Label: value.Label, Level: value.Level, score: score})
			}
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score < results[j].score
		}
		if results[i].Type != results[j].Type {
			return results[i].Type < results[j].Type
		}
		if results[i].Label != results[j].Label {
			return results[i].Label < results[j].Label
		}
		return results[i].NativeCode < results[j].NativeCode
	})
	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}
	return results, nil
}

func hsHierarchy(code string, products map[string]product) []HSNode {
	if code == "ALL" {
		return nil
	}
	path := make([]HSNode, 0, 3)
	for _, length := range []int{2, 4, 6} {
		if len(code) < length {
			break
		}
		if value, ok := products[code[:length]]; ok {
			path = append(path, HSNode{Code: value.Code, Label: value.Label})
		}
	}
	return path
}

func searchScore(query string, candidates ...string) (int, bool) {
	best := 3
	for _, candidate := range candidates {
		value := normalized(candidate)
		switch {
		case value == query:
			best = min(best, 0)
		case strings.HasPrefix(value, query):
			best = min(best, 1)
		case strings.Contains(value, query):
			best = min(best, 2)
		}
	}
	return best, best < 3
}

func WriteSearch(w io.Writer, results []SearchResult, format string) error {
	enc := json.NewEncoder(w)
	if format == "json" {
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	if format == "jsonl" {
		for _, result := range results {
			if err := enc.Encode(result); err != nil {
				return fmt.Errorf("write JSONL: %w", err)
			}
		}
		return nil
	}
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"TYPE", "SELECTOR", "NATIVE_CODE", "LABEL", "LEVEL", "REVISIONS", "HIERARCHY"}); err != nil {
		return err
	}
	for _, result := range results {
		level := ""
		if result.Level > 0 {
			level = strconv.Itoa(result.Level)
		}
		parts := make([]string, 0, len(result.Hierarchy))
		for _, node := range result.Hierarchy {
			parts = append(parts, node.Code+": "+node.Label)
		}
		if err := cw.Write([]string{result.Type, result.Selector, result.NativeCode, result.Label, level, result.Revisions, strings.Join(parts, " > ")}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
