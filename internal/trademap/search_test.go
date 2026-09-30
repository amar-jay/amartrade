package trademap

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSearchRanksCopyReadySelectors(t *testing.T) {
	bodies := map[string]string{
		"/countries":                `[{"countryCd":"156","label":"China"}]`,
		"/countries/groups/generic": `[{"id":42,"label":"EU 27"}]`,
		"/products/HS":              `[{"productCd":"01","label":"Live animals"},{"productCd":"0101","label":"Live horses"}]`,
		"/products/groups/generic":  `[{"id":18,"label":"Vehicles"}]`,
	}
	client := NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := bodies[strings.TrimPrefix(r.URL.Path, "/api")]
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})})
	results, err := client.Search(context.Background(), SearchOptions{Query: "01", Type: "all", Limit: 20, Format: "json"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Selector != "01" || results[0].Level != 2 || len(results[0].Hierarchy) != 1 {
		t.Fatalf("unexpected results: %#v", results)
	}
	children, err := client.Search(context.Background(), SearchOptions{Query: "01", Type: "product", Limit: 20, Format: "json", Children: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 2 || children[1].Selector != "0101" || len(children[1].Hierarchy) != 2 {
		t.Fatalf("unexpected children: %#v", children)
	}
	groups, err := client.Search(context.Background(), SearchOptions{Query: "vehicles", Type: "product-group", Limit: 20, Format: "json"})
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Selector != "group:18" {
		t.Fatalf("unexpected group result: %#v", groups)
	}
}

func TestWriteSearchJSONLAndCSV(t *testing.T) {
	results := []SearchResult{{Type: "economy", Selector: "CHN", NativeCode: "156", Label: "China"}}
	var jsonl bytes.Buffer
	if err := WriteSearch(&jsonl, results, "jsonl"); err != nil {
		t.Fatal(err)
	}
	if got := jsonl.String(); !strings.Contains(got, `"selector":"CHN"`) || strings.Count(got, "\n") != 1 {
		t.Fatalf("unexpected JSONL: %q", got)
	}
	var csv bytes.Buffer
	if err := WriteSearch(&csv, results, "csv"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(csv.String(), "TYPE,SELECTOR,NATIVE_CODE,LABEL,LEVEL,REVISIONS,HIERARCHY\neconomy,CHN,156,China,,,") {
		t.Fatalf("unexpected CSV: %q", csv.String())
	}
}
