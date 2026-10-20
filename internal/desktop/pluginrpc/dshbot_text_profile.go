package pluginrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"
	"whaleisle/internal/store/storelocale"
)

// These are the three actual text identity/persona fields from the pinned
// original profile editor. IDs, model, workspace and other settings are excluded.
type DshbotTextProfileValues struct{ Name, Title, Description string }
type DshbotTextProfileCommand struct {
	Expected DshbotCapabilityStatus
	Values   DshbotTextProfileValues
}

func CloneDshbotTextProfileCommand(c DshbotTextProfileCommand) DshbotTextProfileCommand {
	c.Expected = CloneDshbotCapabilities(c.Expected)
	return c
}
func DshbotTextProfileValue(s DshbotCapabilityStatus) (DshbotTextProfileValues, error) {
	var v DshbotTextProfileValues
	var err error
	if v.Name, err = capabilityText(s.Profile, "name", 120, true); err != nil {
		return v, err
	}
	if v.Title, err = capabilityText(s.Profile, "title", 120, false); err != nil {
		return v, err
	}
	v.Description, err = capabilityText(s.Profile, "description", 12000, false)
	return v, err
}
func ParseDshbotTextProfile(v DshbotTextProfileValues) (DshbotTextProfileValues, error) {
	if !utf8.ValidString(v.Name) || !utf8.ValidString(v.Title) || !utf8.ValidString(v.Description) {
		return DshbotTextProfileValues{}, errors.New(storelocale.T(storelocale.KeyDesktopPluginrpcInvalidUnicode))
	}
	v.Name = trimBotMemory(v.Name)
	v.Title = trimBotMemory(v.Title)
	v.Description = trimBotMemory(v.Description)
	if v.Name == "" || WhaleTextLength(v.Name) > 120 {
		return v, errors.New(storelocale.T(storelocale.KeyDesktopPluginrpcNameIsRequiredMaximum120Utf16Unit))
	}
	if WhaleTextLength(v.Title) > 120 || WhaleTextLength(v.Description) > 12000 {
		return v, errors.New(storelocale.T(storelocale.KeyDesktopPluginrpcTitle120Description12000Utf16Units))
	}
	return v, nil
}
func ValidateDshbotTextProfileCommand(c DshbotTextProfileCommand) error {
	if err := ValidateDshbotCapabilityProfile(c.Expected); err != nil {
		return err
	}
	if c.Expected.WriteError != "" {
		return fmt.Errorf("profile cannot be preserved; use existing Bot Web")
	}
	before, err := DshbotTextProfileValue(c.Expected)
	if err != nil {
		return err
	}
	next, err := ParseDshbotTextProfile(c.Values)
	if err != nil {
		return err
	}
	if next != c.Values {
		return fmt.Errorf("review trimmed values before saving")
	}
	if before == next {
		return fmt.Errorf("text profile is unchanged")
	}
	return nil
}
func DshbotTextProfileApplied(c DshbotTextProfileCommand, s DshbotCapabilityStatus) error {
	if err := ValidateDshbotCapabilityProfile(s); err != nil {
		return err
	}
	after, err := DshbotTextProfileValue(s)
	if err != nil || after != c.Values || s.WriteError != "" || s.Revision <= c.Expected.Revision || s.Bot.ID != c.Expected.Bot.ID || s.Bot.SessionID != c.Expected.Bot.SessionID || len(s.Profile) != len(c.Expected.Profile) {
		return fmt.Errorf("complete reviewed text profile is not confirmed; read back")
	}
	for key, before := range c.Expected.Profile {
		if key == "updatedAt" || key == "name" || key == "title" || key == "description" {
			continue
		}
		if !sameCapabilityJSON(before, s.Profile[key]) {
			return fmt.Errorf("unrelated profile field changed: %s", key)
		}
	}
	return nil
}
func (c *Client) DshbotSaveTextProfile(ctx context.Context, cmd DshbotTextProfileCommand) (DshbotCapabilityReceipt, error) {
	if err := ValidateDshbotTextProfileCommand(cmd); err != nil {
		return DshbotCapabilityReceipt{}, err
	}
	name, _ := json.Marshal(cmd.Values.Name)
	title, _ := json.Marshal(cmd.Values.Title)
	description, _ := json.Marshal(cmd.Values.Description)
	return c.saveDshbotProfileFields(ctx, cmd.Expected, map[string]json.RawMessage{"name": name, "title": title, "description": description})
}
