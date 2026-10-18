package startuprec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Plugin struct {
	Name                      string
	Enabled, Mutable, Suspect bool
	Removable                 bool
	Restriction               string
}
type Evidence struct {
	Name, Line     string
	Known, Mutable bool
}
type Diagnostics struct {
	Kind                      string
	Evidence                  []Evidence
	Plugins                   []Plugin
	RepairNames               []string
	InventoryReady, CanRepair bool
	Receipt                   string
}

var evidencePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)failed to (?:apply|import) loader entry [^()\r\n]+ \((@?[\w.-]+(?:/[\w.-]+)*)\)`),
	regexp.MustCompile(`(?i)cannot resolve profile bundle ['"]([^'"]+)['"]`),
	regexp.MustCompile(`(?i)cannot find package ['"]([^'"]+)['"]`),
	regexp.MustCompile(`(?i)err_module_not_found[^\n'"]*['"]([^'"]+)['"]`),
	regexp.MustCompile(`(?i)failed to compose[^\n]*['"](@?[\w./-]+)['"]`),
}
var bootAuditHeader = regexp.MustCompile(`(?i)\bweb boot:\s+([1-9][0-9]*) entr(?:y|ies) did not activate\b`)
var bootAuditEntry = regexp.MustCompile(`(?i)^\s*(?:\[(?:app|error|dsh)\]\s*)?(@?[\w.-]+(?:/[\w.-]+)*): (?:import failed.*|pending \(waiting for services?.*|failed|disposed|loading|unloading)\s*$`)
var packageName = regexp.MustCompile(`^(@[a-zA-Z0-9_.-]+/)?[a-zA-Z0-9_.-]+$`)
var builtinNames = map[string]bool{
	"@deepseek-ai/dsh-client-ui-agents-panel": true, "@deepseek-ai/dsh-client-ui-diff": true,
	"@deepseek-ai/dsh-client-ui-files": true, "@deepseek-ai/dsh-client-ui-git": true,
	"@deepseek-ai/dsh-client-ui-message-edit": true, "@deepseek-ai/dsh-client-ui-preview": true,
	"@deepseek-ai/dsh-client-ui-settings-mcp": true, "@deepseek-ai/dsh-client-ui-settings-remote": true,
	"@deepseek-ai/dsh-client-ui-settings-market": true, "@deepseek-ai/dsh-client-ui-settings-skills": true,
	"@deepseek-ai/dsh-client-ui-surfaces": true, "@deepseek-ai/dsh-client-ui-titlebar": true,
	"@deepseek-ai/dsh-client-ui-user-terminal": true, "@deepseek-ai/dsh-host-mcp-servers": true,
	"@deepseek-ai/dsh-host-skill-inventory": true, "@deepseek-ai/dsh-llm-vision-fallback": true,
	"@deepseek-ai/dsh-mcp-servers-file": true, "@deepseek-ai/dsh-client-ui-directory-picker-browse": true,
	"@deepseek-ai/dsh-host-directory-picker-browse": true, "dsh-usage-panel": true,
	"@xmanrui/dsh-im": true, "dsh-im": true, "xmanrui-dsh-im": true,
}

func genericCause(text string) string {
	lower := strings.ToLower(text)
	for _, c := range []struct {
		kind  string
		words []string
	}{
		{"oom", []string{"heap out of memory", "js heap", "allocation failed"}},
		{"port-excluded", []string{"listen eacces"}}, {"port-in-use", []string{"eaddrinuse", "address already in use"}},
		{"missing-node", []string{"cannot find node", "node is not recognized", "node' is not recognized", "node.exe: no such file"}},
	} {
		for _, word := range c.words {
			if strings.Contains(lower, word) {
				return c.kind
			}
		}
	}
	if strings.Contains(lower, "session_projcache") && strings.Contains(lower, "does not match its schema") {
		return "session-cache"
	}
	return ""
}
func isBuiltin(name string) bool {
	if strings.Contains(strings.ReplaceAll(name, `\`, "/"), "desktop-plugins/install-dsh-plugin") {
		return true
	}
	for pkg := range builtinNames {
		if name == pkg || strings.HasPrefix(name, pkg+"/") {
			return true
		}
	}
	return false
}

// Diagnose uses only this failure's bounded evidence and a generation-bound
// Host inventory. Unknown names and core damage can never become repair targets.
func Diagnose(logs []string, reason string, plugins []Plugin, inventoryReady bool) Diagnostics {
	d := Diagnostics{InventoryReady: inventoryReady, Plugins: append([]Plugin(nil), plugins...)}
	rows := map[string]Plugin{}
	for _, row := range d.Plugins {
		rows[row.Name] = row
	}
	lines := append(SanitizeTail(logs), RedactLine(reason))
	d.Kind = genericCause(strings.Join(lines, "\n"))
	seen := map[string]bool{}
	suspects := map[string]bool{}
	unsafe := false
	runtimeDamage := false
	add := func(raw, line string) {
		name := raw
		if _, ok := rows[name]; !ok && strings.HasSuffix(name, "/client") {
			if _, exists := rows[strings.TrimSuffix(name, "/client")]; exists {
				name = strings.TrimSuffix(name, "/client")
			}
		}
		key := name + "\x00" + line
		if seen[key] {
			return
		}
		seen[key] = true
		row, known := rows[name]
		mutable := known && row.Mutable && packageName.MatchString(name)
		d.Evidence = append(d.Evidence, Evidence{Name: name, Line: line, Known: known, Mutable: mutable})
		if !known && isBuiltin(name) {
			runtimeDamage = true
		}
		if !known || !mutable {
			unsafe = true
		}
		if known && mutable && row.Enabled {
			suspects[name] = true
		}
	}
	remaining := 0
	for _, line := range lines {
		if m := bootAuditHeader.FindStringSubmatch(line); len(m) > 1 {
			remaining, _ = strconv.Atoi(m[1])
		} else if remaining > 0 {
			if m := bootAuditEntry.FindStringSubmatch(line); len(m) > 1 {
				add(m[1], line)
				remaining--
			} else {
				remaining = 0
			}
		}
		for _, pattern := range evidencePatterns {
			for _, m := range pattern.FindAllStringSubmatch(line, -1) {
				add(m[1], line)
			}
		}
	}
	if d.Kind == "" {
		switch {
		case runtimeDamage:
			d.Kind = "runtime"
		case len(d.Evidence) > 0:
			d.Kind = "plugins"
		default:
			d.Kind = "unknown"
		}
	}
	for i := range d.Plugins {
		d.Plugins[i].Suspect = suspects[d.Plugins[i].Name]
	}
	if d.Kind == "plugins" && !unsafe {
		for name := range suspects {
			d.RepairNames = append(d.RepairNames, name)
		}
		sort.Strings(d.RepairNames)
	}
	d.CanRepair = inventoryReady && len(d.RepairNames) > 0
	raw, _ := json.Marshal(d)
	sum := sha256.Sum256(raw)
	d.Receipt = hex.EncodeToString(sum[:])
	return d
}

// Repair runs inside the existing Host task-protection admission. Check must
// revalidate the runtime/import boundary before each write and before restart.
type RepairActions struct {
	Check         func() error
	Disable       func(name string) error
	ClearRecovery func() error
	Restart       func() error
}

func Repair(names []string, a RepairActions) ([]string, error) {
	if len(names) == 0 || a.Check == nil || a.Disable == nil || a.ClearRecovery == nil || a.Restart == nil {
		return nil, errors.New("incomplete targeted repair")
	}
	var done []string
	for _, name := range names {
		if !packageName.MatchString(name) {
			return done, errors.New("invalid plugin identity")
		}
		if err := a.Check(); err != nil {
			return done, err
		}
		if err := a.Disable(name); err != nil {
			return done, fmt.Errorf("%s: %w", name, err)
		}
		done = append(done, name)
	}
	if err := a.Check(); err != nil {
		return done, err
	}
	if err := a.ClearRecovery(); err != nil {
		return done, err
	}
	if err := a.Restart(); err != nil {
		return done, err
	}
	return done, nil
}
