package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
)

// Avatar edits also use this restricted full-profile transport.
// Shared exact bot/update transport, restricted to the independently reviewed
// capability, memory-review, objective-configuration, route and text-profile editors.
func (c *Client) saveDshbotProfileField(ctx context.Context, expected DshbotCapabilityStatus, field string, value json.RawMessage) (DshbotCapabilityReceipt, error) {
	return c.saveDshbotProfileFields(ctx, expected, map[string]json.RawMessage{field: value})
}
func (c *Client) saveDshbotProfileFields(ctx context.Context, expected DshbotCapabilityStatus, fields map[string]json.RawMessage, route ...DshbotRouteCommand) (DshbotCapabilityReceipt, error) {
	var out DshbotCapabilityReceipt
	if len(route) > 0 && (len(route) != 1 || len(fields) != 2 || fields["model"] == nil || fields["workspaceId"] == nil) {
		return out, fmt.Errorf("route arguments require exactly model and workspaceId")
	}
	switch {
	case len(fields) == 1 && fields["allowedSenderIds"] != nil:
		if err := validateDshbotSenderField(expected, fields["allowedSenderIds"]); err != nil {
			return out, err
		}
	case len(fields) == 1 && fields["notify"] != nil:
		enabled, err := DshbotNotifyEnabled(DshbotCapabilityStatus{Profile: fields})
		if err != nil {
			return out, err
		}
		if err := ValidateDshbotNotifyCommand(DshbotNotifyCommand{Expected: expected, Enabled: enabled}); err != nil {
			return out, err
		}
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
	case len(fields) == 1 && fields["avatar"] != nil:
		values, err := DshbotAvatarValue(DshbotCapabilityStatus{Profile: fields})
		if err != nil {
			return out, err
		}
		if err := ValidateDshbotAvatarCommand(DshbotAvatarCommand{Expected: expected, Values: values}); err != nil {
			return out, err
		}
	case len(fields) == 3 && fields["name"] != nil && fields["title"] != nil && fields["description"] != nil:
		values, err := DshbotTextProfileValue(DshbotCapabilityStatus{Profile: fields})
		if err != nil {
			return out, err
		}
		if err := ValidateDshbotTextProfileCommand(DshbotTextProfileCommand{Expected: expected, Values: values}); err != nil {
			return out, err
		}
	case len(fields) == 2 && fields["model"] != nil && fields["workspaceId"] != nil && len(route) == 1:
		cmd := route[0]
		if err := ValidateDshbotRouteCommand(cmd); err != nil {
			return out, err
		}
		model, _ := json.Marshal(cmd.Values.Model)
		workspace, _ := json.Marshal(cmd.Values.WorkspaceID)
		if !SameDshbotCapabilities(expected, cmd.Expected.DshbotCapabilityStatus) || !sameCapabilityJSON(model, fields["model"]) || !sameCapabilityJSON(workspace, fields["workspaceId"]) {
			return out, fmt.Errorf("route review mismatch")
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
	if len(route) > 0 {
		before, _ := DshbotRouteValue(route[0].Expected)
		if before.WorkspaceID != route[0].Values.WorkspaceID && route[0].Values.WorkspaceID == "" {
			payload["scratchCwd"], _ = json.Marshal(route[0].Expected.Options.ScratchCwd)
		}
	}
	payload["revision"], _ = json.Marshal(expected.Revision)
	var response struct {
		View      dshbotCapabilityView `json:"view"`
		Warnings  []string             `json:"warnings"`
		Item      json.RawMessage      `json:"item"`
		SessionID string               `json:"sessionId"`
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
	if len(route) > 0 {
		if err := DshbotRouteApplied(route[0], out.Status); err != nil {
			return out, err
		}
		if response.SessionID != out.Status.Bot.SessionID || !sameDshbotProfileReceipt(response.Item, out.Status.Profile) {
			return out, fmt.Errorf("canonical route receipt is inconsistent; read back")
		}
		matches := 0
		for _, raw := range response.View.Value.Items {
			var row DshbotCatalogItem
			if json.Unmarshal(raw, &row) != nil {
				return out, fmt.Errorf("catalog identity receipt unavailable")
			}
			if row.SessionID == response.SessionID {
				matches++
			}
		}
		if matches != 1 {
			return out, fmt.Errorf("new canonical session identity is ambiguous")
		}
		return out, nil
	}
	if len(fields) == 1 && fields["notify"] != nil {
		enabled, err := DshbotNotifyEnabled(DshbotCapabilityStatus{Profile: fields})
		if err != nil {
			return out, err
		}
		if err := DshbotNotifyApplied(DshbotNotifyCommand{Expected: expected, Enabled: enabled}, out.Status); err != nil {
			return out, err
		}
		if response.SessionID != expected.Bot.SessionID || !sameDshbotProfileReceipt(response.Item, out.Status.Profile) {
			return out, fmt.Errorf("notification item/session receipt differs; read back")
		}
		return out, nil
	}
	if len(fields) == 1 && fields["avatar"] != nil {
		values, err := DshbotAvatarValue(DshbotCapabilityStatus{Profile: fields})
		if err != nil {
			return out, err
		}
		if err := DshbotAvatarApplied(DshbotAvatarCommand{Expected: expected, Values: values}, out.Status); err != nil {
			return out, err
		}
		if response.SessionID != expected.Bot.SessionID || !dshbotAvatarItemReceipt(response.Item, out.Status) {
			return out, fmt.Errorf("avatar item/session receipt differs; read back")
		}
		return out, nil
	}
	if len(fields) == 3 && fields["name"] != nil && fields["title"] != nil && fields["description"] != nil {
		values, err := DshbotTextProfileValue(DshbotCapabilityStatus{Profile: fields})
		if err != nil {
			return out, err
		}
		if err := DshbotTextProfileApplied(DshbotTextProfileCommand{Expected: expected, Values: values}, out.Status); err != nil {
			return out, err
		}
		if response.SessionID != expected.Bot.SessionID || !sameDshbotProfileReceipt(response.Item, out.Status.Profile) {
			return out, fmt.Errorf("text profile item/session receipt differs; read back")
		}
		return out, nil
	}
	if len(fields) == 1 && fields["allowedSenderIds"] != nil {
		ids, err := senderPolicyIDs(fields["allowedSenderIds"])
		if err != nil {
			return out, err
		}
		cmd := DshbotSenderPolicyCommand{Expected: DshbotSenderPolicyStatus{DshbotCapabilityStatus: expected}, Values: DshbotSenderPolicyValues{Selected: len(ids) > 0, IDs: ids}}
		if err := DshbotSenderPolicyApplied(cmd, out.Status); err != nil {
			return out, err
		}
		if response.SessionID != expected.Bot.SessionID || !sameDshbotProfileReceipt(response.Item, out.Status.Profile) {
			return out, fmt.Errorf("source policy item/session receipt differs; read back")
		}
		return out, nil
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
