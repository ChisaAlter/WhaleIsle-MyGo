package dsh

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"golang.org/x/net/html"
)

const DefaultStopSequenceMS int64 = 500
const MaximumStopSequenceMS int64 = 2147483646

// ReadStopSequenceInterval reads the existing authenticated Host boot document.
// It interprets only the shortcuts plugin's JSON assignment, never JavaScript.
func (c *Client) ReadStopSequenceInterval(ctx context.Context) (int64, error) {
	body, err := c.readBootDocument(ctx)
	if err != nil {
		return 0, err
	}
	return parseStopSequenceInterval(string(body))
}

func parseStopSequenceInterval(document string) (int64, error) {
	tokenizer := html.NewTokenizer(strings.NewReader(document))
	interval := DefaultStopSequenceMS
	script := false
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			if tokenizer.Err() != io.EOF {
				return 0, errors.New("invalid Host boot document")
			}
			return interval, nil
		case html.StartTagToken:
			token := tokenizer.Token()
			script = token.Data == "script"
			for _, attr := range token.Attr {
				if attr.Key == "src" {
					script = false
				}
			}
		case html.EndTagToken:
			script = false
		case html.TextToken:
			if !script {
				continue
			}
			text := strings.TrimSpace(string(tokenizer.Text()))
			const prefix = `globalThis["__DSH_SHORTCUTS_CONFIG__"]`
			if !strings.HasPrefix(text, prefix) {
				continue
			}
			value := strings.TrimSpace(strings.TrimPrefix(text, prefix))
			if !strings.HasPrefix(value, "=") {
				return 0, errors.New("invalid Host shortcut configuration")
			}
			value = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value[1:]), ";"))
			var config map[string]json.RawMessage
			if json.Unmarshal([]byte(value), &config) != nil || config == nil {
				return 0, errors.New("invalid Host shortcut configuration")
			}
			raw, ok := config["stopSequenceMs"]
			if !ok {
				interval = DefaultStopSequenceMS
				continue
			}
			var ms int64
			if json.Unmarshal(raw, &ms) != nil || ms < 1 || ms > MaximumStopSequenceMS {
				return 0, errors.New("invalid Host stopSequenceMs")
			}
			interval = ms
		}
	}
}
