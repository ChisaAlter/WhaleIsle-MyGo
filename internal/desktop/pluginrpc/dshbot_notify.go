package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type DshbotNotifyCommand struct {
	Expected DshbotCapabilityStatus
	Enabled  bool
}

func CloneDshbotNotifyCommand(c DshbotNotifyCommand) DshbotNotifyCommand {
	c.Expected = CloneDshbotCapabilities(c.Expected)
	return c
}
func DshbotNotifyEnabled(s DshbotCapabilityStatus) (bool, error) {
	raw, ok := s.Profile["notify"]
	value := strings.TrimSpace(string(raw))
	if !ok || value != "true" && value != "false" {
		return false, fmt.Errorf("notify is unavailable; do not guess a default")
	}
	return value == "true", nil
}
func ValidateDshbotNotifyCommand(c DshbotNotifyCommand) error {
	if err := ValidateDshbotCapabilityProfile(c.Expected); err != nil {
		return err
	}
	if c.Expected.WriteError != "" {
		return fmt.Errorf("profile cannot be preserved; use Bot Web")
	}
	before, err := DshbotNotifyEnabled(c.Expected)
	if err != nil {
		return err
	}
	if before == c.Enabled {
		return fmt.Errorf("notify unchanged")
	}
	return nil
}
func DshbotNotifyApplied(c DshbotNotifyCommand, s DshbotCapabilityStatus) error {
	if err := ValidateDshbotCapabilityProfile(s); err != nil {
		return err
	}
	after, err := DshbotNotifyEnabled(s)
	if err != nil || after != c.Enabled || s.WriteError != "" || s.Revision <= c.Expected.Revision || s.Bot.ID != c.Expected.Bot.ID || s.Bot.SessionID != c.Expected.Bot.SessionID || len(s.Profile) != len(c.Expected.Profile) {
		return fmt.Errorf("complete reviewed notify profile not confirmed; read back")
	}
	for key, before := range c.Expected.Profile {
		if key != "notify" && key != "updatedAt" && !sameCapabilityJSON(before, s.Profile[key]) {
			return fmt.Errorf("unrelated profile field changed: %s", key)
		}
	}
	return nil
}
func (c *Client) DshbotSaveNotify(ctx context.Context, cmd DshbotNotifyCommand) (DshbotCapabilityReceipt, error) {
	if err := ValidateDshbotNotifyCommand(cmd); err != nil {
		return DshbotCapabilityReceipt{}, err
	}
	value, _ := json.Marshal(cmd.Enabled)
	return c.saveDshbotProfileField(ctx, cmd.Expected, "notify", value)
}
