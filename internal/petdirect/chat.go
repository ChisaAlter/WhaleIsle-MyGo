// Package petdirect implements the original pet's explicitly sent, temporary
// OpenAI-compatible conversation. It has no Host/session API or retry path.
package petdirect

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
)

const Model = "deepseek-chat"
const HistoryTurns = 6
const ReplyLimit = 240
const Timeout = 30 * time.Second

var (
	ErrCredentials = errors.New("desktop API credentials unavailable")
	ErrEndpoint    = errors.New("desktop API endpoint is invalid or cleartext")
	ErrChanged     = errors.New("desktop API settings changed; reopen temporary chat")
	ErrInput       = errors.New("message must contain 1 to 2000 UTF-16 units")
	ErrRequest     = errors.New("API completion unconfirmed; request may have been received")
	ErrResponse    = errors.New("API returned no valid reply")
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type Config struct{ APIKey, BaseURL, Prompt string }

// Client resolves existing native credentials only when Send is explicitly
// called. A card binds to its first resolved route/credential/persona; changing
// those settings never sends its existing memory to a different destination.
// Neither raw HTTP errors, provider bodies nor credentials escape this type.
type Client struct {
	Load        func() (Config, error)
	HTTP        *http.Client // nil in production; fake RoundTripper in fixtures
	mu          sync.Mutex
	bound       bool
	fingerprint [32]byte
}

func endpoint(base string) (string, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		base = "https://api.deepseek.com"
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
		return "", ErrEndpoint
	}
	if u.Scheme == "http" {
		host := strings.ToLower(u.Hostname())
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			return "", ErrEndpoint
		}
	}
	return base + "/chat/completions", nil
}

func Text(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" || len(utf16.Encode([]rune(text))) > 2000 {
		return "", ErrInput
	}
	return text, nil
}

// truncate avoids cutting a surrogate pair while respecting the upstream
// JavaScript UTF-16 length cap.
func truncate(text string, limit int) string {
	units := 0
	for i, r := range text {
		n := 1
		if r > 0xffff {
			n = 2
		}
		if units+n > limit {
			return text[:i]
		}
		units += n
	}
	return text
}

func (c *Client) Send(ctx context.Context, text string, history []Message) (string, error) {
	text, err := Text(text)
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", ErrRequest
	}
	if c == nil || c.Load == nil {
		return "", ErrCredentials
	}
	cfg, err := c.Load()
	if err != nil || strings.TrimSpace(cfg.APIKey) == "" {
		return "", ErrCredentials
	}
	address, err := endpoint(cfg.BaseURL)
	if err != nil {
		return "", err
	}
	// JSON framing avoids ambiguity between concatenated credential/settings.
	identity, _ := json.Marshal([]string{address, cfg.APIKey, cfg.Prompt})
	fingerprint := sha256.Sum256(identity)
	c.mu.Lock()
	changed := c.bound && c.fingerprint != fingerprint
	if !changed {
		c.bound, c.fingerprint = true, fingerprint
	}
	c.mu.Unlock()
	if changed {
		return "", ErrChanged
	}
	if len(history) > HistoryTurns*2 || len(history)%2 != 0 {
		return "", ErrInput
	}
	messages := make([]Message, 0, len(history)+2)
	messages = append(messages, Message{"system", cfg.Prompt})
	for i, m := range history {
		role := "user"
		if i%2 != 0 {
			role = "assistant"
		}
		limit := 2000
		if role == "assistant" {
			limit = ReplyLimit
		}
		if m.Role != role || strings.TrimSpace(m.Content) == "" || len(utf16.Encode([]rune(m.Content))) > limit {
			return "", ErrInput
		}
		messages = append(messages, m)
	}
	messages = append(messages, Message{"user", text})
	body, _ := json.Marshal(struct {
		Model     string    `json:"model"`
		Messages  []Message `json:"messages"`
		MaxTokens int       `json:"max_tokens"`
	}{Model, messages, 360})
	requestCtx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return "", ErrEndpoint
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	client := http.Client{Timeout: Timeout}
	if c.HTTP != nil {
		client = *c.HTTP
	}
	client.Timeout = Timeout
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	// No retry, streaming, cookie jar or Host cookie is used on this route.
	client.Jar = nil
	res, err := client.Do(req)
	if err != nil {
		return "", ErrRequest
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", ErrRequest
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return "", ErrResponse
	}
	var value struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(raw, &value) != nil || len(value.Choices) == 0 {
		return "", ErrResponse
	}
	reply := strings.TrimSpace(value.Choices[0].Message.Content)
	if reply == "" {
		return "", ErrResponse
	}
	if requestCtx.Err() != nil {
		return "", ErrRequest
	}
	return truncate(reply, ReplyLimit), nil
}
