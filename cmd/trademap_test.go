package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTradeMapHelp(t *testing.T) {
	command := NewRootCommand("dev", "unknown")
	command.SetArgs([]string{"trademap", "--help"})
	var stdout, stderr strings.Builder
	if err := execute(command, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"reference", "goods", "services"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help missing %s:\n%s", want, stdout.String())
		}
	}
}

func TestGoodsTimeSeriesRequiresSelectors(t *testing.T) {
	command := NewRootCommand("dev", "unknown")
	command.SetArgs([]string{
		"trademap", "goods", "time-series",
		"--frequency", "yearly", "--dimension", "byCountry",
		"--from", "2024", "--to", "2025", "--flow", "E",
	})
	err := execute(command, strings.NewReader(""), &strings.Builder{}, &strings.Builder{})
	if err == nil || !strings.Contains(err.Error(), "reporter") {
		t.Fatalf("expected reporter validation error, got %v", err)
	}
}

func TestGoodsTimeSeriesJSON(t *testing.T) {
	var gotPath, gotProduct string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotPath = request.URL.Path
		gotProduct = request.URL.Query().Get("product")
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"nbRecords":1,"page":1,"nbRecordPerPage":100,"nbPages":1,"records":[{"reporterCd":"276","partnerCd":"000","productCd":"TOTAL","data":[{"period":2024,"value":10}]}],"aggregateRecords":[],"sources":[]}`))
	}))
	defer server.Close()

	command := NewRootCommand("dev", "unknown")
	command.SetArgs([]string{
		"trademap", "--base-url", server.URL, "--format", "json",
		"goods", "time-series",
		"--frequency", "yearly", "--dimension", "byCountry",
		"--reporter", "276", "--partner", "000", "--product", "ALL",
		"--from", "2024", "--to", "2024", "--flow", "exports",
	})
	var stdout strings.Builder
	if err := execute(command, strings.NewReader(""), &stdout, &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/goods/timeSeries/yearly/byCountry" {
		t.Fatalf("path %s", gotPath)
	}
	if gotProduct != "ALL" {
		t.Fatalf("product %s", gotProduct)
	}
	if !strings.Contains(stdout.String(), `"reporterCd": "276"`) && !strings.Contains(stdout.String(), "276") {
		t.Fatalf("stdout %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "276") {
		t.Fatalf("stdout %s", stdout.String())
	}
}

func TestReferenceEconomiesTable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/countries" {
			t.Errorf("path %s", request.URL.Path)
		}
		_, _ = writer.Write([]byte(`[{"countryCd":"000","label":"World","nes":false,"ti":true}]`))
	}))
	defer server.Close()

	command := NewRootCommand("dev", "unknown")
	command.SetArgs([]string{"trademap", "--base-url", server.URL, "reference", "economies"})
	var stdout strings.Builder
	if err := execute(command, strings.NewReader(""), &stdout, &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "000") || !strings.Contains(stdout.String(), "World") {
		t.Fatalf("stdout %s", stdout.String())
	}
}

func TestServicesTimeSeriesAggregate(t *testing.T) {
	var serviceCode, level string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		serviceCode = request.URL.Query().Get("service")
		level = request.URL.Query().Get("bpmLevel")
		_, _ = writer.Write([]byte(`{"nbRecords":1,"page":1,"nbPages":1,"records":[{"reporterCd":"276","partnerCd":"000","productCd":"S00","data":[{"period":2024,"value":1}]}],"sources":[]}`))
	}))
	defer server.Close()

	command := NewRootCommand("dev", "unknown")
	command.SetArgs([]string{
		"trademap", "--base-url", server.URL, "--format", "jsonl",
		"services", "time-series",
		"--frequency", "yearly", "--dimension", "byService",
		"--reporter", "276", "--partner", "000", "--all-services",
		"--service-level", "3",
		"--from", "2024", "--to", "2024", "--flow", "I",
	})
	var stdout strings.Builder
	if err := execute(command, strings.NewReader(""), &stdout, &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	if serviceCode != "ALL" || level != "3" {
		t.Fatalf("service=%s level=%s", serviceCode, level)
	}
	if !strings.Contains(stdout.String(), "S00") {
		t.Fatalf("stdout %s", stdout.String())
	}
}
