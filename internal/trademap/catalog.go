package trademap

import (
	"fmt"
	"sort"
	"strings"
)

// LookupCandidate identifies one record involved in an ambiguous name lookup.
type LookupCandidate struct {
	Code  string
	Label string
}

// NotFoundError reports an exact code or name lookup that found no record.
type NotFoundError struct {
	Kind  string
	Query string
}

func (err *NotFoundError) Error() string {
	return fmt.Sprintf("trademap: %s %q not found", err.Kind, err.Query)
}

// AmbiguousMatchError reports a normalized exact-name lookup with multiple
// matches. Candidates are always sorted by code and then label.
type AmbiguousMatchError struct {
	Kind       string
	Query      string
	Candidates []LookupCandidate
}

func (err *AmbiguousMatchError) Error() string {
	codes := make([]string, len(err.Candidates))
	for index, candidate := range err.Candidates {
		codes[index] = candidate.Code
	}
	return fmt.Sprintf("trademap: %s name %q is ambiguous; matching codes: %s", err.Kind, err.Query, strings.Join(codes, ", "))
}

// Catalog is a typed, exact-match index over reference records. C is the
// catalog's distinct string-backed code type, preventing accidental lookups
// with a code from another classification.
type Catalog[T any, C ~string] struct {
	items  []T
	byCode map[string]int
	byName map[string][]int
	code   func(T) C
	label  func(T) string
	kind   string
}

func newCatalog[T any, C ~string](kind string, items []T, code func(T) C, label func(T) string) (*Catalog[T, C], error) {
	catalog := &Catalog[T, C]{
		items: items, byCode: make(map[string]int, len(items)),
		byName: make(map[string][]int, len(items)), code: code, label: label, kind: kind,
	}
	for position, item := range items {
		itemCode := string(code(item))
		itemLabel := strings.TrimSpace(label(item))
		if itemCode == "" || itemLabel == "" {
			return nil, fmt.Errorf("trademap: %s catalog contains an empty code or label", kind)
		}
		if _, exists := catalog.byCode[itemCode]; exists {
			return nil, fmt.Errorf("trademap: %s catalog contains duplicate code %q", kind, itemCode)
		}
		catalog.byCode[itemCode] = position
		normalized := normalizeLabel(itemLabel)
		catalog.byName[normalized] = append(catalog.byName[normalized], position)
	}
	return catalog, nil
}

// All returns catalog records in the provider's order.
func (catalog *Catalog[T, C]) All() []T { return catalog.items }

// ByCode performs an exact lookup using this catalog's code type.
func (catalog *Catalog[T, C]) ByCode(code C) (T, error) {
	if position, exists := catalog.byCode[string(code)]; exists {
		return catalog.items[position], nil
	}
	var zero T
	return zero, &NotFoundError{Kind: catalog.kind + " code", Query: string(code)}
}

// ByName performs a case-insensitive, whitespace-normalized exact lookup.
func (catalog *Catalog[T, C]) ByName(name string) (T, error) {
	positions := catalog.byName[normalizeLabel(name)]
	if len(positions) == 1 {
		return catalog.items[positions[0]], nil
	}
	var zero T
	if len(positions) == 0 {
		return zero, &NotFoundError{Kind: catalog.kind + " name", Query: name}
	}
	candidates := make([]LookupCandidate, len(positions))
	for candidateIndex, position := range positions {
		item := catalog.items[position]
		candidates[candidateIndex] = LookupCandidate{Code: string(catalog.code(item)), Label: catalog.label(item)}
	}
	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].Code == candidates[right].Code {
			return candidates[left].Label < candidates[right].Label
		}
		return candidates[left].Code < candidates[right].Code
	})
	return zero, &AmbiguousMatchError{Kind: catalog.kind, Query: name, Candidates: candidates}
}

func normalizeLabel(label string) string {
	return strings.ToLower(strings.Join(strings.Fields(label), " "))
}
