package git

// The original DeepSeek JSON-completion contract shared by commit and PR copy.
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

func requestGeneratedJSON(ctx context.Context, system, user, key, base, failureMessage string, client *http.Client) ([]byte, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	failure := errors.New(failureMessage)
	if base == "" {
		base = "https://api.deepseek.com"
	}
	u, e := url.Parse(strings.TrimRight(base, "/"))
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("invalid generation endpoint; enter a message manually")
	}
	loopback := strings.EqualFold(u.Hostname(), "localhost")
	if ip := net.ParseIP(u.Hostname()); ip != nil {
		loopback = ip.IsLoopback()
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && loopback) {
		return nil, errors.New("generation requires HTTPS or a loopback endpoint")
	}
	if strings.HasSuffix(u.Path, "/v1") {
		u.Path += "/chat/completions"
	} else {
		u.Path += "/v1/chat/completions"
	}
	body, _ := json.Marshal(map[string]any{"model": "deepseek-chat", "temperature": 0.2, "messages": []map[string]string{{"role": "system", "content": system}, {"role": "user", "content": user}}})
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(body))
	if e != nil {
		return nil, e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	if client == nil {
		client = &http.Client{Timeout: 45 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("generation redirects are not allowed") }}
	}
	response, e := client.Do(req)
	if e != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, failure
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, failure
	}
	var payload struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if e = json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&payload); e != nil || len(payload.Choices) == 0 {
		return nil, failure
	}
	text := payload.Choices[0].Message.Content
	if match := generatedJSONFence.FindStringSubmatch(text); len(match) > 1 {
		text = match[1]
	}
	start, end := strings.IndexByte(text, '{'), strings.LastIndexByte(text, '}')
	if start < 0 || end <= start {
		return nil, failure
	}
	return []byte(text[start : end+1]), nil
}

var generatedJSONFence = regexp.MustCompile("(?is)```(?:json)?\\s*(.*?)```")
