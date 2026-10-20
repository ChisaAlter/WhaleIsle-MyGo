// Package whale is the desktop client for the dsh-whale plugin's control
// plane: the settings catalog the Whale page renders and the mutations that
// page performs (settings, long-term memory, skills).
//
// The Host stores Whale's settings in the `dsh-whale` namespace and reports
// the model as FLAT fields (modelProvider/modelModel/modelReasoningEffort);
// only `settings/update` accepts the model as a nested object. Both shapes are
// mirrored here because the plugin's request sanitiser reads the nested one
// while the catalog spreads the stored settings.
package whale

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
)

// pluginID is the RPC prefix the Host mounts for this plugin.
const pluginID = "dsh-whale"

// memoryMaxUTF16 is the plugin's MEMORY_MAX_CHARS, counted in JavaScript
// String.length units (UTF-16 code units), not UTF-8 bytes.
const memoryMaxUTF16 = 40000

// skillNamePattern is the plugin's own skill-name gate.
var skillNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Skill is one row of the plugin's skills list. The plugin reports only the
// directory name and whether it currently lives in the enabled tree.
type Skill struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// Catalog is the `catalog` result: the stored settings snapshot plus the
// resources the page lists (personality presets, skills, memory, home).
type Catalog struct {
	Name                 string   `json:"name"`
	Personality          string   `json:"personality"`
	UserTitle            string   `json:"userTitle"`
	PersonaText          string   `json:"personaText"`
	ModelProvider        string   `json:"modelProvider"`
	ModelModel           string   `json:"modelModel"`
	ModelReasoningEffort string   `json:"modelReasoningEffort"`
	IMDefault            bool     `json:"imDefault"`
	SessionID            string   `json:"sessionId"`
	PersonalityOptions   []string `json:"personalityOptions"`
	Skills               []Skill  `json:"skills"`
	Memory               string   `json:"memory"`
	HomeDir              string   `json:"homeDir"`
}

// Patch is one `settings/update` request. A nil field is left untouched, so a
// form can save one field without rewriting the rest.
type Patch struct {
	Name        *string
	Personality *string
	UserTitle   *string
	PersonaText *string
	IMDefault   *bool
	Provider    *string
	Model       *string
	Effort      *string
}

// Empty reports whether the patch would send nothing.
func (p Patch) Empty() bool {
	return p.Name == nil && p.Personality == nil && p.UserTitle == nil && p.PersonaText == nil &&
		p.IMDefault == nil && p.Provider == nil && p.Model == nil && p.Effort == nil
}

// payload is the wire shape: top-level settings fields plus the plugin's nested
// `model` object, which its sanitiser is the only shape it reads.
func (p Patch) payload() map[string]any {
	out := map[string]any{}
	if p.Name != nil {
		out["name"] = *p.Name
	}
	if p.Personality != nil {
		out["personality"] = *p.Personality
	}
	if p.UserTitle != nil {
		out["userTitle"] = *p.UserTitle
	}
	if p.PersonaText != nil {
		out["personaText"] = *p.PersonaText
	}
	if p.IMDefault != nil {
		out["imDefault"] = *p.IMDefault
	}
	model := map[string]any{}
	if p.Provider != nil {
		model["provider"] = *p.Provider
	}
	if p.Model != nil {
		model["model"] = *p.Model
	}
	if p.Effort != nil {
		model["reasoningEffort"] = *p.Effort
	}
	if len(model) > 0 {
		out["model"] = model
	}
	return out
}

// Client calls the dsh-whale control plane on a Harness origin.
type Client struct {
	plugin *pluginrpc.Client
}

// New validates the Host origin and returns a client for it. The origin rule
// is the one internal/desktop/pet/quickchat_protocol.go uses for the same
// plugin: http(s), a host, no userinfo/path/query/fragment.
func New(origin, cookie string) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || u.Host == "" || u.User != nil || u.Opaque != "" ||
		(u.Scheme != "http" && u.Scheme != "https") || (u.Path != "" && u.Path != "/") ||
		u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" {
		return nil, errors.New("invalid Host origin")
	}
	return &Client{plugin: &pluginrpc.Client{
		BaseURL: origin,
		Cookie:  cookie,
		HTTPClient: &http.Client{
			Timeout:       20 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}}, nil
}

// Catalog reads the current settings snapshot.
func (c *Client) Catalog(ctx context.Context) (Catalog, error) {
	var out Catalog
	if c == nil || c.plugin == nil {
		return out, errors.New("whale: nil client")
	}
	if err := c.plugin.Call(ctx, pluginID, "catalog", map[string]any{}, &out); err != nil {
		return Catalog{}, err
	}
	// The plugin always reports these keys; an empty list is [] not null.
	if out.PersonalityOptions == nil {
		out.PersonalityOptions = []string{}
	}
	if out.Skills == nil {
		out.Skills = []Skill{}
	}
	return out, nil
}

// Update patches the settings and returns the snapshot the plugin stored.
func (c *Client) Update(ctx context.Context, patch Patch) (Catalog, error) {
	var out Catalog
	if c == nil || c.plugin == nil {
		return out, errors.New("whale: nil client")
	}
	if patch.Empty() {
		return out, errors.New("whale: empty settings patch")
	}
	if err := c.plugin.Call(ctx, pluginID, "settings/update", patch.payload(), &out); err != nil {
		return Catalog{}, err
	}
	return out, nil
}

// ReplaceMemory writes the long-term memory and returns the stored length in
// UTF-16 units, as the plugin counts it.
func (c *Client) ReplaceMemory(ctx context.Context, text string) (int, error) {
	if c == nil || c.plugin == nil {
		return 0, errors.New("whale: nil client")
	}
	if n := pluginrpc.WhaleTextLength(text); n > memoryMaxUTF16 {
		return 0, fmt.Errorf("whale: memory exceeds %d characters", memoryMaxUTF16)
	}
	var receipt struct {
		OK    bool   `json:"ok"`
		Chars *int   `json:"chars"`
		Error string `json:"error"`
	}
	if err := c.plugin.Call(ctx, pluginID, "memory/replace", map[string]any{"text": text}, &receipt); err != nil {
		return 0, err
	}
	if !receipt.OK {
		return 0, errors.New(receiptError(receipt.Error, "memory write rejected"))
	}
	if receipt.Chars == nil {
		return 0, errors.New("whale: memory write returned no length")
	}
	return *receipt.Chars, nil
}

// ClearMemory restores the plugin's empty memory note.
func (c *Client) ClearMemory(ctx context.Context) error {
	if c == nil || c.plugin == nil {
		return errors.New("whale: nil client")
	}
	var receipt struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := c.plugin.Call(ctx, pluginID, "memory/clear", map[string]any{}, &receipt); err != nil {
		return err
	}
	if !receipt.OK {
		return errors.New(receiptError(receipt.Error, "memory clear rejected"))
	}
	return nil
}

// ToggleSkill moves one skill directory between the enabled and disabled trees.
func (c *Client) ToggleSkill(ctx context.Context, name string, enabled bool) error {
	if c == nil || c.plugin == nil {
		return errors.New("whale: nil client")
	}
	name = strings.TrimSpace(name)
	if !skillNamePattern.MatchString(name) {
		return errors.New("whale: invalid skill name")
	}
	var receipt struct {
		OK      bool   `json:"ok"`
		Changed bool   `json:"changed"`
		Error   string `json:"error"`
	}
	if err := c.plugin.Call(ctx, pluginID, "skills/toggle", map[string]any{"name": name, "enabled": enabled}, &receipt); err != nil {
		return err
	}
	if !receipt.OK {
		return errors.New(receiptError(receipt.Error, "skill toggle rejected"))
	}
	return nil
}

// receiptError keeps the plugin's own reason when it sends one.
func receiptError(reason, fallback string) string {
	if reason = strings.TrimSpace(reason); reason != "" {
		return "whale: " + reason
	}
	return "whale: " + fallback
}

// DecodeCatalog reads a raw catalog payload, for callers that already hold the
// RPC envelope (the plugin also returns the snapshot from settings/update).
func DecodeCatalog(raw json.RawMessage) (Catalog, error) {
	var out Catalog
	err := json.Unmarshal(raw, &out)
	return out, err
}
