package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf16"
)

// WhaleSettings is dsh-whale/catalog. This plugin has no revision/CAS field:
// do not send the unrelated dshbot catalog revision or task namespace.
type WhaleSettings struct {
	Name        string `json:"name"`
	UserTitle   string `json:"userTitle"`
	PersonaText string `json:"personaText"`
	Memory      string `json:"memory"`
	SessionID   string `json:"sessionId"`
	HomeDir     string `json:"homeDir"`
}

func (s WhaleSettings) SameIdentity(other WhaleSettings) bool {
	return s.HomeDir == other.HomeDir && s.SessionID == other.SessionID
}
func WhaleTextLength(text string) int { return len(utf16.Encode([]rune(text))) }
func (c *Client) WhaleSettings(ctx context.Context) (WhaleSettings, error) {
	raw, err := c.WhaleCatalog(ctx)
	if err != nil {
		return WhaleSettings{}, err
	}
	var shape map[string]json.RawMessage
	if err = json.Unmarshal(raw, &shape); err != nil {
		return WhaleSettings{}, err
	}
	if _, ok := shape["memory"]; !ok {
		return WhaleSettings{}, fmt.Errorf("Whale catalog has no memory field")
	}
	var out WhaleSettings
	if err = json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	if strings.TrimSpace(out.HomeDir) == "" {
		return out, fmt.Errorf("Whale home is unavailable")
	}
	return out, nil
}
func (c *Client) WhaleReplaceMemory(ctx context.Context, text string) error {
	if WhaleTextLength(text) > 40000 {
		return fmt.Errorf("长期记忆超过 40000 字符限制 / Memory exceeds 40000 UTF-16 units")
	}
	var receipt struct {
		OK    bool   `json:"ok"`
		Chars *int   `json:"chars"`
		Error string `json:"error"`
	}
	if err := c.Call(ctx, "dsh-whale", "memory/replace", map[string]any{"text": text}, &receipt); err != nil {
		return err
	}
	if !receipt.OK || receipt.Chars == nil || *receipt.Chars != WhaleTextLength(text) {
		return fmt.Errorf("Whale memory write has no matching success receipt; refresh before retrying")
	}
	return nil
}
func (c *Client) WhaleClearMemory(ctx context.Context) error {
	var receipt struct {
		OK bool `json:"ok"`
	}
	if err := c.Call(ctx, "dsh-whale", "memory/clear", map[string]any{}, &receipt); err != nil {
		return err
	}
	if !receipt.OK {
		return fmt.Errorf("Whale clear has no success receipt; refresh before retrying")
	}
	return nil
}

// Pointer fields ensure this operation never overwrites unrelated settings.
type WhaleSettingsPatch struct {
	Name        *string `json:"name,omitempty"`
	UserTitle   *string `json:"userTitle,omitempty"`
	PersonaText *string `json:"personaText,omitempty"`
}

func ValidateWhaleSettingsPatch(p WhaleSettingsPatch) error {
	if p.Name == nil && p.UserTitle == nil && p.PersonaText == nil {
		return fmt.Errorf("empty Whale settings patch")
	}
	for _, f := range []struct {
		v    *string
		max  int
		name string
	}{{p.Name, 64, "name"}, {p.UserTitle, 64, "userTitle"}, {p.PersonaText, 4000, "personaText"}} {
		if f.v != nil && WhaleTextLength(*f.v) > f.max {
			return fmt.Errorf("%s exceeds %d UTF-16 units", f.name, f.max)
		}
	}
	return nil
}
func (c *Client) WhaleUpdateSettings(ctx context.Context, p WhaleSettingsPatch) (WhaleSettings, error) {
	if err := ValidateWhaleSettingsPatch(p); err != nil {
		return WhaleSettings{}, err
	}
	var raw json.RawMessage
	if err := c.Call(ctx, "dsh-whale", "settings/update", p, &raw); err != nil {
		return WhaleSettings{}, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return WhaleSettings{}, err
	}
	for key, enabled := range map[string]bool{"name": p.Name != nil, "userTitle": p.UserTitle != nil, "personaText": p.PersonaText != nil} {
		if enabled {
			if _, ok := fields[key]; !ok {
				return WhaleSettings{}, fmt.Errorf("Whale settings receipt missing %s; refresh before retrying", key)
			}
		}
	}
	var out WhaleSettings
	err := json.Unmarshal(raw, &out)
	return out, err
}
