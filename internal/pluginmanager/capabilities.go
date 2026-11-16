package pluginmanager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CapabilityKind classifies what runtime surface a plugin or package targets.
type CapabilityKind string

const (
	CapabilityUnknown   CapabilityKind = "unknown"
	CapabilityPureHost  CapabilityKind = "pure-host"
	CapabilityHybrid    CapabilityKind = "hybrid"
	CapabilityWebUIOnly CapabilityKind = "web-ui-only"
	CapabilityMCP       CapabilityKind = "mcp"
	CapabilitySkill     CapabilityKind = "skill"
)

// LocalSkillManifest holds a skill definition parsed from a local SKILL.md directory.
type LocalSkillManifest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content"`
}

// PackageCapabilities describes host/client/MCP/Skill capabilities extracted from
// package.json and companion files without executing package code or launching a browser.
type PackageCapabilities struct {
	Kind                CapabilityKind       `json:"kind,omitempty"`
	NativeCompatible    bool                 `json:"nativeCompatible"`
	CompatibilityLabel  string               `json:"compatibilityLabel,omitempty"`
	Guidance            string               `json:"guidance,omitempty"`
	HasHostContribution bool                 `json:"hasHostContribution,omitempty"`
	HasWebClient        bool                 `json:"hasWebClient,omitempty"`
	HasBundle           bool                 `json:"hasBundle,omitempty"`
	HasMCP              bool                 `json:"hasMcp,omitempty"`
	HasSkill            bool                 `json:"hasSkill,omitempty"`
	MCPServers          []string             `json:"mcpServers,omitempty"`
	SkillNames          []string             `json:"skillNames,omitempty"`
	MCPServerSpecs      []json.RawMessage    `json:"-"`
	LocalSkills         []LocalSkillManifest `json:"-"`
}

// CompatibilityLabelFor returns a human-readable native compatibility badge label.
func CompatibilityLabelFor(kind CapabilityKind) string {
	switch kind {
	case CapabilityPureHost:
		return "纯原生可用 (Pure Host)"
	case CapabilityHybrid:
		return "原生兼容 + 可选 Web (Hybrid)"
	case CapabilityMCP:
		return "MCP 服务 (原生支持)"
	case CapabilitySkill:
		return "Skill 技能包 (原生支持)"
	case CapabilityWebUIOnly:
		return "仅 Web 界面 (需兼容工作台)"
	default:
		return "未探测"
	}
}

// GuidanceFor returns actionable Wave 1 capability guidance without triggering browsers.
func GuidanceFor(kind CapabilityKind) string {
	switch kind {
	case CapabilityPureHost:
		return "Supports headless/native DSH Host mode directly without WebView2."
	case CapabilityHybrid:
		return "Host features work natively; optional browser panels require the explicit A2 compatibility workbench."
	case CapabilityMCP:
		return "MCP server package (no dsh.bundle required): import its server definition via MCP Settings or local manifest."
	case CapabilitySkill:
		return "Skill package (SKILL.md / dsh.skills, no dsh.bundle required): install via native Skills / Skill Hub."
	case CapabilityWebUIOnly:
		return "Web UI-only plugin (dsh.client / ./client without Host entry): open via explicit A2 compatibility workbench when needed."
	default:
		return ""
	}
}

// ClassifyPackageManifestJSON probes package.json bytes plus optional sibling SKILL.md / .mcp.json flags.
func ClassifyPackageManifestJSON(raw []byte, hasSkillMD, hasMCPFile bool) PackageCapabilities {
	caps := PackageCapabilities{HasSkill: hasSkillMD, HasMCP: hasMCPFile}
	if len(raw) > 0 {
		var manifest map[string]json.RawMessage
		if json.Unmarshal(raw, &manifest) == nil {
			applyPackageManifestSignals(&caps, manifest)
		}
	}
	finalizeCapabilities(&caps)
	return caps
}

func applyPackageManifestSignals(caps *PackageCapabilities, manifest map[string]json.RawMessage) {
	mainEntry := strings.TrimSpace(decodeJSONString(manifest["main"]))
	hasCordis := len(manifest["cordis"]) > 0 && string(manifest["cordis"]) != "null"

	hasClientExport, hasRootExport := inspectExports(manifest["exports"])
	if hasClientExport {
		caps.HasWebClient = true
	}

	if mainEntry != "" {
		if looksLikeClientPath(mainEntry) && !hasRootExport {
			caps.HasWebClient = true
		} else {
			caps.HasHostContribution = true
		}
	}
	if hasRootExport || hasCordis {
		caps.HasHostContribution = true
	}

	if dshRaw := manifest["dsh"]; len(dshRaw) > 0 {
		var dshObj map[string]json.RawMessage
		if json.Unmarshal(dshRaw, &dshObj) == nil {
			inspectDSHManifestSection(caps, dshObj)
		}
	}

	if mcpRaw := manifest["mcpServers"]; len(mcpRaw) > 0 {
		names, specs := extractMCPServers(mcpRaw)
		if len(names) > 0 || jsonTruthyNonEmpty(mcpRaw) {
			caps.HasMCP = true
			caps.MCPServers = mergeUniqueStrings(caps.MCPServers, names)
			caps.MCPServerSpecs = append(caps.MCPServerSpecs, specs...)
		}
	}
	if skillsRaw := manifest["skills"]; len(skillsRaw) > 0 {
		names := extractStringListOrKeys(skillsRaw)
		if len(names) > 0 || jsonTruthyNonEmpty(skillsRaw) {
			caps.HasSkill = true
			caps.SkillNames = mergeUniqueStrings(caps.SkillNames, names)
		}
	}
}

func inspectDSHManifestSection(caps *PackageCapabilities, dshObj map[string]json.RawMessage) {
	if clientRaw, ok := dshObj["client"]; ok && jsonTruthyNonEmpty(clientRaw) {
		caps.HasWebClient = true
	}
	if bundleRaw, ok := dshObj["bundle"]; ok && jsonTruthyNonEmpty(bundleRaw) {
		caps.HasBundle = true
		if !bundleClientOnly(bundleRaw) {
			caps.HasHostContribution = true
		} else {
			caps.HasWebClient = true
		}
	}
	for _, key := range []string{"host", "entries", "contributions", "profile", "services"} {
		if raw, ok := dshObj[key]; ok && jsonTruthyNonEmpty(raw) {
			caps.HasHostContribution = true
		}
	}
	for _, key := range []string{"mcp", "mcpServers"} {
		if raw, ok := dshObj[key]; ok && jsonTruthyNonEmpty(raw) {
			names, specs := extractMCPServers(raw)
			caps.HasMCP = true
			caps.MCPServers = mergeUniqueStrings(caps.MCPServers, names)
			caps.MCPServerSpecs = append(caps.MCPServerSpecs, specs...)
		}
	}
	for _, key := range []string{"skills", "skill"} {
		if raw, ok := dshObj[key]; ok && jsonTruthyNonEmpty(raw) {
			caps.HasSkill = true
			caps.SkillNames = mergeUniqueStrings(caps.SkillNames, extractStringListOrKeys(raw))
		}
	}
}

func bundleClientOnly(raw json.RawMessage) bool {
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return false
	}
	target := strings.ToLower(strings.TrimSpace(decodeJSONString(obj["surface"])))
	if target == "web" || target == "client" || target == "browser" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(decodeJSONString(obj["target"])), "client")
}

func inspectExports(raw json.RawMessage) (hasClient, hasRoot bool) {
	if len(raw) == 0 {
		return false, false
	}
	var literal string
	if json.Unmarshal(raw, &literal) == nil {
		if looksLikeClientPath(literal) {
			return true, false
		}
		return false, strings.TrimSpace(literal) != ""
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return false, false
	}
	for key := range obj {
		k := strings.TrimSpace(key)
		switch {
		case k == "./client" || k == "./client.js" || strings.HasPrefix(k, "./client/"):
			hasClient = true
		case k == "." || k == "./host" || k == "./plugin":
			hasRoot = true
		}
	}
	return hasClient, hasRoot
}

func finalizeCapabilities(caps *PackageCapabilities) {
	switch {
	case caps.HasHostContribution && caps.HasWebClient:
		caps.Kind = CapabilityHybrid
	case caps.HasHostContribution:
		caps.Kind = CapabilityPureHost
	case caps.HasWebClient:
		caps.Kind = CapabilityWebUIOnly
	case caps.HasMCP:
		caps.Kind = CapabilityMCP
	case caps.HasSkill:
		caps.Kind = CapabilitySkill
	default:
		caps.Kind = CapabilityUnknown
	}
	caps.NativeCompatible = caps.Kind == CapabilityPureHost ||
		caps.Kind == CapabilityHybrid ||
		caps.Kind == CapabilityMCP ||
		caps.Kind == CapabilitySkill
	caps.CompatibilityLabel = CompatibilityLabelFor(caps.Kind)
	caps.Guidance = GuidanceFor(caps.Kind)
}

// InspectLocalPackage inspects a local directory for package.json, SKILL.md, and .mcp.json.
func InspectLocalPackage(dir string) (PackageCapabilities, error) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return PackageCapabilities{}, err
	}
	var pkgRaw []byte
	if raw, err := os.ReadFile(filepath.Join(dir, "package.json")); err == nil && len(raw) <= 2<<20 {
		pkgRaw = raw
	}
	var mcpSpecs []json.RawMessage
	var mcpNames []string
	hasMCPFile := false
	for _, candidate := range []string{".mcp.json", "mcp.json"} {
		if raw, err := os.ReadFile(filepath.Join(dir, candidate)); err == nil && len(raw) <= 2<<20 {
			hasMCPFile = true
			names, specs := extractMCPServersFromConfigFile(raw)
			mcpNames = mergeUniqueStrings(mcpNames, names)
			mcpSpecs = append(mcpSpecs, specs...)
		}
	}
	skills := discoverLocalSkills(dir)
	hasSkillMD := len(skills) > 0
	caps := ClassifyPackageManifestJSON(pkgRaw, hasSkillMD, hasMCPFile)
	caps.MCPServers = mergeUniqueStrings(caps.MCPServers, mcpNames)
	caps.MCPServerSpecs = append(caps.MCPServerSpecs, mcpSpecs...)
	for _, sk := range skills {
		caps.SkillNames = mergeUniqueStrings(caps.SkillNames, []string{sk.Name})
	}
	caps.LocalSkills = skills
	return caps, nil
}

// InferCapabilityFromCatalogMetadata derives an initial capability profile from curated
// catalog fields before package.json details are fetched.
func InferCapabilityFromCatalogMetadata(category, id, pkgName, description string) PackageCapabilities {
	cat := strings.ToLower(strings.TrimSpace(category))
	lower := strings.ToLower(id + " " + pkgName + " " + description)
	var caps PackageCapabilities
	switch {
	case cat == "skill" || cat == "skills":
		caps.Kind = CapabilitySkill
		caps.HasSkill = true
	case cat == "ui" || cat == "theme" || cat == "themes":
		caps.Kind = CapabilityWebUIOnly
		caps.HasWebClient = true
	case cat == "mcp" || strings.Contains(lower, "mcp server") || strings.HasPrefix(strings.ToLower(pkgName), "mcp-"):
		caps.Kind = CapabilityMCP
		caps.HasMCP = true
	default:
		caps.Kind = CapabilityUnknown
	}
	caps.NativeCompatible = caps.Kind == CapabilityPureHost ||
		caps.Kind == CapabilityHybrid ||
		caps.Kind == CapabilityMCP ||
		caps.Kind == CapabilitySkill
	caps.CompatibilityLabel = CompatibilityLabelFor(caps.Kind)
	caps.Guidance = GuidanceFor(caps.Kind)
	return caps
}

func discoverLocalSkills(dir string) []LocalSkillManifest {
	var out []LocalSkillManifest
	addSkill := func(path, fallbackName string) {
		raw, err := os.ReadFile(path)
		if err != nil || len(raw) > 1<<20 {
			return
		}
		text := strings.TrimSpace(string(raw))
		if text == "" {
			return
		}
		name, desc := parseSkillFrontmatter(text)
		if name == "" {
			name = fallbackName
		}
		if name == "" {
			name = filepath.Base(dir)
		}
		if desc == "" {
			desc = name
		}
		out = append(out, LocalSkillManifest{Name: name, Description: desc, Content: text})
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err == nil {
		addSkill(filepath.Join(dir, "SKILL.md"), filepath.Base(dir))
	}
	if entries, err := os.ReadDir(filepath.Join(dir, "skills")); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillPath := filepath.Join(dir, "skills", entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillPath); err == nil {
				addSkill(skillPath, entry.Name())
			}
		}
	}
	return out
}

func parseSkillFrontmatter(content string) (name, description string) {
	if !strings.HasPrefix(content, "---") {
		return "", ""
	}
	rest := content[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", ""
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "name:")), `"'`)
		} else if strings.HasPrefix(line, "description:") {
			description = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "description:")), `"'`)
		}
	}
	return name, description
}

func extractMCPServersFromConfigFile(raw []byte) ([]string, []json.RawMessage) {
	var top map[string]json.RawMessage
	if json.Unmarshal(raw, &top) != nil {
		return nil, nil
	}
	if sub, ok := top["mcpServers"]; ok {
		return extractMCPServers(sub)
	}
	return extractMCPServers(raw)
}

func extractMCPServers(raw json.RawMessage) ([]string, []json.RawMessage) {
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) == nil {
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		specs := make([]json.RawMessage, 0, len(keys))
		for _, k := range keys {
			var specMap map[string]any
			if json.Unmarshal(obj[k], &specMap) == nil && specMap != nil {
				if _, hasID := specMap["id"]; !hasID {
					specMap["id"] = k
				}
				if b, err := json.Marshal(specMap); err == nil {
					specs = append(specs, b)
				}
			}
		}
		return keys, specs
	}
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) == nil {
		var names []string
		var specs []json.RawMessage
		for _, item := range arr {
			if s := decodeJSONString(item); s != "" {
				names = append(names, s)
				continue
			}
			var specMap map[string]any
			if json.Unmarshal(item, &specMap) == nil && specMap != nil {
				if name, _ := specMap["id"].(string); name != "" {
					names = append(names, name)
				} else if name, _ := specMap["name"].(string); name != "" {
					names = append(names, name)
				}
				specs = append(specs, item)
			}
		}
		return names, specs
	}
	return nil, nil
}

func extractStringListOrKeys(raw json.RawMessage) []string {
	var s string
	if json.Unmarshal(raw, &s) == nil && strings.TrimSpace(s) != "" {
		return []string{strings.TrimSpace(s)}
	}
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) == nil {
		var out []string
		for _, item := range arr {
			if v := decodeJSONString(item); v != "" {
				out = append(out, v)
				continue
			}
			var obj map[string]json.RawMessage
			if json.Unmarshal(item, &obj) == nil {
				if name := decodeJSONString(obj["name"]); name != "" {
					out = append(out, name)
				}
			}
		}
		return out
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) == nil {
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	}
	return nil
}

func decodeJSONString(raw json.RawMessage) string {
	var s string
	if len(raw) == 0 || json.Unmarshal(raw, &s) != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

func jsonTruthyNonEmpty(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed != "" && trimmed != "null" && trimmed != "false" && trimmed != `""` && trimmed != "[]" && trimmed != "{}"
}

func looksLikeClientPath(p string) bool {
	lower := strings.ToLower(strings.TrimSpace(p))
	return lower == "client" ||
		strings.HasSuffix(lower, "/client") ||
		strings.HasSuffix(lower, "/client.js") ||
		strings.HasSuffix(lower, "/client.mjs") ||
		lower == "client.js" ||
		lower == "./client" ||
		lower == "./client.js"
}

func mergeUniqueStrings(base, extra []string) []string {
	seen := make(map[string]bool, len(base)+len(extra))
	var out []string
	for _, s := range base {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, s := range extra {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
