package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type DshbotMemoryReviewCommand struct {
	Expected DshbotCapabilityStatus
	Enabled  bool
}

func DshbotMemoryReviewEnabled(s DshbotCapabilityStatus) (bool, error) {
	raw, ok := s.Profile["memoryReview"]
	value := strings.TrimSpace(string(raw))
	if !ok || value != "true" && value != "false" {
		return false, fmt.Errorf("memoryReview is unavailable; do not guess a default")
	}
	return value == "true", nil
}
func ValidateDshbotMemoryReviewCommand(cmd DshbotMemoryReviewCommand) error {
	if err := ValidateDshbotCapabilityProfile(cmd.Expected); err != nil {
		return err
	}
	if cmd.Expected.WriteError != "" {
		return fmt.Errorf("profile cannot be preserved")
	}
	current, err := DshbotMemoryReviewEnabled(cmd.Expected)
	if err != nil {
		return err
	}
	if current == cmd.Enabled {
		return fmt.Errorf("memoryReview is unchanged")
	}
	return nil
}
func (c *Client) DshbotSaveMemoryReview(ctx context.Context, cmd DshbotMemoryReviewCommand) (DshbotCapabilityReceipt, error) {
	if err := ValidateDshbotMemoryReviewCommand(cmd); err != nil {
		return DshbotCapabilityReceipt{}, err
	}
	value, _ := json.Marshal(cmd.Enabled)
	return c.saveDshbotProfileField(ctx, cmd.Expected, "memoryReview", value)
}
