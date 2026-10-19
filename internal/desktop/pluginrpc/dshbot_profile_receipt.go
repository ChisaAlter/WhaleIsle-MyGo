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
	// normalizeBotDraft omits these room-runtime fields on new items, while
	// ItemSchema materializes their exact empty defaults in the catalog view.
	// This one-way exception is exclusive to create/duplicate item receipts.
	for key, empty := range map[string]json.RawMessage{
		"holds":          json.RawMessage(`{}`),
		"holdCheckpoint": json.RawMessage(`{"version":1,"holds":{},"processedEventIds":[],"processedThroughSeq":-1}`),
	} {
		if _, present := item.Profile[key]; present {
			continue
		}
		if value, viewHas := view.Profile[key]; viewHas {
			if !sameCapabilityJSON(value, empty) {
				return false
			}
			item = CloneDshbotBotItem(item)
			item.Profile[key] = empty
		}
	}
	raw, err := json.Marshal(item.Profile)
	return err == nil && sameDshbotProfileReceipt(raw, view.Profile)
}
