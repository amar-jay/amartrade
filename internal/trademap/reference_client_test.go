package trademap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReferenceCatalogCallsAndLookups(t *testing.T) {
	t.Parallel()
	responses := map[string]string{
		"/api/countries": `[
			{"countryCd":"000","label":"World","nes":false,"ti":true,"yearly246":{"firstPeriod":2001,"lastPeriod":2025,"level":6}},
			{"countryCd":"004","label":"  New   Land  ","nes":true,"ti":false}
		]`,
		"/api/countries/groups/generic": `[
			{"id":42,"label":"EU 27","type":"economicGrouping","note":"European Union","members":[
				{"countryCd":"040","label":"Austria"},{"countryCd":"492","label":"European Union Nes"}
			]}
		]`,
		"/api/products/HS": `[
			{"productCd":"ALL","label":"All products","revisions":"111111"},
			{"productCd":"01","label":"Live animals","revisions":"111111"},
			{"productCd":"02","label":" Same name ","revisions":"111111"},
			{"productCd":"03","label":"same   NAME","revisions":"111111"}
		]`,
		"/api/products/groups/generic": `[
			{"id":"18","label":"Vehicles","level":6,"products":[{"productCd":"8703","label":"Motor cars","revisions":"111111"}]}
		]`,
		"/api/services/EBOPS": `[
			{"productCd":"S00","label":"All services","displayCd":"S","maxLevel":"4"},
			{"productCd":"S03001001","label":"Passenger transport, Sea","displayCd":"3.1.1","maxLevel":"4"}
		]`,
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api/countries/groups/generic" && request.URL.Query().Get("loadMembers") != "true" {
			t.Errorf("loadMembers query = %q", request.URL.RawQuery)
		}
		if request.URL.Path == "/api/products/groups/generic" && request.URL.Query().Get("loadProducts") != "true" {
			t.Errorf("loadProducts query = %q", request.URL.RawQuery)
		}
		body, exists := responses[request.URL.Path]
		if !exists {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(body))
	}))
	defer server.Close()
	client, err := NewClient(WithBaseURL(server.URL + "/api"))
	if err != nil {
		t.Fatal(err)
	}

	economies, err := client.Economies(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	world, err := economies.ByCode(WorldEconomy)
	if err != nil {
		t.Fatal(err)
	}
	if world.Label != "World" || world.YearlyGoods.LastPeriod != 2025 || world.YearlyGoods.Level != 6 {
		t.Fatalf("unexpected world economy: %#v", world)
	}
	newLand, err := economies.ByName("new land")
	if err != nil || newLand.Code != "004" || !newLand.NES {
		t.Fatalf("normalized name lookup = %#v, %v", newLand, err)
	}

	groups, err := client.EconomyGroups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	eu, err := groups.ByCode("42")
	if err != nil {
		t.Fatal(err)
	}
	if eu.Code != "42" || len(eu.Members) != 2 || eu.Members[1].Code != "492" {
		t.Fatalf("group membership not preserved: %#v", eu)
	}
	if string(eu.Code) == string(eu.Members[0].Code) {
		t.Fatal("group ID was confused with a member economy code")
	}

	products, err := client.HSProducts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	product, err := products.ByCode("01")
	if err != nil || product.Label != "Live animals" {
		t.Fatalf("HS lookup = %#v, %v", product, err)
	}
	_, err = products.ByName(" SAME NAME ")
	var ambiguous *AmbiguousMatchError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("error = %v, want AmbiguousMatchError", err)
	}
	if len(ambiguous.Candidates) != 2 || ambiguous.Candidates[0].Code != "02" || ambiguous.Candidates[1].Code != "03" {
		t.Fatalf("candidates are not deterministic: %#v", ambiguous.Candidates)
	}

	productGroups, err := client.ProductGroups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	vehicles, err := productGroups.ByName("vehicles")
	if err != nil || vehicles.Code != "18" || len(vehicles.Products) != 1 || vehicles.Products[0].Code != "8703" {
		t.Fatalf("product group lookup = %#v, %v", vehicles, err)
	}

	services, err := client.EBOPSServices(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	total, err := services.ByCode(TotalServices)
	if err != nil || total.DisplayCode != "S" {
		t.Fatalf("service total lookup = %#v, %v", total, err)
	}
	sea, err := services.ByName("passenger transport, sea")
	if err != nil || sea.Code != "S03001001" {
		t.Fatalf("service name lookup = %#v, %v", sea, err)
	}
}

func TestCatalogLookupErrors(t *testing.T) {
	t.Parallel()
	catalog, err := newCatalog("economy", []Economy{{Code: "004", Label: "Afghanistan"}},
		func(item Economy) EconomyCode { return item.Code }, func(item Economy) string { return item.Label })
	if err != nil {
		t.Fatal(err)
	}
	_, err = catalog.ByCode("999")
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("error = %v, want NotFoundError", err)
	}
	_, err = catalog.ByName("missing")
	if !errors.As(err, &notFound) {
		t.Fatalf("error = %v, want NotFoundError", err)
	}
}

func TestCatalogRejectsDuplicateCodes(t *testing.T) {
	t.Parallel()
	_, err := newCatalog("test", []LookupCandidate{{Code: "1", Label: "One"}, {Code: "1", Label: "Other"}},
		func(item LookupCandidate) string { return item.Code }, func(item LookupCandidate) string { return item.Label })
	if err == nil {
		t.Fatal("duplicate code returned no error")
	}
}

func TestReferenceCallRejectsInvalidMemberCode(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`[{"id":42,"label":"EU","members":[{"countryCd":"40","label":"Austria"}]}]`))
	}))
	defer server.Close()
	client, err := NewClient(WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.EconomyGroups(context.Background()); err == nil {
		t.Fatal("invalid member code returned no error")
	}
}
