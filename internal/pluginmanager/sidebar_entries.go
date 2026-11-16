package pluginmanager

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	maxSidebarPackageJSONBytes = 256 << 10
	maxSidebarClientJSBytes    = 512 << 10
	defaultSidebarPluginOrder  = 100
)

// SidebarPluginCandidate supplies optional Host inventory/bundle metadata so
// pure-Go discovery can enrich localized titles, honor disable state, and cover
// remote Hosts without executing browser JavaScript.
type SidebarPluginCandidate struct {
	Package          string
	Title            json.RawMessage
	Enabled          bool
	ExplicitDisabled bool
	Official         bool
	Removable        bool
	Capability       CapabilityKind
}

// SidebarPluginEntry represents one native left-sidebar launcher button backed
// by an on-demand A2 compatibility view (?desktopPlugin=<ID>).
type SidebarPluginEntry struct {
	ID    string
	Label string
	Order int
}

var excludedSidebarPackages = map[string]bool{
	"whaleisle-desktop-plugin-views":              true,
	"dsh-task-control":                            true,
	"dsh-platform-session":                        true,
	"dsh-usage-panel":                             true,
	"@xmanrui/dsh-im":                             true,
	"dsh-im":                                      true,
	"xmanrui-dsh-im":                              true,
	"dsh-acp-agents":                              true,
	"dsh-acp":                                     true,
	"dshbot":                                      true,
	"dsh-bot":                                     true,
	"dsh-whale":                                   true,
	"dsh-remote":                                  true,
	"dsh-project":                                 true,
	"mygo-proactive":                              true,
	"session-search":                              true,
	"@deepseek-ai/dsh-client-ui-settings-market":  true,
	"@deepseek-ai/libreoffice-kit":                true,
	"sherpa-onnx-node":                            true,
}

var (
	sidebarSlotSnippetPattern = regexp.MustCompile(`sidebar\.(?:panellist|nav\.tab)[^;{}\n]{0,120}\{[^}]{0,600}\}`)
	sidebarLabelDoublePattern = regexp.MustCompile(`label\s*:\s*"([^"\\]{1,80})"`)
	sidebarLabelSinglePattern = regexp.MustCompile(`label\s*:\s*'([^'\\]{1,80})'`)
	sidebarOrderPattern       = regexp.MustCompile(`order\s*:\s*(\d{1,5})`)
)

// DiscoverSidebarPlugins returns left-sidebar custom plugin launchers sorted by
// (Order, Label, ID). It reads only package.json and static client hint text
// under <profileDir>/node_modules without launching a browser or executing JS.
func DiscoverSidebarPlugins(profileDir, locale string, candidates []SidebarPluginCandidate) []SidebarPluginEntry {
	candMap := make(map[string]SidebarPluginCandidate, len(candidates))
	candOrder := make([]string, 0, len(candidates))
	for _, c := range candidates {
		pkg := strings.TrimSpace(c.Package)
		if pkg == "" {
			continue
		}
		if prev, exists := candMap[pkg]; exists {
			prev.Enabled = prev.Enabled || c.Enabled
			prev.ExplicitDisabled = prev.ExplicitDisabled || c.ExplicitDisabled
			prev.Official = prev.Official || c.Official
			prev.Removable = prev.Removable || c.Removable
			if len(prev.Title) == 0 && len(c.Title) > 0 {
				prev.Title = c.Title
			}
			if prev.Capability == "" {
				prev.Capability = c.Capability
			}
			candMap[pkg] = prev
			continue
		}
		c.Package = pkg
		candMap[pkg] = c
		candOrder = append(candOrder, pkg)
	}

	discovered := make(map[string]SidebarPluginEntry)
	if strings.TrimSpace(profileDir) != "" {
		scanProfileNodeModules(profileDir, locale, candMap, discovered)
	}

	for _, pkg := range candOrder {
		cand := candMap[pkg]
		if _, seen := discovered[pkg]; seen {
			continue
		}
		if cand.ExplicitDisabled || !cand.Enabled {
			continue
		}
		if isExcludedSidebarPackage(pkg, cand, true) {
			continue
		}
		if cand.Official && !cand.Removable {
			continue
		}
		if cand.Capability != CapabilityWebUIOnly && cand.Capability != CapabilityHybrid {
			continue
		}
		label := localizedMetaTitle(cand.Title, locale)
		if label == "" {
			label = shortPackageLabel(pkg)
		}
		discovered[pkg] = SidebarPluginEntry{
			ID:    pkg,
			Label: label,
			Order: defaultSidebarPluginOrder,
		}
	}

	if len(discovered) == 0 {
		return nil
	}
	out := make([]SidebarPluginEntry, 0, len(discovered))
	for _, entry := range discovered {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		li, lj := strings.ToLower(out[i].Label), strings.ToLower(out[j].Label)
		if li != lj {
			return li < lj
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func scanProfileNodeModules(profileDir, locale string, candMap map[string]SidebarPluginCandidate, out map[string]SidebarPluginEntry) {
	nodeModules := filepath.Join(profileDir, "node_modules")
	entries, err := os.ReadDir(nodeModules)
	if err != nil {
		return
	}
	for _, ent := range entries {
		name := ent.Name()
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(nodeModules, name)
		if strings.HasPrefix(name, "@") {
			info, statErr := os.Stat(full)
			if statErr != nil || !info.IsDir() {
				continue
			}
			scoped, readErr := os.ReadDir(full)
			if readErr != nil {
				continue
			}
			for _, sub := range scoped {
				subName := sub.Name()
				if subName == "" || strings.HasPrefix(subName, ".") {
					continue
				}
				inspectSidebarPackageDir(filepath.Join(full, subName), name+"/"+subName, locale, candMap, out)
			}
			continue
		}
		inspectSidebarPackageDir(full, name, locale, candMap, out)
	}
}

func inspectSidebarPackageDir(pkgDir, dirPkg, locale string, candMap map[string]SidebarPluginCandidate, out map[string]SidebarPluginEntry) {
	info, err := os.Stat(pkgDir)
	if err != nil || !info.IsDir() {
		return
	}
	raw, err := readBoundedFile(filepath.Join(pkgDir, "package.json"), maxSidebarPackageJSONBytes)
	if err != nil {
		return
	}
	var manifest map[string]json.RawMessage
	if json.Unmarshal(raw, &manifest) != nil {
		return
	}
	if !jsonTruthyNonEmpty(manifest["dsh"]) {
		return
	}
	if !ClassifyPackageManifestJSON(raw, false, false).HasWebClient {
		return
	}

	pkg := strings.TrimSpace(decodeJSONString(manifest["name"]))
	if pkg == "" {
		pkg = dirPkg
	}
	if pkg == "" {
		return
	}

	cand, hasCand := lookupSidebarCandidate(candMap, pkg, dirPkg)
	if hasCand && cand.ExplicitDisabled {
		return
	}
	if isExcludedSidebarPackage(pkg, cand, hasCand) || (dirPkg != pkg && isExcludedSidebarPackage(dirPkg, cand, hasCand)) {
		return
	}

	clientLabel, clientOrder := readStaticSidebarClientHints(pkgDir, manifest)
	label := clientLabel
	if label == "" && hasCand {
		label = localizedMetaTitle(cand.Title, locale)
	}
	if label == "" {
		label = manifestTitle(manifest, locale)
	}
	if label == "" {
		label = shortPackageLabel(pkg)
	}
	order := defaultSidebarPluginOrder
	if clientOrder > 0 {
		order = clientOrder
	}
	out[pkg] = SidebarPluginEntry{
		ID:    pkg,
		Label: label,
		Order: order,
	}
}

func lookupSidebarCandidate(candMap map[string]SidebarPluginCandidate, pkg, dirPkg string) (SidebarPluginCandidate, bool) {
	if c, ok := candMap[pkg]; ok {
		return c, true
	}
	if dirPkg != "" && dirPkg != pkg {
		if c, ok := candMap[dirPkg]; ok {
			return c, true
		}
	}
	return SidebarPluginCandidate{}, false
}

func isExcludedSidebarPackage(pkg string, cand SidebarPluginCandidate, hasCand bool) bool {
	lower := strings.ToLower(strings.TrimSpace(pkg))
	if lower == "" {
		return true
	}
	if excludedSidebarPackages[lower] {
		return true
	}
	if hasCand && cand.Official && !cand.Removable {
		return true
	}
	if strings.HasPrefix(lower, "@deepseek-ai/") {
		return !hasCand || !cand.Removable || cand.Official
	}
	return false
}

func readBoundedFile(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, limit))
}

func readStaticSidebarClientHints(pkgDir string, manifest map[string]json.RawMessage) (string, int) {
	for _, rel := range clientEntryCandidates(manifest) {
		rel = strings.TrimSpace(strings.TrimPrefix(filepath.ToSlash(rel), "./"))
		if rel == "" {
			continue
		}
		abs := filepath.Join(pkgDir, filepath.FromSlash(rel))
		if !isPathWithinDir(pkgDir, abs) {
			continue
		}
		body, err := readBoundedFile(abs, maxSidebarClientJSBytes)
		if err != nil {
			continue
		}
		if label, order := scanStaticSidebarSlotText(string(body)); label != "" || order > 0 {
			return label, order
		}
	}
	return "", 0
}

func clientEntryCandidates(manifest map[string]json.RawMessage) []string {
	var paths []string
	if exportsRaw := manifest["exports"]; len(exportsRaw) > 0 {
		var exportsObj map[string]json.RawMessage
		if json.Unmarshal(exportsRaw, &exportsObj) == nil {
			if clientExport, ok := exportsObj["./client"]; ok {
				paths = append(paths, extractExportTarget(clientExport)...)
			}
		}
	}
	if dshRaw := manifest["dsh"]; len(dshRaw) > 0 {
		var dshObj map[string]json.RawMessage
		if json.Unmarshal(dshRaw, &dshObj) == nil {
			if clientRaw := dshObj["client"]; len(clientRaw) > 0 {
				var s string
				if json.Unmarshal(clientRaw, &s) == nil && strings.TrimSpace(s) != "" {
					paths = append(paths, s)
				} else {
					var obj map[string]json.RawMessage
					if json.Unmarshal(clientRaw, &obj) == nil {
						for _, k := range []string{"entry", "main", "module"} {
							if v := decodeJSONString(obj[k]); v != "" {
								paths = append(paths, v)
							}
						}
					}
				}
			}
		}
	}
	return paths
}

func extractExportTarget(raw json.RawMessage) []string {
	if s := decodeJSONString(raw); s != "" {
		return []string{s}
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) == nil {
		var out []string
		for _, k := range []string{"browser", "import", "default", "require"} {
			if v := decodeJSONString(obj[k]); v != "" {
				out = append(out, v)
			}
		}
		return out
	}
	return nil
}

func isPathWithinDir(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func scanStaticSidebarSlotText(src string) (string, int) {
	match := sidebarSlotSnippetPattern.FindString(src)
	if match == "" {
		return "", 0
	}
	var label string
	if m := sidebarLabelDoublePattern.FindStringSubmatch(match); len(m) == 2 {
		label = strings.TrimSpace(m[1])
	} else if m := sidebarLabelSinglePattern.FindStringSubmatch(match); len(m) == 2 {
		label = strings.TrimSpace(m[1])
	}
	order := 0
	if m := sidebarOrderPattern.FindStringSubmatch(match); len(m) == 2 {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
			order = n
		}
	}
	return label, order
}

func manifestTitle(manifest map[string]json.RawMessage, locale string) string {
	if dshRaw := manifest["dsh"]; len(dshRaw) > 0 {
		var dshObj map[string]json.RawMessage
		if json.Unmarshal(dshRaw, &dshObj) == nil {
			if t := localizedMetaTitle(dshObj["title"], locale); t != "" {
				return t
			}
		}
	}
	if t := decodeJSONString(manifest["displayName"]); t != "" {
		return t
	}
	return ""
}

func localizedMetaTitle(raw json.RawMessage, locale string) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	if s := decodeJSONString(raw); s != "" {
		return s
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil || len(m) == 0 {
		return ""
	}
	pref := strings.ToLower(strings.TrimSpace(locale))
	var keys []string
	if strings.HasPrefix(pref, "zh") {
		keys = []string{"zh-CN", "zh", "zh-Hans", "default", "en", "en-US"}
	} else if pref != "" {
		keys = []string{pref, "en", "en-US", "default", "zh-CN", "zh"}
	} else {
		keys = []string{"default", "zh-CN", "zh", "en", "en-US"}
	}
	for _, k := range keys {
		if v := decodeJSONString(m[k]); v != "" {
			return v
		}
	}
	allKeys := make([]string, 0, len(m))
	for k := range m {
		allKeys = append(allKeys, k)
	}
	sort.Strings(allKeys)
	for _, k := range allKeys {
		if v := decodeJSONString(m[k]); v != "" {
			return v
		}
	}
	return ""
}

func shortPackageLabel(pkg string) string {
	if idx := strings.LastIndex(pkg, "/"); idx >= 0 && idx+1 < len(pkg) {
		return pkg[idx+1:]
	}
	return pkg
}
