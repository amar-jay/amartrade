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

// SearchResult is one fuzzy catalog match. Score ranges from zero to one;
// exact code matches score highest, followed by exact and partial label
// matches, then edit-distance matches.
type SearchResult[T any] struct {
	Item  T
	Score float64
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

// Search returns up to limit deterministic fuzzy matches. It recognizes exact
// codes, normalized names, prefixes, word prefixes, substrings, and nearby
// spellings. Equal scores are ordered by code and then label.
func (catalog *Catalog[T, C]) Search(query string, limit int) ([]SearchResult[T], error) {
	normalizedQuery := normalizeLabel(query)
	if normalizedQuery == "" {
		return nil, fmt.Errorf("trademap: search query must not be empty")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("trademap: search limit must be positive")
	}

	type rankedResult struct {
		SearchResult[T]
		code  string
		label string
	}
	ranked := make([]rankedResult, 0, len(catalog.items))
	for _, item := range catalog.items {
		code := string(catalog.code(item))
		label := catalog.label(item)
		score := fuzzyScore(normalizedQuery, normalizeLabel(label), strings.ToLower(code))
		if score == 0 {
			continue
		}
		ranked = append(ranked, rankedResult{
			SearchResult: SearchResult[T]{Item: item, Score: score},
			code:         code,
			label:        label,
		})
	}

	sort.SliceStable(ranked, func(left, right int) bool {
		if ranked[left].Score != ranked[right].Score {
			return ranked[left].Score > ranked[right].Score
		}
		if ranked[left].code != ranked[right].code {
			return ranked[left].code < ranked[right].code
		}
		return ranked[left].label < ranked[right].label
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	results := make([]SearchResult[T], len(ranked))
	for index := range ranked {
		results[index] = ranked[index].SearchResult
	}
	return results, nil
}

func normalizeLabel(label string) string {
	return strings.ToLower(strings.Join(strings.Fields(label), " "))
}

func fuzzyScore(query, label, code string) float64 {
	if query == code {
		return 1
	}
	if query == label {
		return 0.99
	}
	if strings.HasPrefix(code, query) {
		return 0.97
	}
	if strings.HasPrefix(label, query) {
		return 0.95
	}
	for _, word := range strings.Fields(label) {
		if strings.HasPrefix(word, query) {
			return 0.90
		}
	}
	if strings.Contains(label, query) {
		return 0.85
	}
	if allTermsMatch(strings.Fields(query), strings.Fields(label)) {
		return 0.82
	}

	queryRunes, labelRunes := []rune(query), []rune(label)
	longest := max(len(queryRunes), len(labelRunes))
	if longest == 0 {
		return 0
	}
	distance := levenshteinDistance(queryRunes, labelRunes)
	if distance > max(2, longest/3) {
		return 0
	}
	similarity := 1 - float64(distance)/float64(longest)
	if similarity < 0.6 {
		return 0
	}
	return similarity * 0.8
}

func allTermsMatch(queryTerms, labelTerms []string) bool {
	if len(queryTerms) < 2 {
		return false
	}
	for _, queryTerm := range queryTerms {
		matched := false
		for _, labelTerm := range labelTerms {
			if strings.HasPrefix(labelTerm, queryTerm) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func levenshteinDistance(left, right []rune) int {
	if len(left) > len(right) {
		left, right = right, left
	}
	previous := make([]int, len(left)+1)
	current := make([]int, len(left)+1)
	for index := range previous {
		previous[index] = index
	}
	for rightIndex, rightRune := range right {
		current[0] = rightIndex + 1
		for leftIndex, leftRune := range left {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}
			current[leftIndex+1] = min(
				current[leftIndex]+1,
				previous[leftIndex+1]+1,
				previous[leftIndex]+cost,
			)
		}
		previous, current = current, previous
	}
	return previous[len(left)]
}
