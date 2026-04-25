package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func (c *Client) get(ctx context.Context, path string, params url.Values, dest any) error {
	return c.send(ctx, http.MethodGet, path, params, nil, dest)
}

func (c *Client) put(ctx context.Context, path string, payload any) error {
	return c.send(ctx, http.MethodPut, path, nil, payload, nil)
}

func (c *Client) post(ctx context.Context, path string, payload any) error {
	return c.send(ctx, http.MethodPost, path, nil, payload, nil)
}

func (c *Client) postJSON(ctx context.Context, path string, payload any, dest any) error {
	return c.send(ctx, http.MethodPost, path, nil, payload, dest)
}

func (c *Client) putParams(ctx context.Context, path string, params url.Values) error {
	return c.send(ctx, http.MethodPut, path, params, nil, nil)
}

func (c *Client) postParams(ctx context.Context, path string, params url.Values) error {
	return c.send(ctx, http.MethodPost, path, params, nil, nil)
}

func (c *Client) send(ctx context.Context, method, path string, params url.Values, payload any, dest any) error {
	const (
		// Loop while each step's `Retry-After` is ≤ maxRetryDelay AND the
		// cumulative wait stays ≤ maxTotalWait. Beyond that, the bucket has
		// escalated and the caller is better off surfacing than blocking.
		// These match the connect-state retry caps in connect_transport.go.
		maxRetryDelay = 10 * time.Second
		maxTotalWait  = 15 * time.Second
	)
	var totalWait time.Duration
	for {
		requestURL := c.baseURL + path
		if params == nil {
			if c.market != "" || c.language != "" || ((method == http.MethodPut || method == http.MethodPost || method == http.MethodDelete) && c.device != "") {
				params = url.Values{}
			}
		}
		if params != nil {
			if c.market != "" && params.Get("market") == "" {
				params.Set("market", c.market)
			}
			if c.language != "" && params.Get("locale") == "" {
				params.Set("locale", c.language)
			}
			if method == http.MethodPut || method == http.MethodPost || method == http.MethodDelete {
				if c.device != "" && params.Get("device_id") == "" {
					params.Set("device_id", c.device)
				}
			}
			if encoded := params.Encode(); encoded != "" {
				requestURL += "?" + encoded
			}
		}
		var body io.Reader
		if payload != nil {
			data, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			body = bytes.NewReader(data)
		}
		req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
		if err != nil {
			return err
		}
		token, err := c.token(ctx)
		if err != nil {
			return err
		}
		contentType := ""
		if payload != nil {
			contentType = "application/json"
		}
		applyRequestHeaders(req, requestHeaders{
			AccessToken: token,
			Accept:      "application/json",
			ContentType: contentType,
		})
		resp, err := c.client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := time.Duration(0)
			if header := resp.Header.Get("Retry-After"); header != "" {
				if seconds, err := strconv.Atoi(header); err == nil && seconds > 0 {
					retryAfter = time.Duration(seconds) * time.Second
				}
			}
			if retryAfter > maxRetryDelay {
				defer func() { _ = resp.Body.Close() }()
				return apiErrorFromResponse(resp)
			}
			wait := retryAfter + 250*time.Millisecond
			if totalWait+wait > maxTotalWait {
				defer func() { _ = resp.Body.Close() }()
				return apiErrorFromResponse(resp)
			}
			_ = resp.Body.Close()
			totalWait += wait
			// Force a token refresh on the next attempt — covers the
			// (rare) case where a stale token contributes to the 429.
			c.mu.Lock()
			c.lastToken = Token{}
			c.mu.Unlock()
			select {
			case <-time.After(wait):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == http.StatusNoContent {
			if dest != nil {
				return ErrNoContent
			}
			return nil
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return apiErrorFromResponse(resp)
		}
		if dest == nil {
			return nil
		}
		if resp.ContentLength == 0 {
			return nil
		}
		return json.NewDecoder(resp.Body).Decode(dest)
	}
}
