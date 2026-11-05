// Package dsh is the Go client for a running dsh web process:
// JSON-RPC over HTTP, WebSocket mux (/api/remote.mux), and $events.
package dsh

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Endpoint is an authenticated dsh web origin.
type Endpoint struct {
	Origin string
	Cookie string
}

// Error is a dsh RPC failure (`result.ok == false`).
type Error struct {
	Code    string
	Message string
	Details json.RawMessage
}

func (e *Error) Error() string {
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}

// Client talks to one dsh web origin.
type Client struct {
	Endpoint Endpoint
	HTTP     *http.Client
}

var defaultHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        128,
		MaxIdleConnsPerHost: 64,
		IdleConnTimeout:     90 * time.Second,
	},
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return defaultHTTPClient
}

// Call posts a Typert Remote client-request and unmarshals result.value into out.
// args must be a JSON object (map or struct); nil becomes {}.
func (c *Client) Call(ctx context.Context, method string, args any, out any) error {
	if err := checkMethod(method); err != nil {
		return err
	}
	if args == nil {
		args = map[string]any{}
	}
	return c.callPayload(ctx, method, map[string]any{"args": args}, out)
}

// CallWithProgress reports bytes consumed from the authenticated request body.
// Reaching total means transmission completed, not that the Host accepted it.
func (c *Client) CallWithProgress(ctx context.Context, method string, args any, out any, progress func(sent, total int64)) error {
	if err := checkMethod(method); err != nil {
		return err
	}
	if args == nil {
		args = map[string]any{}
	}
	return c.callPayload(ctx, method, map[string]any{"args": args}, out, progress)
}

// callPayload shares the authenticated Connection envelope across RPC dialects.
func (c *Client) callPayload(ctx context.Context, method string, payload any, out any, progress ...func(int64, int64)) error {
	rpcID := newID()
	body, err := json.Marshal(map[string]any{
		"type":    "client-request",
		"rpcId":   rpcID,
		"method":  method,
		"payload": payload,
	})
	if err != nil {
		return err
	}
	endpoint, err := url.JoinPath(c.Endpoint.Origin, "/api/", method)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	if len(progress) > 0 && progress[0] != nil {
		progress[0](0, int64(len(body)))
		req.Body = &requestProgressBody{ReadCloser: req.Body, total: int64(len(body)), changed: progress[0]}
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Endpoint.Cookie != "" {
		req.Header.Set("Cookie", c.Endpoint.Cookie)
	}
	res, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	// A 25 MiB UTF-8 HTML page can expand six-fold in JSON escaping.
	// Complete document methods have their own bounded wire caps.
	limit := int64(8 << 20)
	if method == "session/readVisualReply" {
		limit = 160 << 20
	} else if method == "officeToPdf/render" {
		// This existing method returns a complete multipart PDF, not byte ranges.
		// Match the 32 MiB PDF preview cap plus multipart metadata overhead.
		limit = 33 << 20
	} else if method == "session/list" || method == "session/projections" || method == "session/page" {
		limit = 128 << 20
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err == nil && int64(len(raw)) > limit {
		return fmt.Errorf("dsh %s: response exceeds bounded wire limit", method)
	}
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("dsh %s: HTTP %d: %s", method, res.StatusCode, clip(raw, 240))
	}
	ct := res.Header.Get("Content-Type")
	var value json.RawMessage
	if strings.HasPrefix(ct, "multipart/") {
		mr, err := parseMultipartResult(ct, raw, rpcID, method)
		if err != nil {
			return err
		}
		if mp, ok := out.(*MultipartResult); ok {
			*mp = *mr
			return nil
		}
		value = mr.Value
	} else {
		var parsed struct {
			Type   string `json:"type"`
			RPCID  string `json:"rpcId"`
			Result struct {
				OK    bool            `json:"ok"`
				Value json.RawMessage `json:"value"`
				Error *struct {
					Code    string          `json:"code"`
					Message string          `json:"message"`
					Details json.RawMessage `json:"details"`
				} `json:"error"`
			} `json:"result"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return fmt.Errorf("dsh %s: non-JSON HTTP %d: %s", method, res.StatusCode, clip(raw, 240))
		}
		if parsed.Type != "server-response" || parsed.RPCID != rpcID {
			return fmt.Errorf("dsh %s: invalid response envelope", method)
		}
		if !parsed.Result.OK {
			msg := method + " failed"
			code := ""
			var details json.RawMessage
			if parsed.Result.Error != nil {
				if parsed.Result.Error.Message != "" {
					msg = parsed.Result.Error.Message
				}
				code = parsed.Result.Error.Code
				details = parsed.Result.Error.Details
			}
			return &Error{Code: code, Message: msg, Details: details}
		}
		value = parsed.Result.Value
	}
	if out == nil || len(value) == 0 || string(value) == "null" {
		return nil
	}
	if err := json.Unmarshal(value, out); err != nil {
		return fmt.Errorf("dsh %s: decode value: %w", method, err)
	}
	return nil
}

// CallMultipart posts an RPC and returns the assembled multipart result (value +
// binary parts). args must be a JSON object; nil becomes {}.
func (c *Client) CallMultipart(ctx context.Context, method string, args any) (MultipartResult, error) {
	var mr MultipartResult
	if err := c.Call(ctx, method, args, &mr); err != nil {
		return MultipartResult{}, err
	}
	return mr, nil
}

func checkMethod(method string) error {
	ns, verb, ok := strings.Cut(method, "/")
	if !ok || ns == "" || verb == "" || strings.Contains(verb, "/") {
		return fmt.Errorf("dsh: method is not namespace/verb: %s", method)
	}
	return nil
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func clip(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= n {
		return s
	}
	return s[:n]
}
