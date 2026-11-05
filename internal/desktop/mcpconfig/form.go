package mcpconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"whaleisle/internal/dsh/api"
)

// FormDraft is an unsaved projection of the real stdio/streamable-http record.
// base retains unknown/advanced JSON fields without converting their numbers.
type FormDraft struct {
	ID, ServerName, Transport, Command, ArgsJSON, Cwd, URL, Timeout, Env, Headers string
	Enabled                                                                       bool
	base                                                                          string
}

var mcpFormName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
var mcpFormTimeout = regexp.MustCompile(`^[1-9][0-9]*$`)
var mcpPairKey = regexp.MustCompile(`^[^=\s]+$`)

const mcpSecretMask = "********"

// NewForm starts the Add action without reusing an older edit/import draft.
func (c *Controller) NewForm() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Busy {
		return
	}
	c.cancelImportDraft()
	form, _ := parseForm("")
	c.state.Form, c.state.FormErrors = &form, nil
	c.state.EditorOpen = true
	c.state.Draft, c.state.EditingID, c.state.Err = "", "", ""
	c.revision++
}

// OpenFormFor keeps the existing server identity and advanced JSON fields.
func (c *Controller) OpenFormFor(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Busy || !c.writable(id) {
		return
	}
	for _, row := range c.state.Rows {
		if row.Id != id {
			continue
		}
		form, err := parseForm(string(row.Spec))
		if err != nil {
			c.cancelImportDraft()
			c.state.Form, c.state.FormErrors = nil, nil
			c.state.EditorOpen = true
			c.state.Draft, c.state.EditingID = string(row.Spec), id
			c.state.Err = err.Error()
			c.revision++
			return
		}
		c.cancelImportDraft()
		c.state.Form, c.state.FormErrors = &form, nil
		c.state.EditorOpen = true
		c.state.Draft, c.state.EditingID, c.state.Err = string(row.Spec), id, ""
		c.revision++
		return
	}
}

func (c *Controller) OpenForm() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Busy || c.state.Form != nil {
		return
	}
	form, err := parseForm(c.state.Draft)
	if err != nil {
		c.state.Err = err.Error()
		return
	}
	c.state.Form = &form
	c.state.EditorOpen = true
	c.state.FormErrors = nil
	c.state.Err = ""
	c.revision++
}
func (c *Controller) SetForm(form FormDraft) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Form == nil {
		return
	}
	form.base = c.state.Form.base
	if form != *c.state.Form {
		c.state.Form = &form
		c.state.FormErrors = nil
		c.state.Err = ""
		c.revision++
	}
}
func (c *Controller) ShowJSON() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Form == nil || c.state.Busy {
		return
	}
	raw, _ := formSpecMode(*c.state.Form, c.state.EditingID, c.state.Rows, false)
	c.state.Draft = string(raw)
	c.state.Form = nil
	c.state.EditorOpen = true
	c.state.FormErrors = nil
	c.state.Err = ""
	c.revision++
}

// SaveFormExpected uses the existing complete-spec upsert lifecycle, with a
// version guard so a queued save cannot publish a newer or different form.
func (c *Controller) SaveFormExpected(ctx context.Context, caller Caller, form FormDraft, editingID string) {
	c.mu.Lock()
	if c.state.Form == nil || *c.state.Form != form || c.state.EditingID != editingID || c.state.Busy {
		c.mu.Unlock()
		return
	}
	raw, issues := formSpec(form, editingID, c.state.Rows)
	if len(issues) > 0 {
		c.state.FormErrors = issues
		c.state.Err = "Fix the highlighted form fields"
		c.mu.Unlock()
		return
	}
	c.state.Draft = string(raw)
	c.state.FormErrors = nil
	c.revision++
	revision := c.revision
	c.mu.Unlock()
	c.saveExpectedVersion(ctx, caller, string(raw), editingID, &revision)
}

func parseForm(text string) (FormDraft, error) {
	form := FormDraft{Enabled: true, Transport: "stdio", ArgsJSON: "[]", base: text}
	if strings.TrimSpace(text) == "" {
		form.base = "{}"
		return form, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &raw); err != nil || raw == nil {
		return form, errors.New("Enter one server JSON object before opening the form")
	}
	readString := func(key string, target *string) error {
		value, ok := raw[key]
		if !ok {
			return nil
		}
		if string(value) == "null" || json.Unmarshal(value, target) != nil {
			return fmt.Errorf("%s must be a string; use the JSON editor to correct it", key)
		}
		return nil
	}
	for _, field := range []struct {
		key    string
		target *string
	}{{"id", &form.ID}, {"serverName", &form.ServerName}, {"transport", &form.Transport}, {"command", &form.Command}, {"cwd", &form.Cwd}, {"url", &form.URL}} {
		if err := readString(field.key, field.target); err != nil {
			return form, err
		}
	}
	if form.ServerName == "" {
		form.ServerName = form.ID
	}
	if form.Transport != "stdio" && form.Transport != "streamable-http" {
		return form, errors.New("Transport must be stdio or streamable-http")
	}
	if value, ok := raw["enabled"]; ok {
		if string(value) == "null" || json.Unmarshal(value, &form.Enabled) != nil {
			return form, errors.New("enabled must be a boolean")
		}
	}
	if value, ok := raw["toolCallTimeoutMs"]; ok {
		var number json.Number
		if string(value) == "null" || json.Unmarshal(value, &number) != nil {
			return form, errors.New("toolCallTimeoutMs must be a number")
		}
		form.Timeout = string(number)
	}
	if value, ok := raw["args"]; ok {
		var args []string
		if string(value) == "null" || json.Unmarshal(value, &args) != nil {
			return form, errors.New("args must be a JSON string array")
		}
		encoded, _ := json.Marshal(args)
		form.ArgsJSON = string(encoded)
	}
	for _, field := range []struct {
		key    string
		target *string
	}{{"env", &form.Env}, {"headers", &form.Headers}} {
		value, ok := raw[field.key]
		if !ok {
			continue
		}
		var pairs map[string]string
		if string(value) == "null" || json.Unmarshal(value, &pairs) != nil {
			return form, fmt.Errorf("%s must be a string map", field.key)
		}
		keys := make([]string, 0, len(pairs))
		for key := range pairs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		lines := make([]string, 0, len(keys))
		for _, key := range keys {
			if !mcpPairKey.MatchString(key) || strings.ContainsAny(pairs[key], "\r\n") {
				return form, fmt.Errorf("%s contains a key or multiline value that requires the JSON editor", field.key)
			}
			lines = append(lines, key+"="+pairs[key])
		}
		*field.target = strings.Join(lines, "\n")
	}
	return form, nil
}

func formSpec(form FormDraft, editingID string, rows []api.McpServerSnapshotServersItem) (json.RawMessage, map[string]string) {
	return formSpecMode(form, editingID, rows, true)
}

// A mode switch serializes an unfinished draft; only Save requires a complete
// server. Invalid scalar text remains visible in JSON for correction.
func formSpecMode(form FormDraft, editingID string, rows []api.McpServerSnapshotServersItem, validate bool) (json.RawMessage, map[string]string) {
	issues := map[string]string{}
	if !mcpFormName.MatchString(form.ID) {
		issues["id"] = "Use 1–32 letters, digits, underscores or hyphens"
	}
	if !mcpFormName.MatchString(form.ServerName) {
		issues["serverName"] = "Use 1–32 letters, digits, underscores or hyphens"
	}
	if editingID != "" && form.ID != editingID {
		issues["id"] = "Keep the existing server ID when editing"
	}
	var existing map[string]json.RawMessage
	for _, row := range rows {
		if row.Id != form.ID {
			continue
		}
		if editingID == "" {
			issues["id"] = "This ID already exists; use its Edit action"
		}
		if !row.Writable {
			issues["id"] = "Server is read-only"
		}
		if row.Id == editingID {
			_ = json.Unmarshal(row.Spec, &existing)
		}
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal([]byte(form.base), &raw) != nil || raw == nil {
		issues["form"] = "Original JSON is unavailable"
		return nil, issues
	}
	set := func(key string, value any) { data, _ := json.Marshal(value); raw[key] = data }
	set("id", form.ID)
	set("serverName", form.ServerName)
	set("enabled", form.Enabled)
	set("transport", form.Transport)
	delete(raw, "toolCallTimeoutMs")
	timeout := strings.TrimSpace(form.Timeout)
	if timeout != "" {
		value, err := strconv.ParseUint(timeout, 10, 64)
		if !mcpFormTimeout.MatchString(timeout) || err != nil || value > 9007199254740991 {
			issues["timeout"] = "Enter a positive integer in milliseconds"
			set("toolCallTimeoutMs", form.Timeout)
		} else {
			set("toolCallTimeoutMs", value)
		}
	}
	for _, key := range []string{"command", "args", "cwd", "env", "url", "headers"} {
		delete(raw, key)
	}
	var oldTransport string
	_ = json.Unmarshal(existing["transport"], &oldTransport)
	pairs := func(field, text string) map[string]string {
		var old map[string]string
		if editingID == form.ID && oldTransport == form.Transport {
			_ = json.Unmarshal(existing[field], &old)
		}
		values := map[string]string{}
		for i, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			key, value, found := strings.Cut(line, "=")
			key = strings.TrimSpace(key)
			if !found || !mcpPairKey.MatchString(key) {
				issues[field] = fmt.Sprintf("Line %d must be KEY=value without spaces in the key", i+1)
				continue
			}
			if _, duplicate := values[key]; duplicate {
				issues[field] = fmt.Sprintf("Duplicate key %s", key)
				continue
			}
			if strings.TrimSpace(value) == mcpSecretMask {
				if old[key] != mcpSecretMask {
					issues[field] = "A hidden-value marker is only valid for the original server and key"
					continue
				}
				// Host mergeSecretMap interprets an empty value for the existing key as
				// keep-existing. Never send the display marker as a new credential.
				value = ""
			}
			values[key] = value
		}
		return values
	}
	switch form.Transport {
	case "stdio":
		if strings.TrimSpace(form.Command) == "" {
			issues["command"] = "Command is required"
		}
		set("command", strings.TrimSpace(form.Command))
		if strings.TrimSpace(form.ArgsJSON) != "" {
			var args []string
			if strings.TrimSpace(form.ArgsJSON) == "null" || json.Unmarshal([]byte(form.ArgsJSON), &args) != nil {
				issues["args"] = "Enter a JSON array of strings, for example [\"--flag\", \"path with spaces\"]"
				set("args", form.ArgsJSON)
			} else if len(args) > 0 {
				set("args", args)
			}
		}
		if strings.TrimSpace(form.Cwd) != "" {
			set("cwd", strings.TrimSpace(form.Cwd))
		}
		if env := pairs("env", form.Env); len(env) > 0 {
			set("env", env)
		}
		if !validate && issues["env"] != "" {
			set("env", form.Env)
		}
	case "streamable-http":
		address := strings.TrimSpace(form.URL)
		parsed, err := url.Parse(address)
		if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
			issues["url"] = "Enter a valid http:// or https:// URL"
		}
		set("url", address)
		if headers := pairs("headers", form.Headers); len(headers) > 0 {
			set("headers", headers)
		}
		if !validate && issues["headers"] != "" {
			set("headers", form.Headers)
		}
	default:
		issues["transport"] = "Transport must be stdio or streamable-http"
	}
	if validate && len(issues) > 0 {
		return nil, issues
	}
	result, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		issues["form"] = err.Error()
		return nil, issues
	}
	return result, nil
}
