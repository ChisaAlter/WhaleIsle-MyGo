package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
)

// Shared exact bot/update transport. Only the two independently reviewed editors
// can select a field; never accept an arbitrary profile patch or invented RPC.
func (c *Client) saveDshbotProfileField(ctx context.Context, expected DshbotCapabilityStatus, field string, value json.RawMessage) (DshbotCapabilityReceipt, error) {
	var out DshbotCapabilityReceipt
	if field != "capabilities" && field != "memoryReview" {
		return out, fmt.Errorf("unsupported profile editor field")
	}
	if field == "memoryReview" {
		if _, err := DshbotMemoryReviewEnabled(DshbotCapabilityStatus{Profile: map[string]json.RawMessage{"memoryReview": value}}); err != nil {
			return out, err
		}
	} else if _, err := decodeCapabilityPolicies(value); err != nil {
		return out, err
	}
	if err := ValidateDshbotCapabilityProfile(expected); err != nil {
		return out, err
	}
	if expected.WriteError != "" {
		return out, fmt.Errorf("profile cannot be preserved")
	}
	payload := CloneDshbotCapabilities(expected).Profile
	payload[field] = append(json.RawMessage(nil), value...)
	payload["revision"], _ = json.Marshal(expected.Revision)
	var response struct {
		View     dshbotCapabilityView `json:"view"`
		Warnings []string             `json:"warnings"`
	}
	out.Attempted = true
	if err := c.Call(ctx, "dshbot", "bot/update", payload, &response); err != nil {
		return out, fmt.Errorf("profile save outcome uncertain; read back before another save")
	}
	out.Warnings = append([]string(nil), response.Warnings...)
	var err error
	out.Status, err = response.View.status(expected.Bot.ID, expected.Catalog)
	if err != nil {
		return out, err
	}
	if out.Status.Bot.SessionID != expected.Bot.SessionID || out.Status.Revision <= expected.Revision || !sameCapabilityJSON(value, out.Status.Profile[field]) {
		return out, fmt.Errorf("profile receipt differs; read back")
	}
	for key, before := range expected.Profile {
		if key != field && key != "updatedAt" && !sameCapabilityJSON(before, out.Status.Profile[key]) {
			return out, fmt.Errorf("profile field changed in receipt; read back: %s", key)
		}
	}
	return out, nil
}
