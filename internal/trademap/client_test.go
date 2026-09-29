package trademap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetJSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/countries" {
			t.Errorf("path = %q, want /api/countries", request.URL.Path)
		}
		if got := request.URL.Query().Get("loadMembers"); got != "true" {
			t.Errorf("loadMembers = %q, want true", got)
		}
		if request.Header.Get("Accept") != "application/json" || request.Header.Get("User-Agent") != "amartrade-test" {
			t.Errorf("unexpected headers: %v", request.Header)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"countryCd":"004"}`))
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL+"/api"), WithUserAgent("amartrade-test"))
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		CountryCode string `json:"countryCd"`
	}
	if err := client.GetJSON(context.Background(), "/countries", url.Values{"loadMembers": {"true"}}, &result); err != nil {
		t.Fatal(err)
	}
	if result.CountryCode != "004" {
		t.Fatalf("country code = %q, want 004", result.CountryCode)
	}
}

func TestGetBytes(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/csv")
		_, _ = writer.Write([]byte("code,value\n004,1\n"))
	}))
	defer server.Close()
	client, err := NewClient(WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	body, mediaType, err := client.GetBytes(context.Background(), "export", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "code,value\n004,1\n" || mediaType != "text/csv" {
		t.Fatalf("body = %q, media type = %q", body, mediaType)
	}
}

func TestAPIError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "invalid period", http.StatusBadRequest)
	}))
	defer server.Close()
	client, err := NewClient(WithBaseURL(server.URL), WithMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	err = client.GetJSON(context.Background(), "data", nil, &struct{}{})
	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("error = %v, want *APIError", err)
	}
	if apiError.StatusCode != http.StatusBadRequest || apiError.Body != "invalid period" {
		t.Fatalf("unexpected API error: %#v", apiError)
	}
}

func TestRetriesTransientResponses(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) < 3 {
			writer.Header().Set("Retry-After", "7")
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	client, err := NewClient(WithBaseURL(server.URL), WithMaxRetries(2))
	if err != nil {
		t.Fatal(err)
	}
	var delays []time.Duration
	client.sleep = func(_ context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return nil
	}
	var result struct {
		OK bool `json:"ok"`
	}
	if err := client.GetJSON(context.Background(), "data", nil, &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || requests.Load() != 3 {
		t.Fatalf("result = %#v, requests = %d", result, requests.Load())
	}
	if len(delays) != 2 || delays[0] != 7*time.Second || delays[1] != 7*time.Second {
		t.Fatalf("retry delays = %v", delays)
	}
}

func TestResponseSizeLimit(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"value":"too large"}`))
	}))
	defer server.Close()
	client, err := NewClient(WithBaseURL(server.URL), WithMaxResponseSize(8))
	if err != nil {
		t.Fatal(err)
	}
	err = client.GetJSON(context.Background(), "data", nil, &struct{}{})
	if err == nil || !strings.Contains(err.Error(), "exceeds 8 bytes") {
		t.Fatalf("error = %v, want response-size error", err)
	}
}

func TestClientOptionValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		option Option
	}{
		{"relative base URL", WithBaseURL("api")},
		{"base URL query", WithBaseURL("https://example.com/api?x=1")},
		{"nil HTTP client", WithHTTPClient(nil)},
		{"empty user agent", WithUserAgent(" ")},
		{"negative retries", WithMaxRetries(-1)},
		{"invalid response size", WithMaxResponseSize(0)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewClient(test.option); err == nil {
				t.Fatal("NewClient returned nil error")
			}
		})
	}
}

func TestRejectsAbsoluteEndpoint(t *testing.T) {
	t.Parallel()
	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	err = client.GetJSON(context.Background(), "https://example.com/steal", nil, &struct{}{})
	if err == nil || !strings.Contains(err.Error(), "relative path") {
		t.Fatalf("error = %v", err)
	}
}
