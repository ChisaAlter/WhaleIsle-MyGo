// Package shortcutprefs owns revision-checked shortcut preferences shared by
// native and product clients. It stores physical bindings, not OS hotkeys.
package shortcutprefs

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Binding describes one or two distinct physical keys held simultaneously.
type Binding struct {
	Code       string   `json:"code"`
	SecondCode string   `json:"secondCode,omitempty"`
	Modifiers  []string `json:"modifiers"`
}

// Document is the original schema. Missing IDs inherit defaults; nil disables.
type Document struct {
	SchemaVersion int                            `json:"schemaVersion"`
	Profiles      map[string]map[string]*Binding `json:"profiles"`
}

// Definition is a trusted owner's command catalog entry. Non-nil Fixed entries
// reserve combinations and cannot be edited by that owner.
type Definition struct {
	ID       string             `json:"id"`
	Defaults map[string]Binding `json:"defaults"`
	Fixed    []Binding          `json:"fixed,omitempty"`
}

// Edit changes one override; reset-all affects only the owner's editable IDs.
type Edit struct {
	Type    string   `json:"type"`
	ID      string   `json:"id,omitempty"`
	Binding *Binding `json:"binding,omitempty"`
}

// EffectiveShortcut is a projection of accepted preferences and current catalog.
type EffectiveShortcut struct {
	ID        string   `json:"id"`
	Binding   *Binding `json:"binding"`
	Modified  bool     `json:"modified"`
	Issue     *string  `json:"issue"`
	Conflicts []string `json:"conflicts"`
}

var commandPattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9-]*(\.[a-zA-Z][a-zA-Z0-9-]*)+$`)
var profilePattern = regexp.MustCompile(`^(desktop|web):(macos|windows|linux)$`)
var codePattern = regexp.MustCompile(`^(Key[A-Z]|Digit[0-9]|F([1-9]|1[0-9]|2[0-4]))$`)
var namedCodes = map[string]bool{"Slash": true, "Comma": true, "Period": true, "Backslash": true, "Backquote": true, "Minus": true, "Equal": true, "BracketLeft": true, "BracketRight": true, "Semicolon": true, "Quote": true, "Enter": true, "Escape": true, "Space": true, "Tab": true, "Backspace": true, "Delete": true, "ArrowUp": true, "ArrowDown": true, "ArrowLeft": true, "ArrowRight": true}
var modifierOrder = []string{"control", "alt", "shift", "meta"}

func object(raw []byte) (map[string]json.RawMessage, error) {
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return nil, errors.New("expected object")
	}
	return value, nil
}
func fields(value map[string]json.RawMessage, allowed ...string) bool {
	for key := range value {
		found := false
		for _, name := range allowed {
			if key == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
func stringValue(raw []byte) (string, error) {
	var value string
	if len(raw) == 0 || string(raw) == "null" {
		return "", errors.New("expected string")
	}
	err := json.Unmarshal(raw, &value)
	return value, err
}
func parseBinding(raw []byte) (*Binding, error) {
	if string(raw) == "null" {
		return nil, nil
	}
	value, err := object(raw)
	if err != nil || !fields(value, "code", "secondCode", "modifiers") {
		return nil, errors.New("invalid binding")
	}
	code, err := stringValue(value["code"])
	if err != nil {
		return nil, err
	}
	modsRaw, ok := value["modifiers"]
	if !ok || len(modsRaw) == 0 || string(modsRaw) == "null" {
		return nil, errors.New("invalid modifiers")
	}
	var mods []string
	if err = json.Unmarshal(modsRaw, &mods); err != nil || mods == nil {
		return nil, errors.New("invalid modifiers")
	}
	out := &Binding{Code: code, Modifiers: mods}
	if raw, ok := value["secondCode"]; ok {
		out.SecondCode, err = stringValue(raw)
		if err != nil || out.SecondCode == "" {
			return nil, errors.New("invalid second code")
		}
	}
	if err = validateBinding(*out); err != nil {
		return nil, err
	}
	return out, nil
}
func validateBinding(binding Binding) error {
	if !codePattern.MatchString(binding.Code) && !namedCodes[binding.Code] {
		return errors.New("unsupported shortcut code")
	}
	if binding.SecondCode != "" && ((!codePattern.MatchString(binding.SecondCode) && !namedCodes[binding.SecondCode]) || binding.SecondCode == binding.Code) {
		return errors.New("invalid second shortcut code")
	}
	for _, m := range binding.Modifiers {
		if m != "primary" && m != "control" && m != "alt" && m != "shift" && m != "meta" {
			return errors.New("invalid modifier")
		}
	}
	return nil
}

// ParseDocument accepts schema 1 or 2 and preserves dormant command bindings.
// A nil byte slice denotes missing preferences; errors are invalid or future.
func ParseDocument(raw []byte) (Document, string) {
	empty := Document{SchemaVersion: 2, Profiles: map[string]map[string]*Binding{}}
	if raw == nil {
		return empty, ""
	}
	value, err := object(raw)
	if err != nil {
		return empty, "invalid"
	}
	var version float64
	if json.Unmarshal(value["schemaVersion"], &version) == nil && version > 2 {
		return empty, "future"
	}
	if !fields(value, "schemaVersion", "profiles") || (version != 1 && version != 2) {
		return empty, "invalid"
	}
	profiles, err := object(value["profiles"])
	if err != nil {
		return empty, "invalid"
	}
	doc := Document{SchemaVersion: int(version), Profiles: map[string]map[string]*Binding{}}
	for profile, rawProfile := range profiles {
		if !profilePattern.MatchString(profile) {
			return empty, "invalid"
		}
		entries, err := object(rawProfile)
		if err != nil {
			return empty, "invalid"
		}
		bindings := map[string]*Binding{}
		for id, rawBinding := range entries {
			if !commandPattern.MatchString(id) {
				return empty, "invalid"
			}
			binding, err := parseBinding(rawBinding)
			if err != nil || (version == 1 && binding != nil && binding.SecondCode != "") {
				return empty, "invalid"
			}
			bindings[id] = binding
		}
		doc.Profiles[profile] = bindings
	}
	return doc, ""
}

// ParseEdit validates the exact original IPC edit shape, including explicit null.
func ParseEdit(raw []byte) (Edit, error) {
	value, err := object(raw)
	if err != nil {
		return Edit{}, err
	}
	kind, err := stringValue(value["type"])
	if err != nil {
		return Edit{}, err
	}
	if kind == "reset-all" && len(value) == 1 {
		return Edit{Type: kind}, nil
	}
	id, err := stringValue(value["id"])
	if err != nil || !commandPattern.MatchString(id) {
		return Edit{}, errors.New("invalid shortcut command")
	}
	if kind == "reset" && len(value) == 2 {
		return Edit{Type: kind, ID: id}, nil
	}
	if kind == "set" && len(value) == 3 {
		rawBinding, ok := value["binding"]
		if !ok {
			return Edit{}, errors.New("binding required")
		}
		binding, err := parseBinding(rawBinding)
		if err != nil {
			return Edit{}, err
		}
		return Edit{Type: kind, ID: id, Binding: binding}, nil
	}
	return Edit{}, errors.New("invalid shortcut edit")
}

// MarshalJSON preserves the distinct set-null, reset, and reset-all wire forms.
func (edit Edit) MarshalJSON() ([]byte, error) {
	switch edit.Type {
	case "set":
		return json.Marshal(struct {
			Type    string   `json:"type"`
			ID      string   `json:"id"`
			Binding *Binding `json:"binding"`
		}{edit.Type, edit.ID, edit.Binding})
	case "reset":
		return json.Marshal(struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}{edit.Type, edit.ID})
	case "reset-all":
		return []byte(`{"type":"reset-all"}`), nil
	default:
		return nil, errors.New("invalid shortcut edit")
	}
}

// UnmarshalJSON applies the same strict boundary as ParseEdit.
func (edit *Edit) UnmarshalJSON(raw []byte) error {
	value, err := ParseEdit(raw)
	if err == nil {
		*edit = value
	}
	return err
}

// ParseDefinitions validates a serialized trusted catalog before registration.
func ParseDefinitions(raw []byte) ([]Definition, error) {
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil || entries == nil {
		return nil, errors.New("invalid shortcut catalog")
	}
	defs := make([]Definition, 0, len(entries))
	for _, raw := range entries {
		entry, err := object(raw)
		if err != nil || !fields(entry, "id", "defaults", "fixed") {
			return nil, errors.New("invalid shortcut definition")
		}
		id, err := stringValue(entry["id"])
		if err != nil {
			return nil, err
		}
		defaults, err := object(entry["defaults"])
		if err != nil {
			return nil, err
		}
		def := Definition{ID: id, Defaults: map[string]Binding{}}
		for profile, rawBinding := range defaults {
			b, err := parseBinding(rawBinding)
			if err != nil || b == nil {
				return nil, errors.New("invalid shortcut default")
			}
			def.Defaults[profile] = *b
		}
		if fixedRaw, ok := entry["fixed"]; ok {
			var fixed []json.RawMessage
			if err = json.Unmarshal(fixedRaw, &fixed); err != nil || len(fixed) == 0 {
				return nil, errors.New("invalid fixed shortcuts")
			}
			def.Fixed = []Binding{}
			for _, rawBinding := range fixed {
				b, err := parseBinding(rawBinding)
				if err != nil || b == nil {
					return nil, errors.New("invalid fixed shortcut")
				}
				def.Fixed = append(def.Fixed, *b)
			}
		}
		defs = append(defs, def)
	}
	if err := validateDefinitions(defs); err != nil {
		return nil, err
	}
	return defs, nil
}
func validateDefinitions(defs []Definition) error {
	ids := map[string]bool{}
	defaults := map[string][]Binding{}
	for _, def := range defs {
		if !commandPattern.MatchString(def.ID) || ids[def.ID] || def.Defaults == nil {
			return errors.New("invalid or duplicate shortcut definition")
		}
		ids[def.ID] = true
		if def.Fixed != nil {
			if len(def.Fixed) == 0 || len(def.Defaults) > 0 {
				return errors.New("invalid fixed shortcut definition")
			}
			for _, b := range def.Fixed {
				if err := validateBinding(b); err != nil {
					return err
				}
			}
		}
		for profile, b := range def.Defaults {
			if !profilePattern.MatchString(profile) {
				return errors.New("invalid shortcut profile")
			}
			if err := validateBinding(b); err != nil {
				return err
			}
			normalized := normalize(b, profile)
			if bindingIssue(normalized, profile) != "" {
				return errors.New("reserved shortcut default")
			}
			for _, other := range defaults[profile] {
				if overlap(normalized, other) {
					return errors.New("conflicting shortcut defaults")
				}
			}
			defaults[profile] = append(defaults[profile], normalized)
		}
	}
	return nil
}
func normalize(b Binding, profile string) Binding {
	out := Binding{Code: b.Code, SecondCode: b.SecondCode, Modifiers: []string{}}
	if out.SecondCode != "" && out.Code > out.SecondCode {
		out.Code, out.SecondCode = out.SecondCode, out.Code
	}
	mods := map[string]bool{}
	for _, m := range b.Modifiers {
		if m == "primary" {
			if strings.HasSuffix(profile, ":macos") {
				m = "meta"
			} else {
				m = "control"
			}
		}
		mods[m] = true
	}
	for _, m := range modifierOrder {
		if mods[m] {
			out.Modifiers = append(out.Modifiers, m)
		}
	}
	return out
}
func overlap(a, b Binding) bool {
	if strings.Join(a.Modifiers, "+") != strings.Join(b.Modifiers, "+") {
		return false
	}
	if a.SecondCode != "" && b.SecondCode != "" {
		return a.Code == b.Code && a.SecondCode == b.SecondCode
	}
	return a.Code == b.Code || (a.SecondCode != "" && a.SecondCode == b.Code) || (b.SecondCode != "" && a.Code == b.SecondCode)
}
func has(mods []string, want string) bool {
	for _, m := range mods {
		if m == want {
			return true
		}
	}
	return false
}
func webAllowed(b Binding, profile string) bool {
	if b.SecondCode != "" {
		return false
	}
	if strings.HasSuffix(profile, ":windows") || strings.HasSuffix(profile, ":macos") {
		if len(b.Modifiers) >= 3 {
			return true
		}
		primary := "control"
		if strings.HasSuffix(profile, ":macos") {
			primary = "meta"
		}
		if len(b.Modifiers) == 1 && (((b.Code == "Comma" || b.Code == "Backslash") && b.Modifiers[0] == primary) || (b.Code == "Backquote" && b.Modifiers[0] == "control")) {
			return true
		}
		if len(b.Modifiers) == 2 && has(b.Modifiers, primary) && (has(b.Modifiers, "alt") || has(b.Modifiers, "shift")) {
			return true
		}
	}
	for _, candidate := range []Binding{{Code: "Slash", Modifiers: []string{"primary"}}, {Code: "Comma", Modifiers: []string{"primary", "shift"}}, {Code: "Period", Modifiers: []string{"primary", "shift"}}} {
		if overlap(b, normalize(candidate, profile)) {
			return true
		}
	}
	return false
}
func bindingIssue(b Binding, profile string) string {
	mac := strings.HasSuffix(profile, ":macos")
	win := strings.HasSuffix(profile, ":windows")
	web := strings.HasPrefix(profile, "web:")
	if !web && (mac || win) {
		return ""
	}
	if b.SecondCode != "" {
		return "unsupported-key"
	}
	if len(b.Modifiers) == 0 || (len(b.Modifiers) == 1 && b.Modifiers[0] == "shift") {
		return "modifier-required"
	}
	if (mac || win) && len(b.Modifiers) >= 3 {
		return ""
	}
	if web && (mac || win) && webAllowed(b, profile) {
		return ""
	}
	primary := has(b.Modifiers, "control")
	if mac {
		primary = has(b.Modifiers, "meta")
	}
	reserved := map[string]bool{"Escape": true, "Tab": true, "Space": true, "Backspace": true, "Delete": true, "ArrowUp": true, "ArrowDown": true, "ArrowLeft": true, "ArrowRight": true}
	editing := map[string]bool{"KeyC": true, "KeyV": true, "KeyX": true, "KeyZ": true, "KeyY": true, "KeyQ": true, "KeyH": true}
	if reserved[b.Code] || (b.Code == "Enter" && !has(b.Modifiers, "alt")) || (primary && editing[b.Code]) || (primary && b.Code == "KeyA" && !has(b.Modifiers, "shift")) || (!mac && (has(b.Modifiers, "meta") || (has(b.Modifiers, "alt") && (b.Code == "F4" || b.Code == "F2")))) || (mac && has(b.Modifiers, "control") && has(b.Modifiers, "meta")) || (mac && has(b.Modifiers, "alt") && !primary) {
		return "reserved"
	}
	if web && !webAllowed(b, profile) {
		return "unsupported-browser"
	}
	return ""
}

// Effective resolves the entire catalog before returning deterministic rows.
// Shared IDs are collapsed; all fixed reservations participate in conflicts.
func Effective(defs []Definition, doc Document, profile string) []EffectiveShortcut {
	type reservation struct {
		id string
		b  Binding
	}
	fixed := []reservation{}
	byID := map[string]Definition{}
	for _, def := range defs {
		if def.Fixed != nil {
			for _, b := range def.Fixed {
				fixed = append(fixed, reservation{def.ID, normalize(b, profile)})
			}
		} else {
			byID[def.ID] = def
		}
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	rows := make([]EffectiveShortcut, 0, len(ids))
	overrides := doc.Profiles[profile]
	for _, id := range ids {
		b, modified := overrides[id]
		if !modified {
			if def, ok := byID[id].Defaults[profile]; ok {
				b = &def
			}
		}
		row := EffectiveShortcut{ID: id, Modified: modified, Conflicts: []string{}}
		if b != nil {
			normalized := normalize(*b, profile)
			row.Binding = &normalized
			if issue := bindingIssue(normalized, profile); issue != "" {
				row.Issue = &issue
			}
		}
		rows = append(rows, row)
	}
	for i := range rows {
		row := &rows[i]
		if row.Binding == nil {
			continue
		}
		conflicts := map[string]bool{}
		for _, other := range rows {
			if other.ID != row.ID && other.Binding != nil && other.Issue == nil && (!row.Modified || other.Modified) && overlap(*row.Binding, *other.Binding) {
				conflicts[other.ID] = true
			}
		}
		for _, other := range fixed {
			if other.id != row.ID && overlap(*row.Binding, other.b) {
				conflicts[other.id] = true
			}
		}
		for id := range conflicts {
			row.Conflicts = append(row.Conflicts, id)
		}
		sort.Strings(row.Conflicts)
	}
	return rows
}
func cloneBinding(b *Binding) *Binding {
	if b == nil {
		return nil
	}
	out := *b
	out.Modifiers = append([]string{}, b.Modifiers...)
	return &out
}
func cloneDocument(doc Document) Document {
	out := Document{SchemaVersion: doc.SchemaVersion, Profiles: map[string]map[string]*Binding{}}
	for profile, entries := range doc.Profiles {
		values := map[string]*Binding{}
		for id, b := range entries {
			values[id] = cloneBinding(b)
		}
		out.Profiles[profile] = values
	}
	return out
}
func cloneDefinitions(defs []Definition) []Definition {
	out := make([]Definition, len(defs))
	for i, d := range defs {
		out[i] = Definition{ID: d.ID, Defaults: map[string]Binding{}}
		for p, b := range d.Defaults {
			out[i].Defaults[p] = *cloneBinding(&b)
		}
		if d.Fixed != nil {
			out[i].Fixed = []Binding{}
			for _, b := range d.Fixed {
				out[i].Fixed = append(out[i].Fixed, *cloneBinding(&b))
			}
		}
	}
	return out
}
func validateEdit(e Edit) error {
	if e.Type == "reset-all" && e.ID == "" && e.Binding == nil {
		return nil
	}
	if !commandPattern.MatchString(e.ID) {
		return errors.New("invalid shortcut command")
	}
	switch e.Type {
	case "reset":
		if e.Binding != nil {
			return errors.New("reset has binding")
		}
	case "set":
		if e.Binding != nil {
			return validateBinding(*e.Binding)
		}
	default:
		return fmt.Errorf("invalid shortcut edit %q", e.Type)
	}
	return nil
}
