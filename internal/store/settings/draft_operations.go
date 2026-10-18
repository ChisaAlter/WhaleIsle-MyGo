package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"whaleisle/internal/dsh/api"
)

func copyFlags(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func hasSetSecrets(secrets []api.SettingsDescribeValueNamespacesItemSecretsItem) bool {
	for _, secret := range secrets {
		if secret.Set {
			return true
		}
	}
	return false
}

func configuredSecretPaths(secrets []api.SettingsDescribeValueNamespacesItemSecretsItem) [][]string {
	var paths [][]string
	for _, secret := range secrets {
		if secret.Set {
			paths = append(paths, append([]string(nil), secret.Path...))
		}
	}
	return paths
}

func cloneSecretPaths(paths [][]string) [][]string {
	out := make([][]string, len(paths))
	for i, path := range paths {
		out[i] = append([]string(nil), path...)
	}
	return out
}

// A redacted subtree cannot be safely rebuilt from visible JSON. Arrays are
// atomic edits, so even an innocent row edit can otherwise drop hidden keys.
// Explicit Reset is handled separately before this ordinary-edit guard.
func validateJSONSecretPaths(ns Namespace, ops []json.RawMessage) error {
	if len(ops) == 0 || (!ns.HasSecrets && len(ns.SecretPaths) == 0) {
		return nil
	}
	if len(ns.SecretPaths) == 0 {
		return fmt.Errorf("Hidden-secret paths are unavailable; refresh before editing this JSON section")
	}
	for _, raw := range ops {
		var op struct {
			Path []string `json:"path"`
		}
		if err := json.Unmarshal(raw, &op); err != nil {
			return err
		}
		for _, secret := range ns.SecretPaths {
			overlaps := true
			for i := 0; i < len(op.Path) && i < len(secret); i++ {
				if op.Path[i] != secret[i] {
					overlaps = false
					break
				}
			}
			if overlaps {
				return fmt.Errorf("JSON edit at %s would replace or remove hidden secrets; edit non-secret fields individually or use explicit Reset to remove the override", strings.Join(op.Path, "."))
			}
		}
	}
	return nil
}

// ResetDraft stages an unset, rather than serializing a default or null as an
// override. The accepted value remains authoritative until Save succeeds.
func (s *Store) ResetDraft(key string) {
	s.mu.Lock()
	ns := s.findNamespaceLocked(s.selectedNS)
	if ns == nil || !s.writable || s.saving || s.loading {
		s.mu.Unlock()
		return
	}
	for _, field := range ns.Fields {
		if field.Key != key {
			continue
		}
		s.ensureDraftsLocked(ns.NS)
		if s.resets == nil {
			s.resets = map[string]map[string]bool{}
		}
		if s.resets[ns.NS] == nil {
			s.resets[ns.NS] = map[string]bool{}
		}
		if s.dirty[ns.NS] == nil {
			s.dirty[ns.NS] = map[string]bool{}
		}
		s.resets[ns.NS][key] = true
		s.dirty[ns.NS][key] = true
		s.drafts[ns.NS][key] = ValueDraft(ns.Base, key, field.Kind)
		s.err = nil
		break
	}
	s.mu.Unlock()
	s.fire()
}

// draftWrite keeps untouched values out of writes, including optional fields,
// inherited defaults, and secret values intentionally omitted by describe.
func draftWrite(ns Namespace, drafts map[string]string, dirty, resets map[string]bool) (map[string]json.RawMessage, []json.RawMessage, map[string]string, error) {
	patch := map[string]json.RawMessage{}
	var ops []json.RawMessage
	submitted := map[string]string{}
	var effective map[string]json.RawMessage
	_ = json.Unmarshal(ns.Value, &effective)
	for _, field := range ns.Fields {
		if !dirty[field.Key] {
			continue
		}
		draft := drafts[field.Key]
		submitted[field.Key] = draft
		path := []string{field.Key}
		if resets[field.Key] || (field.Kind == FieldNumber && strings.TrimSpace(draft) == "") {
			ops = append(ops, fieldOperation("unset", path, nil))
			continue
		}
		raw, err := PatchValue(field.Kind, draft)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%s: %w", field.Key, err)
		}
		if field.Kind == FieldJSON {
			// Diff visible leaves; replacing the entire redacted object would
			// erase secrets and materialize unrelated inherited settings.
			var fieldOps []json.RawMessage
			diffVisibleJSON(effective[field.Key], raw, path, &fieldOps)
			if err := validateJSONSecretPaths(ns, fieldOps); err != nil {
				return nil, nil, nil, err
			}
			ops = append(ops, fieldOps...)
		} else {
			patch[field.Key] = raw
		}
	}
	if len(ops) > 0 {
		keys := make([]string, 0, len(patch))
		for key := range patch {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			ops = append(ops, fieldOperation("set", []string{key}, patch[key]))
		}
		patch = map[string]json.RawMessage{}
	}
	return patch, ops, submitted, nil
}

func fieldOperation(op string, path []string, value json.RawMessage) json.RawMessage {
	payload := map[string]any{"op": op, "path": path}
	if op == "set" {
		payload["value"] = value
	}
	raw, _ := json.Marshal(payload)
	return raw
}

func jsonObject(raw json.RawMessage) (map[string]json.RawMessage, bool) {
	var value map[string]json.RawMessage
	err := json.Unmarshal(raw, &value)
	return value, err == nil && value != nil
}

func sameJSON(a, b json.RawMessage) bool {
	var left, right bytes.Buffer
	return json.Compact(&left, a) == nil && json.Compact(&right, b) == nil && bytes.Equal(left.Bytes(), right.Bytes())
}

func diffVisibleJSON(before, after json.RawMessage, path []string, ops *[]json.RawMessage) {
	if sameJSON(before, after) {
		return
	}
	oldObject, wasObject := jsonObject(before)
	newObject, isObject := jsonObject(after)
	if wasObject && isObject {
		keys := map[string]bool{}
		for key := range oldObject {
			keys[key] = true
		}
		for key := range newObject {
			keys[key] = true
		}
		ordered := make([]string, 0, len(keys))
		for key := range keys {
			ordered = append(ordered, key)
		}
		sort.Strings(ordered)
		for _, key := range ordered {
			childPath := append(append([]string(nil), path...), key)
			next, exists := newObject[key]
			if !exists {
				*ops = append(*ops, fieldOperation("unset", childPath, nil))
			} else {
				diffVisibleJSON(oldObject[key], next, childPath, ops)
			}
		}
		return
	}
	*ops = append(*ops, fieldOperation("set", path, after))
}

func replacementFromDrafts(ns Namespace, drafts map[string]string, dirty, resets map[string]bool) (map[string]json.RawMessage, error) {
	if ns.HasSecrets {
		return nil, fmt.Errorf("This section contains hidden secrets. Use Save to preserve them; full replacement is unavailable")
	}
	section, _ := jsonObject(ns.User)
	if section == nil {
		section = map[string]json.RawMessage{}
	}
	patch, ops, _, err := draftWrite(ns, drafts, dirty, resets)
	if err != nil {
		return nil, err
	}
	for key, value := range patch {
		section[key] = value
	}
	for _, raw := range ops {
		var op struct {
			Op    string          `json:"op"`
			Path  []string        `json:"path"`
			Value json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal(raw, &op); err != nil {
			return nil, err
		}
		applySectionPath(section, op.Path, op.Value, op.Op == "unset")
	}
	return section, nil
}

// Only object paths are generated: arrays remain atomic values.
func applySectionPath(section map[string]json.RawMessage, path []string, value json.RawMessage, unset bool) {
	if len(path) == 0 {
		return
	}
	key := path[0]
	if len(path) == 1 {
		if unset {
			delete(section, key)
		} else {
			section[key] = value
		}
		return
	}
	child, ok := jsonObject(section[key])
	if !ok {
		if unset {
			return
		}
		child = map[string]json.RawMessage{}
	}
	applySectionPath(child, path[1:], value, unset)
	section[key], _ = json.Marshal(child)
}
