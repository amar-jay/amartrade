package trademap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL         = "https://www.trademap.org/api/"
	DefaultTimeout         = 30 * time.Second
	DefaultMaxRetries      = 2
	DefaultMaxResponseSize = int64(64 << 20) // 64 MiB
	defaultUserAgent       = "amartrade"
	maxErrorBodySize       = int64(64 << 10) // 64 KiB
)

// Client is safe for concurrent use. Its methods only perform transport-level
// work; endpoint-specific validation belongs to the higher-level services.
type Client struct {
	baseURL         *url.URL
	httpClient      *http.Client
	userAgent       string
	maxRetries      int
	maxResponseSize int64
	sleep           func(context.Context, time.Duration) error
}

// Option configures a Client.
type Option func(*clientConfig) error

type clientConfig struct {
	baseURL         string
	httpClient      *http.Client
	userAgent       string
	maxRetries      int
	maxResponseSize int64
}

// NewClient constructs a Trade Map client with finite timeouts and bounded
// retries. The default client retries 429, 502, 503, and 504 responses twice.
func NewClient(options ...Option) (*Client, error) {
	config := clientConfig{
		baseURL: DefaultBaseURL, httpClient: &http.Client{Timeout: DefaultTimeout},
		userAgent: defaultUserAgent, maxRetries: DefaultMaxRetries,
		maxResponseSize: DefaultMaxResponseSize,
	}
	for _, option := range options {
		if option == nil {
			return nil, errors.New("trademap: nil client option")
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}
	baseURL, err := parseBaseURL(config.baseURL)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL: baseURL, httpClient: config.httpClient, userAgent: config.userAgent,
		maxRetries: config.maxRetries, maxResponseSize: config.maxResponseSize,
		sleep: sleepContext,
	}, nil
}

// WithBaseURL changes the API root. A trailing slash is added automatically.
func WithBaseURL(rawURL string) Option {
	return func(config *clientConfig) error {
		if _, err := parseBaseURL(rawURL); err != nil {
			return err
		}
		config.baseURL = rawURL
		return nil
	}
}

// WithHTTPClient uses client for all requests. Its networking policy is kept.
func WithHTTPClient(client *http.Client) Option {
	return func(config *clientConfig) error {
		if client == nil {
			return errors.New("trademap: HTTP client must not be nil")
		}
		config.httpClient = client
		return nil
	}
}

// WithUserAgent sets the value sent in the User-Agent header.
func WithUserAgent(userAgent string) Option {
	return func(config *clientConfig) error {
		if strings.TrimSpace(userAgent) == "" {
			return errors.New("trademap: user agent must not be empty")
		}
		config.userAgent = userAgent
		return nil
	}
}

// WithMaxRetries sets the number of retries after the initial request.
func WithMaxRetries(maxRetries int) Option {
	return func(config *clientConfig) error {
		if maxRetries < 0 {
			return errors.New("trademap: maximum retries must not be negative")
		}
		config.maxRetries = maxRetries
		return nil
	}
}

// WithMaxResponseSize limits successful response bodies.
func WithMaxResponseSize(bytes int64) Option {
	return func(config *clientConfig) error {
		if bytes <= 0 {
			return errors.New("trademap: maximum response size must be positive")
		}
		config.maxResponseSize = bytes
		return nil
	}
}

// GetJSON performs an anonymous GET and decodes its JSON response into result.
func (client *Client) GetJSON(ctx context.Context, endpoint string, query url.Values, result any) error {
	if result == nil {
		return errors.New("trademap: JSON result must not be nil")
	}
	response, err := client.get(ctx, endpoint, query, "application/json")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := readLimited(response.Body, client.maxResponseSize)
	if err != nil {
		return fmt.Errorf("trademap: read %s: %w", response.Request.URL.Redacted(), err)
	}
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("trademap: decode %s: %w", response.Request.URL.Redacted(), err)
	}
	return nil
}

// GetBytes returns a response body and media type for CSV and Excel exports.
func (client *Client) GetBytes(ctx context.Context, endpoint string, query url.Values) ([]byte, string, error) {
	response, err := client.get(ctx, endpoint, query, "*/*")
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	body, err := readLimited(response.Body, client.maxResponseSize)
	if err != nil {
		return nil, "", fmt.Errorf("trademap: read %s: %w", response.Request.URL.Redacted(), err)
	}
	return body, response.Header.Get("Content-Type"), nil
}

func (client *Client) get(ctx context.Context, endpoint string, query url.Values, accept string) (*http.Response, error) {
	requestURL, err := client.resolve(endpoint, query)
	if err != nil {
		return nil, err
	}
	for attempt := 0; ; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("trademap: create request: %w", err)
		}
		request.Header.Set("Accept", accept)
		request.Header.Set("User-Agent", client.userAgent)
		response, err := client.httpClient.Do(request)
		if err != nil {
			return nil, fmt.Errorf("trademap: GET %s: %w", requestURL.Redacted(), err)
		}
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return response, nil
		}
		if retryableStatus(response.StatusCode) && attempt < client.maxRetries {
			delay := retryDelay(response.Header.Get("Retry-After"), attempt, time.Now())
			_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxErrorBodySize))
			response.Body.Close()
			if err := client.sleep(ctx, delay); err != nil {
				return nil, fmt.Errorf("trademap: retry %s: %w", requestURL.Redacted(), err)
			}
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, maxErrorBodySize))
		response.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("trademap: read error response from %s: %w", requestURL.Redacted(), readErr)
		}
		return nil, &APIError{StatusCode: response.StatusCode, Status: response.Status,
			Method: http.MethodGet, URL: requestURL.Redacted(), Body: strings.TrimSpace(string(body))}
	}
}

func (client *Client) resolve(endpoint string, query url.Values) (*url.URL, error) {
	reference, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("trademap: invalid endpoint %q: %w", endpoint, err)
	}
	if endpoint == "" || reference.IsAbs() || reference.Host != "" || reference.RawQuery != "" || reference.Fragment != "" {
		return nil, fmt.Errorf("trademap: endpoint must be a non-empty relative path without query or fragment: %q", endpoint)
	}
	reference.Path = strings.TrimLeft(reference.Path, "/")
	resolved := client.baseURL.ResolveReference(reference)
	resolved.RawQuery = query.Encode()
	return resolved, nil
}

func parseBaseURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("trademap: invalid base URL: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("trademap: base URL must be an absolute HTTP(S) URL without query or fragment: %q", rawURL)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/"
	return parsed, nil
}

func retryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusBadGateway ||
		status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

func retryDelay(header string, attempt int, now time.Time) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(header)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if date, err := http.ParseTime(header); err == nil && date.After(now) {
		return date.Sub(now)
	}
	return time.Duration(1<<attempt) * time.Second
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return body, nil
}
