package pluginrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Go replaces malformed UTF-16 escapes with U+FFFD; do not silently rewrite
// a capability name that JavaScript could retain verbatim.
func validCapabilityUnicode(raw json.RawMessage) bool {
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return false
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' {
			continue
		}
		if i+1 >= len(raw) {
			return false
		}
		if raw[i+1] != 'u' {
			i++
			continue
		}
		if i+6 > len(raw) {
			return false
		}
		n, err := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
		if err != nil {
			return false
		}
		if n >= 0xdc00 && n <= 0xdfff {
			return false
		}
		if n >= 0xd800 && n <= 0xdbff {
			if i+12 > len(raw) || raw[i+6] != '\\' || raw[i+7] != 'u' {
				return false
			}
			next, err := strconv.ParseUint(string(raw[i+8:i+12]), 16, 16)
			if err != nil || next < 0xdc00 || next > 0xdfff {
				return false
			}
			i += 6
		}
		i += 5
	}
	return true
}

func sameCapabilityJSON(a, b json.RawMessage) bool {
	decode := func(raw json.RawMessage) any {
		var v any
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		if d.Decode(&v) != nil {
			return nil
		}
		return v
	}
	return len(a) > 0 && len(b) > 0 && reflect.DeepEqual(decode(a), decode(b))
}
func capabilityObject(raw json.RawMessage, allowed ...string) (map[string]json.RawMessage, error) {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil || m == nil {
		return nil, fmt.Errorf("missing object")
	}
	for key := range m {
		if !slices.Contains(allowed, key) {
			return nil, fmt.Errorf("unknown nested field: %s", key)
		}
	}
	return m, nil
}
func capabilityText(p map[string]json.RawMessage, key string, max int, required bool) (string, error) {
	var s string
	raw, ok := p[key]
	if !ok || !validCapabilityUnicode(raw) || string(raw) == "null" || json.Unmarshal(raw, &s) != nil || trimBotMemory(s) != s || WhaleTextLength(s) > max || required && s == "" {
		return "", fmt.Errorf("profile %s is missing or would be normalized", key)
	}
	return s, nil
}
func capabilityNames(raw json.RawMessage) ([]string, error) {
	var names []string
	if !validCapabilityUnicode(raw) || string(raw) == "null" || json.Unmarshal(raw, &names) != nil || names == nil {
		return nil, fmt.Errorf("missing names")
	}
	seen := map[string]bool{}
	for _, name := range names {
		if trimBotMemory(name) != name || name == "" || WhaleTextLength(name) > 300 || seen[name] {
			return nil, fmt.Errorf("names would be normalized")
		}
		seen[name] = true
	}
	return names, nil
}
func decodeCapabilityPolicies(raw json.RawMessage) (DshbotCapabilityPolicies, error) {
	root, err := capabilityObject(raw, "tools", "skills", "mcp")
	if err != nil {
		return nil, err
	}
	out := DshbotCapabilityPolicies{}
	for _, group := range DshbotCapabilityGroups {
		p, err := capabilityObject(root[group], "mode", "names")
		if err != nil {
			return nil, err
		}
		mode, err := capabilityText(p, "mode", 20, true)
		if err != nil || (mode != "all" && mode != "selected") {
			return nil, fmt.Errorf("unknown capability mode")
		}
		names, err := capabilityNames(p["names"])
		if err != nil {
			return nil, err
		}
		out[group] = DshbotCapabilityPolicy{Mode: mode, Names: names}
	}
	return out, nil
}

// bot/update rebuilds these nested values and normalizes several scalar fields.
// Refuse incomplete or newer shapes rather than supplying defaults that can erase data.
func ValidateDshbotCapabilityProfile(s DshbotCapabilityStatus) error {
	p := s.Profile
	encoded, err := json.Marshal(p)
	if err != nil || !validCapabilityUnicode(encoded) {
		return fmt.Errorf("profile cannot be decoded losslessly")
	}
	if s.Bot.ID == "" || s.Bot.Kind != "bot" || s.Bot.Hidden || s.Bot.SessionID == "" || s.Revision < 0 {
		return fmt.Errorf("current individual Bot unavailable")
	}
	for _, f := range []struct {
		key      string
		max      int
		required bool
	}{{"id", 200, true}, {"kind", 20, true}, {"sessionId", 240, true}, {"name", 120, true}, {"title", 120, false}, {"description", 12000, false}, {"workspaceId", 2000, false}, {"sectionId", 200, false}, {"objective", 2000, false}} {
		if _, err := capabilityText(p, f.key, f.max, f.required); err != nil {
			return err
		}
	}
	var item DshbotCatalogItem
	raw, _ := json.Marshal(p)
	if json.Unmarshal(raw, &item) != nil || item.ID != s.Bot.ID || item.SessionID != s.Bot.SessionID || item.Kind != "bot" {
		return fmt.Errorf("profile identity mismatch")
	}
	for _, key := range []string{"pinned", "hidden", "readInitialized", "memoryReview", "notify"} {
		var b bool
		if p[key] == nil || string(p[key]) == "null" || json.Unmarshal(p[key], &b) != nil {
			return fmt.Errorf("profile %s unavailable", key)
		}
	}
	for _, key := range []string{"pinOrder", "lastSeenSeq", "createdAt", "updatedAt", "objectiveMaxRounds", "maxRounds", "maxSpeaks"} {
		var n float64
		if p[key] == nil || string(p[key]) == "null" || json.Unmarshal(p[key], &n) != nil {
			return fmt.Errorf("profile %s unavailable", key)
		}
		if (key == "maxRounds" && (n < 1 || n > 3 || n != float64(int(n)))) || (key == "maxSpeaks" && (n < 1 || n > 10 || n != float64(int(n)))) {
			return fmt.Errorf("group limit fields unavailable")
		}
		if key == "lastSeenSeq" && (n < 0 || n != float64(int64(n))) {
			return fmt.Errorf("read state would be normalized")
		}
		if key == "objectiveMaxRounds" && (n < 1 || n > 200 || n != float64(int(n))) {
			return fmt.Errorf("objective cap unavailable")
		}
	}
	var members []string
	if json.Unmarshal(p["memberBotIds"], &members) != nil || members == nil || len(members) != 0 {
		return fmt.Errorf("individual Bot members would be reset")
	}
	var senders []string
	if json.Unmarshal(p["allowedSenderIds"], &senders) != nil || senders == nil {
		return fmt.Errorf("source IDs unavailable")
	}
	seen := map[string]bool{}
	for _, id := range senders {
		if seen[id] || id == s.Bot.ID || !slices.Contains(s.PeerIDs, id) {
			return fmt.Errorf("source IDs unavailable or would be normalized")
		}
		seen[id] = true
	}
	var inbox []json.RawMessage
	if json.Unmarshal(p["inbox"], &inbox) != nil || inbox == nil {
		return fmt.Errorf("inbox unavailable")
	}
	model, err := capabilityObject(p["model"], "provider", "model", "reasoningEffort")
	if err != nil {
		return fmt.Errorf("model cannot be preserved: %w", err)
	}
	provider, err := capabilityText(model, "provider", 200, false)
	if err != nil {
		return err
	}
	name, err := capabilityText(model, "model", 500, false)
	if err != nil {
		return err
	}
	effort, err := capabilityText(model, "reasoningEffort", 100, false)
	if err != nil {
		return err
	}
	if (provider == "") != (name == "") || provider == "" && effort != "" {
		return fmt.Errorf("model cannot be preserved")
	}
	avatar, err := capabilityObject(p["avatar"], "kind", "shape", "color", "dataUrl", "crop")
	if err != nil {
		return fmt.Errorf("avatar cannot be preserved: %w", err)
	}
	stringValue := func(key string) string { var v string; json.Unmarshal(avatar[key], &v); return v }
	kind := stringValue("kind")
	switch kind {
	case "blob":
		if !slices.Contains([]string{"circle", "soft", "square", "pill", "triangle", "hex", "cloud", "drop"}, stringValue("shape")) || !slices.Contains([]string{"ink", "brown", "red", "orange", "yellow", "green", "teal", "blue", "purple", "pink", "grey"}, stringValue("color")) || stringValue("dataUrl") != "" || stringValue("crop") != "" {
			return fmt.Errorf("avatar would be normalized")
		}
	case "image":
		url, crop := stringValue("dataUrl"), stringValue("crop")
		lower := strings.ToLower(url)
		if !(strings.HasPrefix(lower, "data:image/jpeg;base64,") || strings.HasPrefix(lower, "data:image/jpg;base64,") || strings.HasPrefix(lower, "data:image/png;base64,")) || WhaleTextLength(url) > 120000 || (crop != "circle" && crop != "square") || stringValue("shape") != "" || stringValue("color") != "" {
			return fmt.Errorf("avatar would be normalized")
		}
	default:
		return fmt.Errorf("unknown avatar would be replaced")
	}
	for key, value := range avatar {
		var text string
		if string(value) == "null" || json.Unmarshal(value, &text) != nil {
			return fmt.Errorf("avatar %s unavailable", key)
		}
	}
	policy, err := decodeCapabilityPolicies(p["capabilities"])
	if err != nil {
		return fmt.Errorf("capabilities cannot be preserved: %w", err)
	}
	if !reflect.DeepEqual(policy, s.Policies) {
		return fmt.Errorf("capability profile snapshot mismatch")
	}
	return nil
}
