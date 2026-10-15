package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
)

// Shared exact bot/update transport, restricted to the independently reviewed
// capability, memory-review and objective-configuration editors.
func (c *Client) saveDshbotProfileField(ctx context.Context, expected DshbotCapabilityStatus, field string, value json.RawMessage) (DshbotCapabilityReceipt, error) {
	return c.saveDshbotProfileFields(ctx, expected, map[string]json.RawMessage{field: value})
}
func (c *Client) saveDshbotProfileFields(ctx context.Context, expected DshbotCapabilityStatus, fields map[string]json.RawMessage) (DshbotCapabilityReceipt, error) {
	var out DshbotCapabilityReceipt
	switch {
	case len(fields) == 1 && fields["memoryReview"] != nil:
		if _, err := DshbotMemoryReviewEnabled(DshbotCapabilityStatus{Profile: fields}); err != nil {
			return out, err
		}
	case len(fields) == 1 && fields["capabilities"] != nil:
		if _, err := decodeCapabilityPolicies(fields["capabilities"]); err != nil {
			return out, err
		}
	case len(fields) == 2 && fields["objective"] != nil && fields["objectiveMaxRounds"] != nil:
		value, err := DshbotObjectiveConfigValue(DshbotCapabilityStatus{Profile: fields})
		if err != nil {
			return out, err
		}
		if err := ValidateDshbotObjectiveConfigCommand(DshbotObjectiveConfigCommand{Expected: expected, Desired: value}); err != nil {
			return out, err
		}
	default:
		return out, fmt.Errorf("unsupported profile editor fields")
	}
	if err := ValidateDshbotCapabilityProfile(expected); err != nil {
		return out, err
	}
	if expected.WriteError != "" {
		return out, fmt.Errorf("profile cannot be preserved")
	}
	payload := CloneDshbotCapabilities(expected).Profile
	for field, value := range fields {
		payload[field] = append(json.RawMessage(nil), value...)
	}
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
	if out.Status.WriteError != "" || out.Status.Bot.SessionID != expected.Bot.SessionID || out.Status.Revision <= expected.Revision {
		return out, fmt.Errorf("profile receipt differs; read back")
	}
	for field, value := range fields {
		if !sameCapabilityJSON(value, out.Status.Profile[field]) {
			return out, fmt.Errorf("profile receipt differs; read back: %s", field)
		}
	}
	for key, before := range expected.Profile {
		if _, edited := fields[key]; !edited && key != "updatedAt" && !sameCapabilityJSON(before, out.Status.Profile[key]) {
			return out, fmt.Errorf("profile field changed in receipt; read back: %s", key)
		}
	}
	return out, nil
}
