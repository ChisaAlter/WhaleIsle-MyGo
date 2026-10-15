package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
)

// WhalePreferences is the dedicated dsh-whale catalog, not a personal Bot profile.
type WhalePreferences struct {
	ModelProvider        string       `json:"modelProvider"`
	ModelModel           string       `json:"modelModel"`
	ModelReasoningEffort string       `json:"modelReasoningEffort"`
	IMDefault            bool         `json:"imDefault"`
	Skills               []WhaleSkill `json:"skills"`
	HomeDir              string       `json:"homeDir"`
	SessionID            string       `json:"sessionId"`
}
type WhaleSkill struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}
type WhaleModelRoute struct {
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoningEffort"`
}

func (v WhalePreferences) Route() WhaleModelRoute {
	return WhaleModelRoute{v.ModelProvider, v.ModelModel, v.ModelReasoningEffort}
}
func (c *Client) WhalePreferences(ctx context.Context) (WhalePreferences, error) {
	raw, err := c.WhaleCatalog(ctx)
	var out WhalePreferences
	if err == nil {
		err = json.Unmarshal(raw, &out)
	}
	return out, err
}
func (c *Client) WhaleSaveModel(ctx context.Context, v WhaleModelRoute) error {
	var result json.RawMessage
	return c.Call(ctx, "dsh-whale", "settings/update", map[string]any{"model": v}, &result)
}
func (c *Client) WhaleSaveIMDefault(ctx context.Context, enabled bool) error {
	var result json.RawMessage
	return c.Call(ctx, "dsh-whale", "settings/update", map[string]any{"imDefault": enabled}, &result)
}
func (c *Client) WhaleToggleSkill(ctx context.Context, name string, enabled bool) error {
	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := c.Call(ctx, "dsh-whale", "skills/toggle", map[string]any{"name": name, "enabled": enabled}, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("Whale skill update failed: %s", result.Error)
	}
	return nil
}
