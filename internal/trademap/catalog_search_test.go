package trademap

import "testing"

func TestCatalogSearchRanking(t *testing.T) {
	t.Parallel()
	catalog, err := newCatalog("economy", []Economy{
		{Code: "276", Label: "Germany"},
		{Code: "288", Label: "Ghana"},
		{Code: "792", Label: "Türkiye"},
		{Code: "004", Label: "Afghanistan"},
		{Code: "016", Label: "American Samoa"},
	}, func(item Economy) EconomyCode { return item.Code }, func(item Economy) string { return item.Label })
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		query string
		limit int
		want  []EconomyCode
	}{
		{"exact code ranks first", "792", 3, []EconomyCode{"792"}},
		{"partial code", "79", 3, []EconomyCode{"792"}},
		{"normalized exact name", "  GERMANY ", 3, []EconomyCode{"276"}},
		{"label prefix", "amer", 3, []EconomyCode{"016"}},
		{"word prefix", "samo", 3, []EconomyCode{"016"}},
		{"substring", "ghan", 3, []EconomyCode{"288", "004"}},
		{"misspelling", "germnay", 3, []EconomyCode{"276"}},
		{"limit", "a", 2, []EconomyCode{"004", "016"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			results, err := catalog.Search(test.query, test.limit)
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != len(test.want) {
				t.Fatalf("results = %#v, want codes %v", results, test.want)
			}
			for index, want := range test.want {
				if results[index].Item.Code != want {
					t.Fatalf("result %d code = %s, want %s", index, results[index].Item.Code, want)
				}
				if results[index].Score <= 0 || results[index].Score > 1 {
					t.Fatalf("result %d has invalid score %f", index, results[index].Score)
				}
			}
		})
	}
}

func TestCatalogSearchMatchesReorderedTerms(t *testing.T) {
	t.Parallel()
	catalog, err := newCatalog("service", []EBOPSService{
		{Code: "S03001001", Label: "Passenger transport, Sea"},
		{Code: "S03001002", Label: "Freight transport, Sea"},
	}, func(item EBOPSService) ServiceCode { return item.Code }, func(item EBOPSService) string { return item.Label })
	if err != nil {
		t.Fatal(err)
	}
	results, err := catalog.Search("sea passenger", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Item.Code != "S03001001" {
		t.Fatalf("results = %#v", results)
	}
}

func TestCatalogSearchStableTieBreak(t *testing.T) {
	t.Parallel()
	catalog, err := newCatalog("product", []HSProduct{
		{Code: "03", Label: "Alpha fish"},
		{Code: "01", Label: "Alpha animals"},
		{Code: "02", Label: "Alpha meat"},
	}, func(item HSProduct) HSProductCode { return item.Code }, func(item HSProduct) string { return item.Label })
	if err != nil {
		t.Fatal(err)
	}
	results, err := catalog.Search("alpha", 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []HSProductCode{"01", "02", "03"}
	for index := range want {
		if results[index].Item.Code != want[index] {
			t.Fatalf("result order = %#v, want %v", results, want)
		}
	}
}

func TestCatalogSearchValidation(t *testing.T) {
	t.Parallel()
	catalog, err := newCatalog("economy", []Economy{{Code: "000", Label: "World"}},
		func(item Economy) EconomyCode { return item.Code }, func(item Economy) string { return item.Label })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Search(" ", 1); err == nil {
		t.Fatal("empty query returned no error")
	}
	if _, err := catalog.Search("world", 0); err == nil {
		t.Fatal("zero limit returned no error")
	}
}
