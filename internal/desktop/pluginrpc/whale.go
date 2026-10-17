package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// WhaleEnsureResult is the value of dsh-whale/assistant/ensure.
type WhaleEnsureResult struct {
	OK        bool   `json:"ok"`
	SessionID string `json:"sessionId"`
	Created   bool   `json:"created"`
	Error     string `json:"error"`
}

// WhalePetChatResult is the value of dsh-whale/pet/chat.
type WhalePetChatResult struct {
	OK    bool   `json:"ok"`
	Reply string `json:"reply"`
	Error string `json:"error"`
}

// WhaleEnsureTyped calls assistant/ensure and unwraps the nested ok envelope.
func (c *Client) WhaleEnsureTyped(ctx context.Context) (WhaleEnsureResult, error) {
	raw, err := c.WhaleEnsure(ctx)
	if err != nil {
		return WhaleEnsureResult{}, err
	}
	var out WhaleEnsureResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return WhaleEnsureResult{}, fmt.Errorf("pluginrpc: decode ensure: %w", err)
	}
	if !out.OK {
		errMsg := strings.TrimSpace(out.Error)
		if errMsg == "" {
			errMsg = "ensure failed"
		}
		return out, fmt.Errorf("pluginrpc: whale ensure: %s", errMsg)
	}
	out.SessionID = strings.TrimSpace(out.SessionID)
	if out.SessionID == "" {
		return out, fmt.Errorf("pluginrpc: whale ensure: empty sessionId")
	}
	return out, nil
}

// WhalePetChatTyped calls pet/chat (long-running turn; uses a 130s HTTP
// timeout when the client has no custom HTTPClient).
func (c *Client) WhalePetChatTyped(ctx context.Context, text string) (WhalePetChatResult, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return WhalePetChatResult{}, fmt.Errorf("pluginrpc: empty pet/chat text")
	}
	cli := c
	if c != nil && c.HTTPClient == nil {
		clone := *c
		clone.HTTPClient = &http.Client{Timeout: 130 * time.Second}
		cli = &clone
	}
	raw, err := cli.WhalePetChat(ctx, text)
	if err != nil {
		return WhalePetChatResult{}, err
	}
	var out WhalePetChatResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return WhalePetChatResult{}, fmt.Errorf("pluginrpc: decode pet/chat: %w", err)
	}
	if !out.OK {
		errMsg := strings.TrimSpace(out.Error)
		if errMsg == "" {
			errMsg = "pet/chat failed"
		}
		return out, fmt.Errorf("pluginrpc: whale pet/chat: %s", errMsg)
	}
	return out, nil
}

// WhalePetStateModel is the nested model selection from pet/state.
type WhalePetStateModel struct {
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoningEffort"`
}

// WhalePetStateResult is a partial decode of dsh-whale/pet/state.
type WhalePetStateResult struct {
	OK        bool                `json:"ok"`
	SessionID string              `json:"sessionId"`
	Model     *WhalePetStateModel `json:"model"`
	Error     string              `json:"error"`
}

// WhalePetStateTyped calls pet/state and returns session + model selection.
func (c *Client) WhalePetStateTyped(ctx context.Context, includeCatalog bool) (WhalePetStateResult, error) {
	raw, err := c.WhalePetState(ctx, includeCatalog)
	if err != nil {
		return WhalePetStateResult{}, err
	}
	var out WhalePetStateResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return WhalePetStateResult{}, fmt.Errorf("pluginrpc: decode pet/state: %w", err)
	}
	if !out.OK {
		errMsg := strings.TrimSpace(out.Error)
		if errMsg == "" {
			errMsg = "pet/state failed"
		}
		return out, fmt.Errorf("pluginrpc: whale pet/state: %s", errMsg)
	}
	return out, nil
}
