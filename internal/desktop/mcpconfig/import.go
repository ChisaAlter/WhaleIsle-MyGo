package mcpconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"whaleisle/internal/dsh/api"
)

// ImportState is a local preview, never a queue of Host writes.
type ImportState struct {
	Text, DefaultName, Err, Notice string
	Candidates                     []ImportCandidate
	Selected, Ignored              int
	Ready, ReplaceExisting         bool
	Token                          uint64
	StagedID                       string
}
type ImportCandidate struct {
	Key, ID, Name, Transport, Command, ArgsJSON, Cwd, URL, Err string
	Enabled, Existing, Writable                                bool
	EnvCount, HeaderCount                                      int
	Warnings                                                   []string
	spec                                                       json.RawMessage
}

func cloneImport(s ImportState) ImportState {
	s.Candidates = append([]ImportCandidate(nil), s.Candidates...)
	for i := range s.Candidates {
		s.Candidates[i].spec = append(json.RawMessage(nil), s.Candidates[i].spec...)
		s.Candidates[i].Warnings = append([]string(nil), s.Candidates[i].Warnings...)
	}
	return s
}
func (c *Controller) SetImportInput(text, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Import.Text == text && c.state.Import.DefaultName == name {
		return
	}
	c.importSequence++
	c.state.Import = ImportState{Text: text, DefaultName: name, Selected: -1, Token: c.importSequence}
}
func (c *Controller) ClearImport() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.importSequence++
	c.state.Import = ImportState{Selected: -1, Token: c.importSequence}
}
func (c *Controller) ParseImport() {
	c.mu.Lock()
	source := c.state.Import
	revision := c.revision
	fallback := FormDraft{Enabled: true}
	if c.state.Form != nil {
		fallback = *c.state.Form
	} else if form, err := parseForm(c.state.Draft); err == nil {
		fallback = form
	}
	if source.DefaultName != "" {
		fallback.ID = source.DefaultName
		fallback.ServerName = source.DefaultName
	}
	rows := append([]api.McpServerSnapshotServersItem(nil), c.state.Rows...)
	c.mu.Unlock()
	candidates, ignored, err := parseImportRecords(source.Text, fallback, rows)
	c.mu.Lock()
	defer c.mu.Unlock()
	if source.Token != c.state.Import.Token || revision != c.revision {
		return
	}
	c.importSequence++
	c.state.Import.Token = c.importSequence
	c.state.Import.Selected = -1
	c.state.Import.ReplaceExisting = false
	c.state.Import.Notice = ""
	c.state.Import.StagedID = ""
	c.state.Import.Candidates = candidates
	c.state.Import.Ignored = ignored
	c.state.Import.Ready = err == nil
	c.state.Import.Err = ""
	if err != nil {
		c.state.Import.Err = err.Error()
	}
}
func (c *Controller) SelectImport(index int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.state.Import.Ready || index < 0 || index >= len(c.state.Import.Candidates) {
		return
	}
	c.state.Import.Selected = index
	c.state.Import.ReplaceExisting = false
	c.state.Import.Err = ""
}
func (c *Controller) SetImportReplace(value bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Import.ReplaceExisting = value
}

// StageImportExpected applies exactly one explicitly selected preview to local
// draft state. Only the existing Save action may call mcpServers/upsert later.
func (c *Controller) StageImportExpected(token uint64, index int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state := &c.state.Import
	if c.state.Busy || !state.Ready || state.Token != token || state.Selected != index || index < 0 || index >= len(state.Candidates) {
		return
	}
	candidate := state.Candidates[index]
	target := ""
	for _, row := range c.state.Rows {
		if row.Id != candidate.ID {
			if ServerName(row) == candidate.Name {
				state.Err = "Tool namespace is already used by " + row.Id
				return
			}
			continue
		}
		if !row.Writable {
			state.Err = "This server is read-only"
			return
		}
		if !state.ReplaceExisting {
			state.Err = "This ID already exists. Explicitly choose replacement before loading its draft"
			return
		}
		target = row.Id
	}
	form, err := parseForm(string(candidate.spec))
	if err != nil {
		state.Err = err.Error()
		return
	}
	raw, issues := formSpec(form, target, c.state.Rows)
	if len(issues) > 0 {
		state.Err = importIssues(issues)
		return
	}
	c.state.Draft = string(raw)
	c.state.EditingID = target
	c.state.Form = &form
	c.state.FormErrors = nil
	c.state.Err = ""
	c.state.LastAction = ""
	c.revision++
	state.StagedID = form.ID
	state.Err = ""
	state.Notice = "Loaded " + form.ID + " into the draft. Save explicitly to apply; other preview entries are untouched"
}
func (c *Controller) finishImportSave(spec json.RawMessage) {
	if c.state.Import.StagedID == "" {
		return
	}
	var identity struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(spec, &identity)
	c.state.Import.Notice = "Saved " + identity.ID + ". Other preview entries were not applied"
	c.state.Import.StagedID = ""
}
func (c *Controller) cancelImportDraft() {
	if c.state.Import.StagedID == "" {
		return
	}
	c.state.Import.Notice = "Imported draft discarded. No configuration was written by this draft"
	c.state.Import.StagedID = ""
}

type importEntry struct {
	key   string
	value json.RawMessage
}

func importObject(raw json.RawMessage) ([]importEntry, error) {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || object == nil {
		return nil, errors.New("Import must be a JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	_, _ = decoder.Token()
	entries := []importEntry{}
	seen := map[string]bool{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, errors.New("Invalid object key")
		}
		if seen[key] {
			return nil, fmt.Errorf("Duplicate JSON key %q is ambiguous", key)
		}
		seen[key] = true
		var value json.RawMessage
		if err = decoder.Decode(&value); err != nil {
			return nil, err
		}
		entries = append(entries, importEntry{key, value})
	}
	return entries, nil
}
func importString(raw json.RawMessage) string {
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}
func importBool(raw json.RawMessage) (bool, bool) {
	var value bool
	if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &value) != nil {
		return false, false
	}
	return value, true
}
func isImportSpec(raw json.RawMessage) bool {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || object == nil {
		return false
	}
	for _, key := range []string{"command", "url", "transport", "type"} {
		var value string
		if field, ok := object[key]; ok && string(field) != "null" && json.Unmarshal(field, &value) == nil {
			return true
		}
	}
	return false
}
func parseImportRecords(text string, fallback FormDraft, rows []api.McpServerSnapshotServersItem) ([]ImportCandidate, int, error) {
	if len(text) > 1024*1024 {
		return nil, 0, errors.New("Import preview is limited to 1 MiB of pasted JSON")
	}
	root := json.RawMessage(text)
	entries, err := importObject(root)
	if err != nil {
		return nil, 0, err
	}
	for _, entry := range entries {
		if entry.key == "mcpServers" {
			if nested, nestedErr := importObject(entry.value); nestedErr == nil {
				root = entry.value
				entries = nested
			}
			break
		}
	}
	if isImportSpec(root) {
		name := fallback.ServerName
		if name == "" {
			name = fallback.ID
		}
		if name == "" {
			return nil, 0, errors.New("A single spec needs a default name or current editor identity; alternatively use a named object")
		}
		entries = []importEntry{{name, root}}
	}
	candidates := []ImportCandidate{}
	ignored := 0
	for _, entry := range entries {
		if !isImportSpec(entry.value) {
			ignored++
			continue
		}
		if len(candidates) >= 256 {
			return nil, ignored, errors.New("Preview is limited to 256 server entries; split this input")
		}
		normalized, warnings := normalizeImportSpec(entry.key, entry.value, fallback)
		candidate := ImportCandidate{Key: entry.key, spec: normalized, Warnings: warnings, Writable: true}
		form, formErr := parseForm(string(normalized))
		candidate.ID = form.ID
		candidate.Name = form.ServerName
		candidate.Transport = form.Transport
		candidate.Enabled = form.Enabled
		candidate.Command = form.Command
		candidate.ArgsJSON = form.ArgsJSON
		candidate.Cwd = form.Cwd
		candidate.URL = form.URL
		var maps struct {
			Env     map[string]string `json:"env"`
			Headers map[string]string `json:"headers"`
		}
		_ = json.Unmarshal(normalized, &maps)
		candidate.EnvCount = len(maps.Env)
		candidate.HeaderCount = len(maps.Headers)
		target := ""
		for _, row := range rows {
			if row.Id == candidate.ID {
				candidate.Existing = true
				candidate.Writable = row.Writable
				target = row.Id
				break
			}
		}
		if formErr != nil {
			candidate.Err = formErr.Error()
		} else {
			_, issues := formSpec(form, target, rows)
			candidate.Err = importIssues(issues)
		}
		for _, row := range rows {
			if row.Id != candidate.ID && ServerName(row) == candidate.Name {
				candidate.Err = "Tool namespace is already used by " + row.Id
			}
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return nil, ignored, errors.New("No supported server spec was found")
	}
	return candidates, ignored, nil
}
func importIssues(issues map[string]string) string {
	keys := make([]string, 0, len(issues))
	for key := range issues {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	messages := []string{}
	for _, key := range keys {
		messages = append(messages, key+": "+issues[key])
	}
	return strings.Join(messages, "; ")
}
func importName(values ...string) string {
	for _, value := range values {
		if mcpFormName.MatchString(value) {
			return value
		}
	}
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
func normalizeImportSpec(name string, input json.RawMessage, fallback FormDraft) (json.RawMessage, []string) {
	var source map[string]json.RawMessage
	_ = json.Unmarshal(input, &source)
	record := map[string]json.RawMessage{}
	warnings := []string{}
	set := func(key string, value any) { encoded, _ := json.Marshal(value); record[key] = encoded }
	set("id", importName(importString(source["id"]), name, fallback.ID))
	set("serverName", importName(importString(source["serverName"]), name, fallback.ServerName))
	enabled, ok := importBool(source["enabled"])
	if !ok {
		enabled = fallback.Enabled
	}
	set("enabled", enabled)
	transport := importString(source["transport"])
	if transport == "" {
		transport = importString(source["type"])
	}
	address := importString(source["url"])
	httpish := transport == "streamable-http" || transport == "http" || transport == "sse" || (address != "" && transport != "stdio")
	if httpish {
		set("transport", "streamable-http")
		set("url", address)
		if transport != "streamable-http" {
			warnings = append(warnings, "HTTP/SSE alias or URL inferred as streamable-http")
		}
	} else {
		set("transport", "stdio")
		set("command", importString(source["command"]))
	}
	if value, ok := importBool(source["failOnStartupError"]); ok {
		set("failOnStartupError", value)
	}
	if value, ok := importPositive(source["toolCallTimeoutMs"]); ok && math.Trunc(value) == value {
		record["toolCallTimeoutMs"] = json.RawMessage(strconv.FormatFloat(value, 'f', 0, 64))
	}
	var reconnect map[string]json.RawMessage
	if json.Unmarshal(source["reconnect"], &reconnect) == nil && reconnect != nil {
		retained := map[string]json.RawMessage{}
		if value, ok := importBool(reconnect["enabled"]); ok {
			retained["enabled"], _ = json.Marshal(value)
		}
		for _, key := range []string{"initialDelayMs", "maxDelayMs", "maxAttempts"} {
			if _, ok := importPositive(reconnect[key]); ok {
				retained[key] = reconnect[key]
			}
		}
		if len(retained) > 0 {
			set("reconnect", retained)
		}
	}
	mapField := "env"
	if httpish {
		mapField = "headers"
	}
	var pairs map[string]json.RawMessage
	if json.Unmarshal(source[mapField], &pairs) == nil && pairs != nil {
		values := map[string]string{}
		for key, raw := range pairs {
			var value string
			if string(raw) != "null" && json.Unmarshal(raw, &value) == nil {
				values[key] = value
			} else {
				warnings = append(warnings, mapField+": ignored non-string value for "+key)
			}
		}
		if len(values) > 0 {
			set(mapField, values)
		}
	}
	if !httpish {
		args := []string{}
		var array []json.RawMessage
		if json.Unmarshal(source["args"], &array) == nil && array != nil {
			for _, raw := range array {
				var value string
				if string(raw) != "null" && json.Unmarshal(raw, &value) == nil && value != "" {
					args = append(args, value)
				} else {
					warnings = append(warnings, "Ignored empty or non-string argument, matching the original importer")
				}
			}
		} else if value := importString(source["args"]); value != "" {
			args = strings.Fields(value)
			warnings = append(warnings, "String arguments split on whitespace, matching the original importer")
		}
		if len(args) > 0 {
			set("args", args)
		}
		if cwd := strings.TrimSpace(importString(source["cwd"])); cwd != "" {
			set("cwd", cwd)
		}
	}
	result, _ := json.Marshal(record)
	return result, warnings
}
func importPositive(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var value float64
	if json.Unmarshal(raw, &value) != nil || math.IsInf(value, 0) || math.IsNaN(value) || value <= 0 {
		return 0, false
	}
	return value, true
}
