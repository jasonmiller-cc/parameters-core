// Package client provides a configured HTTP client with retry logic and
// service-to-service authentication for parameters internal APIs.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a configured HTTP client for inter-service calls.
type Client struct {
	base    string
	http    *http.Client
	token   string
	apiKey  string
	maxRetry int
}

type Option func(*Client)

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.http.Timeout = d }
}

// WithBearerToken sets the Authorization header for all requests.
func WithBearerToken(token string) Option {
	return func(c *Client) { c.token = token }
}

// WithAPIKey sets the Authorization header for all requests.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKey = key }
}

// WithRetry configures the maximum number of retry attempts on 5xx.
func WithRetry(max int) Option {
	return func(c *Client) { c.maxRetry = max }
}

// New creates a Client pointed at baseURL.
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		base:     baseURL,
		http:     &http.Client{Timeout: 30 * time.Second},
		maxRetry: 2,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Get sends a GET request and JSON-decodes the response into out.
func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

// Post sends a POST request with JSON body and decodes the response.
func (c *Client) Post(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

// Put sends a PUT request with JSON body and decodes the response.
func (c *Client) Put(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPut, path, body, out)
}

// Delete sends a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyBytes = b
	}

	var lastErr error
	for attempt := range c.maxRetry + 1 {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}

		var bodyR io.Reader
		if bodyBytes != nil {
			bodyR = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, c.base+path, bodyR)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Accept", "application/json")

		switch {
		case c.token != "":
			req.Header.Set("Authorization", "Bearer "+c.token)
		case c.apiKey != "":
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode >= 500 && attempt < c.maxRetry {
			lastErr = fmt.Errorf("server error %d", resp.StatusCode)
			_ = resp.Body.Close()
			continue
		}

		if resp.StatusCode >= 400 {
			b, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return fmt.Errorf("http %d: %s", resp.StatusCode, string(b))
		}

		if out != nil {
			err := json.NewDecoder(resp.Body).Decode(out)
			_ = resp.Body.Close()
			return err
		}
		_ = resp.Body.Close()
		return nil
	}
	return lastErr
}
