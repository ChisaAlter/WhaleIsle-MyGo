package pluginmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"whaleisle/internal/dsh/api"
)

var nativeMCPName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
var nativeSkillName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// normalizeMCP owns the external-config to Host-spec conversion. Unknown input
// is rejected rather than losing environment, transport, or invocation options.
func normalizeMCP(raw json.RawMessage, enabled *bool) (json.RawMessage, error) {
	var spec map[string]json.RawMessage
	if json.Unmarshal(raw, &spec) != nil || spec == nil {
		return nil, errors.New("MCP server must be an object")
	}
	set := func(key string, value any) { spec[key], _ = json.Marshal(value) }
	id := decodeJSONString(spec["id"])
	if id == "" {
		id = decodeJSONString(spec["name"])
		set("id", id)
	}
	name := decodeJSONString(spec["serverName"])
	if name == "" {
		name = id
		set("serverName", name)
	}
	if !nativeMCPName.MatchString(id) || !nativeMCPName.MatchString(name) {
		return nil, fmt.Errorf("MCP ID and serverName must contain 1–32 letters, digits, _ or -")
	}
	if enabled != nil {
		set("enabled", *enabled)
	} else if _, ok := spec["enabled"]; !ok {
		if disabled, ok := spec["disabled"]; ok {
			var value bool
			if err := json.Unmarshal(disabled, &value); err != nil {
				return nil, fmt.Errorf("MCP %s: disabled must be boolean", id)
			}
			set("enabled", !value)
		} else {
			set("enabled", false)
		}
	}
	var active bool
	if err := json.Unmarshal(spec["enabled"], &active); err != nil || string(spec["enabled"]) == "null" {
		return nil, fmt.Errorf("MCP %s: enabled must be boolean", id)
	}
	transport := decodeJSONString(spec["transport"])
	if transport == "" {
		transport = decodeJSONString(spec["type"])
	}
	if transport == "" {
		if decodeJSONString(spec["url"]) != "" {
			transport = "streamable-http"
		} else {
			transport = "stdio"
		}
	}
	if transport == "http" {
		transport = "streamable-http"
	}
	if transport != "stdio" && transport != "streamable-http" {
		return nil, fmt.Errorf("MCP %s: transport %q is not supported by this Host", id, transport)
	}
	set("transport", transport)
	delete(spec, "type")
	delete(spec, "name")
	delete(spec, "disabled")
	allowed := map[string]bool{"id": true, "serverName": true, "enabled": true, "transport": true, "toolCallTimeoutMs": true, "failOnStartupError": true, "reconnect": true}
	if transport == "stdio" {
		for _, key := range []string{"command", "args", "env", "cwd"} {
			allowed[key] = true
		}
		if decodeJSONString(spec["command"]) == "" {
			return nil, fmt.Errorf("MCP %s: command is required", id)
		}
		if raw, ok := spec["args"]; ok {
			var args []string
			if string(raw) == "null" || json.Unmarshal(raw, &args) != nil {
				return nil, fmt.Errorf("MCP %s: args must be a string array", id)
			}
		}
		if raw, ok := spec["cwd"]; ok {
			var cwd string
			if json.Unmarshal(raw, &cwd) != nil {
				return nil, fmt.Errorf("MCP %s: cwd must be a string", id)
			}
		}
	} else {
		allowed["url"], allowed["headers"] = true, true
		u, err := url.Parse(decodeJSONString(spec["url"]))
		if err != nil || u.Host == "" || u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("MCP %s: HTTP URL is required", id)
		}
	}
	for key, raw := range spec {
		if !allowed[key] {
			return nil, fmt.Errorf("MCP %s: unsupported field %q; use MCP Settings to review it", id, key)
		}
		if key == "env" || key == "headers" {
			var values map[string]string
			if string(raw) == "null" || json.Unmarshal(raw, &values) != nil {
				return nil, fmt.Errorf("MCP %s: %s values must be strings", id, key)
			}
		}
		if key == "failOnStartupError" {
			var value bool
			if json.Unmarshal(raw, &value) != nil {
				return nil, fmt.Errorf("MCP %s: failOnStartupError must be boolean", id)
			}
		}
		if key == "toolCallTimeoutMs" {
			var value float64
			if json.Unmarshal(raw, &value) != nil || value <= 0 || math.Trunc(value) != value {
				return nil, fmt.Errorf("MCP %s: toolCallTimeoutMs must be a positive integer", id)
			}
		}
		if key == "reconnect" {
			var values map[string]json.RawMessage
			if json.Unmarshal(raw, &values) != nil || values == nil {
				return nil, fmt.Errorf("MCP %s: reconnect must be an object", id)
			}
			for k, v := range values {
				if k == "enabled" {
					var b bool
					if json.Unmarshal(v, &b) != nil {
						return nil, fmt.Errorf("MCP %s: reconnect.enabled must be boolean", id)
					}
				} else if k == "initialDelayMs" || k == "maxDelayMs" || k == "maxAttempts" {
					var n float64
					if json.Unmarshal(v, &n) != nil || n < 0 || math.Trunc(n) != n {
						return nil, fmt.Errorf("MCP %s: invalid reconnect.%s", id, k)
					}
				} else {
					return nil, fmt.Errorf("MCP %s: unsupported reconnect.%s", id, k)
				}
			}
		}
	}
	return json.Marshal(spec)
}

func nativeAction(op action) bool {
	return op.kind == "install" && (len(op.mcpSpecs) > 0 || len(op.localSkills) > 0)
}

func prepareNativeAction(op *action) error {
	ids, namespaces := map[string]bool{}, map[string]bool{}
	for i, raw := range op.mcpSpecs {
		normalized, err := normalizeMCP(raw, op.options.Enabled)
		if err != nil {
			return err
		}
		var spec struct {
			ID   string `json:"id"`
			Name string `json:"serverName"`
		}
		_ = json.Unmarshal(normalized, &spec)
		if ids[spec.ID] || namespaces[spec.Name] {
			return fmt.Errorf("duplicate MCP server identity: %s", spec.ID)
		}
		ids[spec.ID], namespaces[spec.Name] = true, true
		op.mcpSpecs[i] = normalized
	}
	seenSkills := map[string]bool{}
	for _, skill := range op.localSkills {
		if !nativeSkillName.MatchString(skill.Name) || len(skill.Name) > 64 {
			return fmt.Errorf("skill name must be kebab-case: %s", skill.Name)
		}
		if seenSkills[skill.Name] {
			return fmt.Errorf("duplicate skill name: %s", skill.Name)
		}
		seenSkills[skill.Name] = true
	}
	return nil
}

// Existing records are checked before any write. This install path never
// overwrites a configured MCP server or an installed skill with the same name.
func (c Client) preflightNative(ctx context.Context, op action) error {
	if len(op.mcpSpecs) > 0 {
		var caps struct {
			CreateOnly bool `json:"createOnly"`
		}
		if err := c.Caller.Call(ctx, "mygoMCP/capabilities", nil, &caps); err != nil {
			return fmt.Errorf("MCP import is unavailable on this Host: %w", err)
		}
		if !caps.CreateOnly {
			return errors.New("this Host does not support create-only MCP import")
		}
		rows, err := c.MCPServers(ctx)
		if err != nil {
			return err
		}
		for _, raw := range op.mcpSpecs {
			var incoming struct {
				ID   string `json:"id"`
				Name string `json:"serverName"`
			}
			_ = json.Unmarshal(raw, &incoming)
			for _, row := range rows.Servers {
				var existing struct {
					Name string `json:"serverName"`
				}
				_ = json.Unmarshal(row.Spec, &existing)
				if row.Id == incoming.ID || existing.Name == incoming.Name {
					return fmt.Errorf("MCP %s already exists; edit it in MCP Settings", incoming.ID)
				}
			}
		}
	}
	if len(op.localSkills) > 0 {
		var inventory api.SkillInventorySnapshot
		if err := c.Caller.Call(ctx, "skillInventory/list", map[string]any{"request": map[string]any{}}, &inventory); err != nil {
			return err
		}
		for _, skill := range op.localSkills {
			for _, row := range inventory.Skills {
				if row.Name == skill.Name {
					return fmt.Errorf("skill %s already exists; edit it in Skills", skill.Name)
				}
			}
		}
		for _, skill := range op.localSkills {
			if skill.Directory != "" {
				var caps struct {
					ImportDirectory bool `json:"importDirectory"`
				}
				if err := c.Caller.Call(ctx, "mygoSkills/capabilities", nil, &caps); err != nil {
					return fmt.Errorf("local skill import is unavailable on this Host: %w", err)
				}
				if !caps.ImportDirectory {
					return errors.New("this Host does not support local skill directories")
				}
				break
			}
		}
	}
	return nil
}

func failedNativeChange(op action, stage string, err error) Change {
	return Change{Application: "failed", Stage: stage, Target: op.target, Error: &ManagementError{Code: "native-install/" + stage, Diagnostic: err.Error()}}
}

func (c Client) applyNative(ctx context.Context, op action) (Change, error) {
	if err := c.preflightNative(ctx, op); err != nil {
		return failedNativeChange(op, "prepare", err), nil
	}
	preparedSkills := make([][]skillFile, len(op.localSkills))
	var preparedBytes int
	for i, skill := range op.localSkills {
		if skill.Directory == "" {
			continue
		}
		files, err := readSkillFiles(ctx, skill)
		if err != nil {
			return failedNativeChange(op, "read-skill", err), nil
		}
		preparedSkills[i] = files
		for _, file := range files {
			preparedBytes += len(file.Data)
		}
		if preparedBytes > 64<<20 {
			return failedNativeChange(op, "read-skill", errors.New("skill selection exceeds 64 MiB; import fewer directories together")), nil
		}
	}
	for _, spec := range op.mcpSpecs {
		if err := ctx.Err(); err != nil {
			return Change{}, err
		}
		if op.progress != nil {
			op.progress.startNext()
		}
		if err := c.CreateMCPServer(ctx, spec); err != nil {
			return Change{}, err
		}
		if op.progress != nil {
			op.progress.completeNext()
		}
	}
	for i, skill := range op.localSkills {
		if err := ctx.Err(); err != nil {
			return Change{}, err
		}
		if op.progress != nil {
			op.progress.startNext()
		}
		if skill.Directory != "" {
			request := map[string]any{"name": skill.Name, "requestId": op.options.RequestID, "files": preparedSkills[i]}
			if op.options.Enabled != nil {
				request["enabled"] = *op.options.Enabled
			}
			if err := c.Caller.Call(ctx, "mygoSkills/import", map[string]any{"request": request}, nil); err != nil {
				return Change{}, err
			}
		} else if err := c.CreateSkill(ctx, skill); err != nil {
			return Change{}, err
		}
		if op.progress != nil {
			op.progress.completeNext()
		}
	}
	return c.confirmNative(ctx, op)
}

func (c Client) confirmNative(ctx context.Context, op action) (Change, error) {
	var warnings []string
	if len(op.mcpSpecs) > 0 {
		rows, err := c.MCPServers(ctx)
		if err != nil {
			return Change{}, err
		}
		for _, wanted := range op.mcpSpecs {
			var spec map[string]any
			_ = json.Unmarshal(wanted, &spec)
			found := false
			for _, row := range rows.Servers {
				if row.Id != spec["id"] {
					continue
				}
				var actual map[string]any
				_ = json.Unmarshal(row.Spec, &actual)
				found = visibleMCPMatches(spec, actual)
				if found {
					if enabled, _ := spec["enabled"].(bool); !enabled {
						warnings = append(warnings, fmt.Sprintf("MCP %s saved disabled; enable it in MCP Settings", row.Id))
					} else if row.FiberPhase == "failed" || row.Connection != nil && row.Connection.Health == "failed" {
						warnings = append(warnings, fmt.Sprintf("MCP %s saved; connection failed, inspect MCP Settings", row.Id))
					} else if row.Connection == nil || row.Connection.Health != "connected" {
						warnings = append(warnings, fmt.Sprintf("MCP %s saved; connection is pending in MCP Settings", row.Id))
					}
				}
				break
			}
			if !found {
				return Change{}, fmt.Errorf("MCP %s installation is not confirmed; inspect MCP Settings before retrying", spec["id"])
			}
		}
	}
	for _, skill := range op.localSkills {
		var detail api.SkillInventoryDetail
		if skill.Directory != "" {
			var status struct {
				Installed bool                     `json:"installed"`
				Detail    api.SkillInventoryDetail `json:"detail"`
			}
			if err := c.Caller.Call(ctx, "mygoSkills/status", map[string]any{"request": map[string]string{"name": skill.Name, "requestId": op.options.RequestID}}, &status); err != nil {
				return Change{}, err
			}
			if !status.Installed {
				return Change{}, fmt.Errorf("skill %s installation is not confirmed", skill.Name)
			}
			detail = status.Detail
		} else if err := c.Caller.Call(ctx, "skillInventory/get", map[string]any{"request": api.SkillInventoryGetRequest{Name: skill.Name}}, &detail); err != nil {
			return Change{}, err
		}
		if detail.Name != skill.Name || strings.TrimSpace(detail.Content) != strings.TrimSpace(skill.Content) {
			return Change{}, fmt.Errorf("skill %s content was not confirmed", skill.Name)
		}
		wantModel, wantUser := skill.ModelInvocable, skill.UserInvocable
		if op.options.Enabled != nil && !*op.options.Enabled {
			wantModel, wantUser = false, false
		}
		if detail.ModelInvocable != wantModel || detail.UserInvocable != wantUser {
			return Change{}, fmt.Errorf("skill %s invocation settings were not confirmed", skill.Name)
		}
	}
	return Change{Changed: true, Application: "applied", Stage: "done", Target: op.target, Warnings: warnings}, nil
}

// The Host masks secret fields in list responses. Confirm only the visible
// configuration and the presence of those keys, never fetch or log secrets.
func visibleMCPMatches(wanted, actual map[string]any) bool {
	for key, want := range wanted {
		got, ok := actual[key]
		if !ok {
			return false
		}
		if key == "env" || key == "headers" {
			w, wok := want.(map[string]any)
			a, aok := got.(map[string]any)
			if !wok || !aok || len(w) != len(a) {
				return false
			}
			for k, v := range w {
				g, ok := a[k]
				if !ok || g != "********" && !reflect.DeepEqual(g, v) {
					return false
				}
			}
		} else if !reflect.DeepEqual(want, got) {
			return false
		}
	}
	return true
}

type skillFile struct {
	Path       string `json:"path"`
	Data       []byte `json:"data"`
	Executable bool   `json:"executable,omitempty"`
}

func readSkillFiles(ctx context.Context, skill LocalSkillManifest) ([]skillFile, error) {
	var files []skillFile
	var size int64
	root, err := filepath.EvalSymlinks(skill.Directory)
	if err != nil {
		return nil, err
	}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("skill directory contains a link: %s", rel)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("skill file is not regular: %s", rel)
		}
		if rel == ".mygo-import.json" {
			return errors.New("skill contains reserved import receipt")
		}
		size += info.Size()
		if size > 64<<20 || len(files) >= 4096 {
			return errors.New("skill directory exceeds 64 MiB or 4096 files")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if rel == "SKILL.md" {
			if !bytes.Equal(data, []byte(skill.Markdown)) {
				return errors.New("SKILL.md changed after inspection; inspect again")
			}
			if skill.ImportMarkdown != "" {
				data = []byte(skill.ImportMarkdown)
			}
		}
		files = append(files, skillFile{Path: filepath.ToSlash(rel), Data: data, Executable: info.Mode()&0111 != 0})
		return nil
	})
	return files, err
}
