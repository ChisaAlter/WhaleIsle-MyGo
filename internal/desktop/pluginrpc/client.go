package pluginrpc

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Envelope request/response shapes match dsh-whale / dshbot control planes.
type clientRequest struct {
	Type    string `json:"type"`
	RPCID   string `json:"rpcId"`
	Method  string `json:"method"`
	Payload any    `json:"payload"`
}

type serverResponse struct {
	Type   string          `json:"type"`
	RPCID  string          `json:"rpcId"`
	Result json.RawMessage `json:"result"`
}

type resultEnvelope struct {
	OK    bool            `json:"ok"`
	Value json.RawMessage `json:"value"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Client posts plugin control-plane RPCs to a harness origin.
type Client struct {
	BaseURL    string // e.g. http://127.0.0.1:8787
	HTTPClient *http.Client
	Cookie     string // optional Cookie header (dsh-auth)
}

func (c *Client) http() *http.Client {
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// Call posts to {BaseURL}/{prefix}/{method} with a client-request body.
// prefix is "dsh-whale" or "dshbot" (no leading slash).
func (c *Client) Call(ctx context.Context, prefix, method string, payload any, out any) error {
	if c == nil {
		return fmt.Errorf("pluginrpc: nil client")
	}
	base, err := validateBaseURL(c.BaseURL)
	if err != nil {
		return err
	}
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	method = strings.Trim(strings.TrimSpace(method), "/")
	if prefix == "" || method == "" {
		return fmt.Errorf("pluginrpc: empty prefix/method")
	}
	u, err := url.JoinPath(base, prefix, method)
	if err != nil {
		return err
	}
	rpcID := newRPCID()
	body, err := json.Marshal(clientRequest{
		Type:    "client-request",
		RPCID:   rpcID,
		Method:  method,
		Payload: payload,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if ck := strings.TrimSpace(c.Cookie); ck != "" {
		req.Header.Set("Cookie", ck)
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pluginrpc: HTTP %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	var sr serverResponse
	if err := json.Unmarshal(raw, &sr); err != nil {
		return fmt.Errorf("pluginrpc: decode response: %w", err)
	}
	var re resultEnvelope
	if err := json.Unmarshal(sr.Result, &re); err != nil {
		return fmt.Errorf("pluginrpc: decode result: %w", err)
	}
	if !re.OK {
		if re.Error != nil {
			return fmt.Errorf("pluginrpc: %s: %s", re.Error.Code, re.Error.Message)
		}
		return fmt.Errorf("pluginrpc: call failed")
	}
	if out == nil || len(re.Value) == 0 || string(re.Value) == "null" {
		return nil
	}
	return json.Unmarshal(re.Value, out)
}

func validateBaseURL(raw string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	if base == "" {
		return "", fmt.Errorf("pluginrpc: base URL required")
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme != "http" && u.Scheme != "https" || u.User != nil || u.Hostname() == "" {
		return "", fmt.Errorf("pluginrpc: base URL must be http(s) without userinfo")
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host != "localhost" && net.ParseIP(host) == nil || (net.ParseIP(host) != nil && !net.ParseIP(host).IsLoopback()) {
		return "", fmt.Errorf("pluginrpc: base URL must target loopback")
	}
	return base, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func newRPCID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("rpc-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// WhaleEnsure calls dsh-whale/assistant/ensure.
func (c *Client) WhaleEnsure(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	err := c.Call(ctx, "dsh-whale", "assistant/ensure", map[string]any{}, &out)
	return out, err
}

// WhaleCatalog calls dsh-whale/catalog.
func (c *Client) WhaleCatalog(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	err := c.Call(ctx, "dsh-whale", "catalog", map[string]any{}, &out)
	return out, err
}

// WhalePetChat calls dsh-whale/pet/chat with text payload.
func (c *Client) WhalePetChat(ctx context.Context, text string) (json.RawMessage, error) {
	var out json.RawMessage
	err := c.Call(ctx, "dsh-whale", "pet/chat", map[string]any{"text": text}, &out)
	return out, err
}

// DshbotDescribe calls dshbot/describe.
func (c *Client) DshbotDescribe(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	err := c.Call(ctx, "dshbot", "describe", map[string]any{}, &out)
	return out, err
}

// DshbotDescribeBot calls dshbot/describe with an optional botId. The
// control-plane uses botId to scope capability discovery; an empty id keeps
// the full-catalog behavior of DshbotDescribe.
func (c *Client) DshbotDescribeBot(ctx context.Context, botID string) (json.RawMessage, error) {
	payload := map[string]any{}
	if id := strings.TrimSpace(botID); id != "" {
		payload["botId"] = id
	}
	var out json.RawMessage
	err := c.Call(ctx, "dshbot", "describe", payload, &out)
	return out, err
}

// DshbotCommand sends a raw dshbot control-plane command. The endpoint set
// belongs to the vendored dshbot control-plane; callers pass the exact
// payload defined by that source so unknown fields are not silently dropped.
func (c *Client) DshbotCommand(ctx context.Context, endpoint string, payload any, out any) error {
	return c.Call(ctx, "dshbot", endpoint, payload, out)
}

// DshbotCatalog is a read alias — catalog/items is a write; use describe.
func (c *Client) DshbotCatalog(ctx context.Context) (json.RawMessage, error) {
	return c.DshbotDescribe(ctx)
}

// WhalePetState calls dsh-whale/pet/state (history + session status).
func (c *Client) WhalePetState(ctx context.Context, includeCatalog bool) (json.RawMessage, error) {
	var out json.RawMessage
	err := c.Call(ctx, "dsh-whale", "pet/state", map[string]any{"includeCatalog": includeCatalog}, &out)
	return out, err
}

// WhalePetLook calls dsh-whale/pet/look (vision glance; image is raw base64).
// Uses a 130s HTTP timeout when the client has no custom HTTPClient.
func (c *Client) WhalePetLook(ctx context.Context, provider, model, imageBase64 string) (json.RawMessage, error) {
	provider = strings.TrimSpace(provider)
	model = strings.TrimSpace(model)
	imageBase64 = strings.TrimSpace(imageBase64)
	if provider == "" || model == "" {
		return nil, fmt.Errorf("pluginrpc: pet/look missing provider/model")
	}
	if imageBase64 == "" {
		return nil, fmt.Errorf("pluginrpc: pet/look missing image")
	}
	cli := c
	if c != nil && c.HTTPClient == nil {
		clone := *c
		clone.HTTPClient = &http.Client{Timeout: 130 * time.Second}
		cli = &clone
	}
	var out json.RawMessage
	err := cli.Call(ctx, "dsh-whale", "pet/look", map[string]any{
		"provider": provider,
		"model":    model,
		"image":    imageBase64,
	}, &out)
	return out, err
}

// IMChannelStatus posts Cordis-style {channel}/connection.status.
func (c *Client) IMChannelStatus(ctx context.Context, channel string) (json.RawMessage, error) {
	ch := strings.Trim(strings.TrimSpace(channel), "/")
	var out json.RawMessage
	err := c.Call(ctx, ch, "connection.status", map[string]any{}, &out)
	return out, err
}

// IMBotReconnect calls the dsh-im Cordis bot.reconnect endpoint. The
// vendored contract permits sendTest=true for a fixed connection-test message;
// it is deliberately separate from arbitrary IM draft sending.
func (c *Client) IMBotReconnect(ctx context.Context, channel, botID string, sendTest bool) (json.RawMessage, error) {
	channel = strings.Trim(strings.TrimSpace(channel), "/")
	botID = strings.TrimSpace(botID)
	if channel == "" || botID == "" {
		return nil, fmt.Errorf("pluginrpc: bot.reconnect missing channel/botId")
	}
	payload := map[string]any{
		"botId": botID,
	}
	// dsh-im only accepts sendTest when the fixed connection-test ping was
	// explicitly requested. Omitting false keeps vendor channels that reject
	// the field from seeing an unsupported option while preserving true.
	if sendTest {
		payload["sendTest"] = true
	}
	var out json.RawMessage
	err := c.Call(ctx, channel, "bot.reconnect", payload, &out)
	return out, err
}

// DshbotActivity calls dshbot/bot/activity.
func (c *Client) DshbotActivity(ctx context.Context, payload map[string]any) (json.RawMessage, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	var out json.RawMessage
	err := c.Call(ctx, "dshbot", "bot/activity", payload, &out)
	return out, err
}