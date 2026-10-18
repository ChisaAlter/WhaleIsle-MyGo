package pluginrpc

import "encoding/json"

// Receipt-only equality for the exact AvatarSchema empty-string defaults.
// This must never replace CAS snapshot equality or permit unknown avatar keys.
func sameDshbotAvatarReceipt(a, b json.RawMessage) bool {
	if !validCapabilityUnicode(a) || !validCapabilityUnicode(b) {
		return false
	}
	left, err := DshbotAvatarValue(DshbotCapabilityStatus{Profile: map[string]json.RawMessage{"avatar": a}})
	if err != nil {
		return false
	}
	right, err := DshbotAvatarValue(DshbotCapabilityStatus{Profile: map[string]json.RawMessage{"avatar": b}})
	return err == nil && left == right
}
func sameDshbotProfileReceipt(item json.RawMessage, profile map[string]json.RawMessage) bool {
	if !validCapabilityUnicode(item) {
		return false
	}
	var before map[string]json.RawMessage
	if json.Unmarshal(item, &before) != nil || before == nil || len(before) != len(profile) {
		return false
	}
	for key, value := range profile {
		if key == "avatar" {
			if !sameDshbotAvatarReceipt(before[key], value) {
				return false
			}
		} else if !sameCapabilityJSON(before[key], value) {
			return false
		}
	}
	return true
}
func sameDshbotCreatedItemReceipt(item, view DshbotBotItem) bool {
	if item.ID != view.ID || item.SessionID != view.SessionID || item.Kind != view.Kind {
		return false
	}
	// New normalizeBotDraft items omit holds; ItemSchema supplies exactly {}.
	// This one-way exception is exclusive to create/duplicate item receipts.
	if _, present := item.Profile["holds"]; !present {
		if value, viewHas := view.Profile["holds"]; viewHas {
			if !sameCapabilityJSON(value, json.RawMessage(`{}`)) {
				return false
			}
			item = CloneDshbotBotItem(item)
			item.Profile["holds"] = json.RawMessage(`{}`)
		}
	}
	raw, err := json.Marshal(item.Profile)
	return err == nil && sameDshbotProfileReceipt(raw, view.Profile)
}
