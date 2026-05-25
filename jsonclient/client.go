package jsonclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
}

type userAgentTransport struct {
	base         http.RoundTripper
	getUserAgent func() string
}

// RoundTrip sets the User-Agent and delegates to the base transport
func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.Header = req.Header.Clone()
	cloned.Header.Set("User-Agent", t.getUserAgent())
	return t.base.RoundTrip(cloned)
}

type Option func(*Client)

// custom option for setting the http.Client
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithUserAgent sets the User-Agent
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		c.userAgent = ua
	}
}

func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:   baseURL,
		userAgent: "JSON-Client/1.0",
	}

	for _, opt := range opts {
		opt(c)
	}

	baseTransport := c.httpClient.Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	c.httpClient.Transport = &userAgentTransport{
		base: baseTransport,
		getUserAgent: func() string {
			return c.userAgent
		},
	}

	return c
}

type ResponseError struct {
	StatusCode int
	Status     string
	Body       []byte
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("HTTP %d %s: %s", e.StatusCode, e.Status, string(e.Body))
}

// Do makes an HTTP request with context, JSON encoding/decoding, and explicit status handling
func (c *Client) Do(ctx context.Context, method, path string, requestBody, responseBody interface{}) error {
	url := c.baseURL + path

	var bodyReader io.Reader
	if requestBody != nil {
		jsonData, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &ResponseError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       respBody,
		}
	}

	if responseBody != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, responseBody); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

func (c *Client) Get(ctx context.Context, path string, responseBody interface{}) error {
	return c.Do(ctx, http.MethodGet, path, nil, responseBody)
}

func (c *Client) Post(ctx context.Context, path string, requestBody, responseBody interface{}) error {
	return c.Do(ctx, http.MethodPost, path, requestBody, responseBody)
}

func (c *Client) Put(ctx context.Context, path string, requestBody, responseBody interface{}) error {
	return c.Do(ctx, http.MethodPut, path, requestBody, responseBody)
}

func (c *Client) Delete(ctx context.Context, path string, responseBody interface{}) error {
	return c.Do(ctx, http.MethodDelete, path, nil, responseBody)
}
