package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Exact pinned Host avatar vocabulary. Native writes only explicit blob choices;
// image import/cropping remains in the original Web editor.
func DshbotAvatarShapes() []string {
	return []string{"circle", "soft", "square", "pill", "triangle", "hex", "cloud", "drop"}
}
func DshbotAvatarColors() []string {
	return []string{"ink", "brown", "red", "orange", "yellow", "green", "teal", "blue", "purple", "pink", "grey"}
}

type DshbotAvatarValues struct {
	Kind    string `json:"kind"`
	Shape   string `json:"shape,omitempty"`
	Color   string `json:"color,omitempty"`
	DataURL string `json:"dataUrl,omitempty"`
	Crop    string `json:"crop,omitempty"`
}
type DshbotAvatarCommand struct {
	Expected DshbotCapabilityStatus
	Values   DshbotAvatarValues
}

func CloneDshbotAvatarCommand(c DshbotAvatarCommand) DshbotAvatarCommand {
	c.Expected = CloneDshbotCapabilities(c.Expected)
	return c
}
func DshbotAvatarValue(s DshbotCapabilityStatus) (DshbotAvatarValues, error) {
	var v DshbotAvatarValues
	p, err := capabilityObject(s.Profile["avatar"], "kind", "shape", "color", "dataUrl", "crop")
	if err != nil {
		return v, err
	}
	for _, raw := range p {
		var text string
		if string(raw) == "null" || json.Unmarshal(raw, &text) != nil {
			return v, fmt.Errorf("avatar fields must be strings")
		}
	}
	if err := json.Unmarshal(s.Profile["avatar"], &v); err != nil {
		return v, err
	}
	return ParseDshbotAvatar(v)
}

// No fallback, trimming, hashing, remote fetch or coercion: an image never
// becomes a blob without an explicit user choice.
func ParseDshbotAvatar(v DshbotAvatarValues) (DshbotAvatarValues, error) {
	switch v.Kind {
	case "blob":
		if slices.Contains(DshbotAvatarShapes(), v.Shape) && slices.Contains(DshbotAvatarColors(), v.Color) && v.DataURL == "" && v.Crop == "" {
			return v, nil
		}
	case "image":
		url := strings.ToLower(v.DataURL)
		if v.Shape == "" && v.Color == "" && (v.Crop == "circle" || v.Crop == "square") && WhaleTextLength(v.DataURL) <= 120000 && (strings.HasPrefix(url, "data:image/jpeg;base64,") || strings.HasPrefix(url, "data:image/jpg;base64,") || strings.HasPrefix(url, "data:image/png;base64,")) {
			return v, nil
		}
	}
	return v, fmt.Errorf("avatar is not an exact supported record; use Bot Web")
}
func ValidateDshbotAvatarCommand(c DshbotAvatarCommand) error {
	if err := ValidateDshbotCapabilityProfile(c.Expected); err != nil {
		return err
	}
	if c.Expected.WriteError != "" {
		return fmt.Errorf("profile cannot be preserved; use Bot Web")
	}
	before, err := DshbotAvatarValue(c.Expected)
	if err != nil {
		return err
	}
	next, err := ParseDshbotAvatar(c.Values)
	if err != nil {
		return err
	}
	if next.Kind != "blob" {
		return fmt.Errorf("native avatar save requires an explicit blob choice; image editing remains in Bot Web")
	}
	if next == before {
		return fmt.Errorf("avatar unchanged")
	}
	return nil
}
func DshbotAvatarApplied(c DshbotAvatarCommand, s DshbotCapabilityStatus) error {
	if err := ValidateDshbotCapabilityProfile(s); err != nil {
		return err
	}
	after, err := DshbotAvatarValue(s)
	if err != nil || after != c.Values || s.WriteError != "" || s.Revision <= c.Expected.Revision || s.Bot.ID != c.Expected.Bot.ID || s.Bot.SessionID != c.Expected.Bot.SessionID || len(s.Profile) != len(c.Expected.Profile) {
		return fmt.Errorf("complete reviewed avatar profile not confirmed; read back")
	}
	for key, before := range c.Expected.Profile {
		if key != "avatar" && key != "updatedAt" && !sameCapabilityJSON(before, s.Profile[key]) {
			return fmt.Errorf("unrelated profile field changed: %s", key)
		}
	}
	return nil
}

// next from bot/update precedes AvatarSchema defaults while view can include
// empty shape/color/dataUrl/crop defaults. Only those proven avatar differences
// are accepted. Unknown nested keys and every unrelated item difference fail.
func dshbotAvatarItemReceipt(raw json.RawMessage, s DshbotCapabilityStatus) bool {
	var item map[string]json.RawMessage
	if json.Unmarshal(raw, &item) != nil || len(item) != len(s.Profile) {
		return false
	}
	a, err := DshbotAvatarValue(DshbotCapabilityStatus{Profile: item})
	if err != nil {
		return false
	}
	b, err := DshbotAvatarValue(s)
	if err != nil || a != b {
		return false
	}
	for key, value := range s.Profile {
		if key != "avatar" && !sameCapabilityJSON(value, item[key]) {
			return false
		}
	}
	return true
}
func (c *Client) DshbotSaveAvatar(ctx context.Context, cmd DshbotAvatarCommand) (DshbotCapabilityReceipt, error) {
	if err := ValidateDshbotAvatarCommand(cmd); err != nil {
		return DshbotCapabilityReceipt{}, err
	}
	value, _ := json.Marshal(cmd.Values)
	return c.saveDshbotProfileField(ctx, cmd.Expected, "avatar", value)
}
