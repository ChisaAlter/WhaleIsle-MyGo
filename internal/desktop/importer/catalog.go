package importer

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Catalog is the Electron scanImport-shaped inventory for the launcher import tab.
type Catalog struct {
	SourceHome     string
	DestHome       string
	DestEmpty      bool
	SourceHasData  bool
	HasAttachments bool
	SkillRoots     []string
	Sessions       []CatalogItem
	Skills         []CatalogItem
	Plugins        []CatalogItem
	MCP            []CatalogItem
	Settings       []CatalogItem
	Presets        []CatalogItem
}

// CatalogItem is one selectable row (sessions use Rel as ID; skills use prefix:name).
type CatalogItem struct {
	ID          string
	Label       string
	Group       string // sessions: cwd; skills: source prefix
	Rel         string
	Abs         string
	Conflict    bool
	Unsupported bool
	Selected    bool // UI default: true when importable
}

var presetIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// OfficialDshHome is ~/.dsh (Electron officialDshHome).
func OfficialDshHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".dsh")
}

// AgentsSkillsRoot is ~/.agents/skills.
func AgentsSkillsRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".agents", "skills")
}

// BuildCatalog scans sourceHome (default ~/.dsh) against destHome (default paths.DshHome via dest).
func BuildCatalog(sourceHome, destHome string) (Catalog, error) {
	src := strings.TrimSpace(sourceHome)
	if src == "" {
		src = OfficialDshHome()
	}
	src = filepath.Clean(src)
	dst := strings.TrimSpace(destHome)
	if dst == "" {
		// Prefer Electron desktop dsh-home under default userData when MyGo override is unset.
		dst = filepath.Join(ElectronUserDataDir(), "dsh-home")
	}
	dst = filepath.Clean(dst)

	cat := Catalog{
		SourceHome: src,
		DestHome:   dst,
	}
	if src == "" || src == "." {
		return cat, ErrNotDirectory
	}
	if info, err := os.Stat(src); err != nil || !info.IsDir() {
		if err != nil && !os.IsNotExist(err) {
			return cat, err
		}
		return cat, nil
	}

	cat.Sessions = scanSessions(src, dst)
	cat.Skills, cat.SkillRoots = scanSkills(src, dst)
	cat.Plugins = scanPlugins(src, dst)
	var err error
	cat.MCP, err = scanMCP(src, dst)
	if err != nil {
		return cat, err
	}
	cat.Settings, err = scanSettings(src, dst)
	if err != nil {
		return cat, err
	}
	cat.Presets = scanPresets(src, dst)
	cat.HasAttachments = entryExists(filepath.Join(src, "attachments"), true)
	cat.DestEmpty = len(walkSessionRels(filepath.Join(dst, "sessions"))) == 0
	cat.SourceHasData = len(cat.Sessions)+len(cat.Skills)+len(cat.Plugins)+len(cat.MCP)+len(cat.Settings)+len(cat.Presets) > 0 || cat.HasAttachments

	// Default-select importable items (Electron: all choosable rows checked).
	for i := range cat.Sessions {
		cat.Sessions[i].Selected = !cat.Sessions[i].Unsupported
	}
	for i := range cat.Skills {
		cat.Skills[i].Selected = true
	}
	for i := range cat.Plugins {
		cat.Plugins[i].Selected = !cat.Plugins[i].Unsupported
	}
	for i := range cat.MCP {
		cat.MCP[i].Selected = true
	}
	for i := range cat.Settings {
		cat.Settings[i].Selected = true
	}
	for i := range cat.Presets {
		cat.Presets[i].Selected = !cat.Presets[i].Unsupported
	}
	return cat, nil
}

// AppendSkillDirectory adds skills from an extra directory (Electron「添加技能目录」).
func AppendSkillDirectory(cat *Catalog, dir string) int {
	if cat == nil {
		return 0
	}
	dir = filepath.Clean(strings.TrimSpace(dir))
	if dir == "" || dir == "." || !entryExists(dir, true) {
		return 0
	}
	for _, r := range cat.SkillRoots {
		if filepath.Clean(r) == dir {
			return 0
		}
	}
	cat.SkillRoots = append(cat.SkillRoots, dir)
	destSkills := filepath.Join(cat.DestHome, "skills")
	prefix := "extra"
	added := 0
	if entryExists(filepath.Join(dir, "SKILL.md"), false) {
		name := filepath.Base(dir)
		id := prefix + ":" + name
		if !skillIDExists(cat.Skills, id) {
			cat.Skills = append(cat.Skills, CatalogItem{
				ID: id, Label: name, Group: prefix, Abs: dir,
				Conflict: entryExists(filepath.Join(destSkills, name), true),
				Selected: true,
			})
			added++
		}
		return added
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		abs := filepath.Join(dir, e.Name())
		if !entryExists(filepath.Join(abs, "SKILL.md"), false) {
			continue
		}
		name := e.Name()
		id := prefix + ":" + name
		if skillIDExists(cat.Skills, id) {
			continue
		}
		cat.Skills = append(cat.Skills, CatalogItem{
			ID: id, Label: name, Group: prefix, Abs: abs,
			Conflict: entryExists(filepath.Join(destSkills, name), true),
			Selected: true,
		})
		added++
	}
	return added
}

func skillIDExists(items []CatalogItem, id string) bool {
	for _, it := range items {
		if it.ID == id {
			return true
		}
	}
	return false
}

// CategoryCounts returns chip n/d for each launcher category id.
func (c Catalog) CategoryCounts() map[string][2]int {
	count := func(items []CatalogItem) [2]int {
		sel := 0
		for _, it := range items {
			if it.Selected && !it.Unsupported {
				sel++
			}
		}
		return [2]int{sel, len(items)}
	}
	return map[string][2]int{
		"sessions": count(c.Sessions),
		"skills":   count(c.Skills),
		"plugins":  count(c.Plugins),
		"mcp":      count(c.MCP),
		"settings": count(c.Settings),
		"presets":  count(c.Presets),
	}
}

// ItemsFor returns the slice for a category chip id.
func (c *Catalog) ItemsFor(category string) *[]CatalogItem {
	switch category {
	case "sessions":
		return &c.Sessions
	case "skills":
		return &c.Skills
	case "plugins":
		return &c.Plugins
	case "mcp":
		return &c.MCP
	case "settings":
		return &c.Settings
	case "presets":
		return &c.Presets
	default:
		return nil
	}
}

func scanSessions(src, dst string) []CatalogItem {
	srcRoot := filepath.Join(src, "sessions")
	dstRoot := filepath.Join(dst, "sessions")
	rels := walkSessionRels(srcRoot)
	out := make([]CatalogItem, 0, len(rels))
	for _, rel := range rels {
		if strings.HasPrefix(rel, "_no-cwd/preset-") || strings.HasPrefix(rel, `_no-cwd\preset-`) {
			continue
		}
		abs := filepath.Join(srcRoot, filepath.FromSlash(rel))
		id := filepath.Base(abs)
		label, group := readPlainSessionMeta(abs)
		if label == "" {
			label = id
		}
		if group == "" {
			group = "无工作区"
		}
		unsupported := sessionUnsupported(abs)
		conflict := entryExists(filepath.Join(dstRoot, filepath.FromSlash(rel)), true)
		out = append(out, CatalogItem{
			ID: id, Label: label, Group: group, Rel: rel, Abs: abs,
			Conflict: conflict, Unsupported: unsupported,
		})
	}
	return out
}

// readPlainSessionMeta peeks plain session*.jsonl for title/cwd (skip zstd).
func readPlainSessionMeta(dir string) (title, cwd string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", ""
	}
	var logPath string
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		if strings.HasPrefix(name, "session") && strings.HasSuffix(name, ".jsonl") && !strings.Contains(name, ".zstd") {
			logPath = filepath.Join(dir, e.Name())
			break
		}
	}
	if logPath == "" {
		return "", ""
	}
	raw, err := os.ReadFile(logPath)
	if err != nil {
		return "", ""
	}
	// Cap peek to first 64KiB.
	if len(raw) > 64<<10 {
		raw = raw[:64<<10]
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] != '{' {
			continue
		}
		if title == "" {
			if v := jsonStringField(line, "title"); v != "" {
				title = v
			}
		}
		if cwd == "" {
			if v := jsonStringField(line, "cwd"); v != "" {
				cwd = v
			}
		}
		if title != "" && cwd != "" {
			break
		}
	}
	return title, cwd
}

func jsonStringField(line, key string) string {
	needle := `"` + key + `"`
	idx := strings.Index(line, needle)
	if idx < 0 {
		return ""
	}
	rest := line[idx+len(needle):]
	colon := strings.IndexByte(rest, ':')
	if colon < 0 {
		return ""
	}
	rest = strings.TrimSpace(rest[colon+1:])
	if !strings.HasPrefix(rest, `"`) {
		return ""
	}
	rest = rest[1:]
	var b strings.Builder
	for i := 0; i < len(rest); i++ {
		ch := rest[i]
		if ch == '\\' && i+1 < len(rest) {
			b.WriteByte(rest[i+1])
			i++
			continue
		}
		if ch == '"' {
			break
		}
		b.WriteByte(ch)
	}
	return b.String()
}

func walkSessionRels(sessionsRoot string) []string {
	var found []string
	if !entryExists(sessionsRoot, true) {
		return found
	}
	_ = filepath.WalkDir(sessionsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || !d.IsDir() {
			return nil
		}
		if path == sessionsRoot {
			return nil
		}
		if hasSessionLog(path) {
			rel, err := filepath.Rel(sessionsRoot, path)
			if err != nil {
				return nil
			}
			found = append(found, filepath.ToSlash(rel))
			return filepath.SkipDir
		}
		return nil
	})
	return found
}

func hasSessionLog(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		if strings.HasPrefix(name, "session") && (strings.HasSuffix(name, ".jsonl") || strings.Contains(name, ".jsonl.")) {
			return true
		}
	}
	return false
}

func sessionUnsupported(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true
	}
	hasLog, hasDB := false, false
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		if strings.HasSuffix(name, ".db") {
			hasDB = true
		}
		if strings.HasPrefix(name, "session") && strings.Contains(name, ".jsonl") {
			hasLog = true
		}
	}
	return hasDB && !hasLog
}

func scanSkills(src, dst string) ([]CatalogItem, []string) {
	roots := []struct{ prefix, dir string }{
		{"home", filepath.Join(src, "skills")},
		{"agents", AgentsSkillsRoot()},
	}
	var skillRoots []string
	var out []CatalogItem
	destSkills := filepath.Join(dst, "skills")
	for _, root := range roots {
		if root.dir == "" || !entryExists(root.dir, true) {
			continue
		}
		skillRoots = append(skillRoots, root.dir)
		entries, err := os.ReadDir(root.dir)
		if err != nil {
			continue
		}
		// Root itself may be a skill package.
		if entryExists(filepath.Join(root.dir, "SKILL.md"), false) {
			name := filepath.Base(root.dir)
			id := root.prefix + ":" + name
			out = append(out, CatalogItem{
				ID: id, Label: name, Group: root.prefix, Abs: root.dir,
				Conflict: entryExists(filepath.Join(destSkills, name), true),
			})
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			abs := filepath.Join(root.dir, e.Name())
			if !entryExists(filepath.Join(abs, "SKILL.md"), false) {
				continue
			}
			name := e.Name()
			id := root.prefix + ":" + name
			out = append(out, CatalogItem{
				ID: id, Label: name, Group: root.prefix, Abs: abs,
				Conflict: entryExists(filepath.Join(destSkills, name), true),
			})
		}
	}
	return out, skillRoots
}

func scanPlugins(src, dst string) []CatalogItem {
	manifest := filepath.Join(src, "profiles", "web", "package.json")
	if !entryExists(manifest, false) {
		// Fallback: list plugin dirs when package.json absent (desktop layout).
		for _, sub := range []string{"plugins", "user-plugins"} {
			dir := filepath.Join(src, sub)
			if !entryExists(dir, true) {
				continue
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			var out []CatalogItem
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				out = append(out, CatalogItem{
					ID: e.Name(), Label: e.Name(), Abs: filepath.Join(dir, e.Name()),
					Conflict:    entryExists(filepath.Join(dst, sub, e.Name()), true),
					Unsupported: true, // apply skips plugins — install via market
					Group:       "需插件市场安装",
				})
			}
			return out
		}
		return nil
	}
	raw, err := os.ReadFile(manifest)
	if err != nil {
		return nil
	}
	// Lightweight dep name extraction without full JSON parse of nested fields.
	deps := extractJSONObjectKeys(string(raw), "dependencies")
	var out []CatalogItem
	for _, name := range deps {
		out = append(out, CatalogItem{
			ID: name, Label: name,
			// Copy-import cannot reinstall npm plugins; surface in UI as unsupported.
			Unsupported: true,
			Group:       "需插件市场安装",
			Conflict:    entryExists(filepath.Join(dst, "profiles", "web"), true),
		})
	}
	return out
}

func scanMCP(src, dst string) ([]CatalogItem, error) {
	source, err := readImportText(filepath.Join(src, "mcp-servers.yaml"))
	if err != nil {
		return nil, err
	}
	target, err := readImportText(filepath.Join(dst, "mcp-servers.yaml"))
	if err != nil {
		return nil, err
	}
	_, from, err := splitMCP(string(source))
	if err != nil {
		return nil, err
	}
	_, to, err := splitMCP(string(target))
	if err != nil {
		return nil, err
	}
	var rows []CatalogItem
	for _, id := range from.order {
		_, conflict := to.blocks[id]
		rows = append(rows, CatalogItem{ID: id, Label: id, Abs: filepath.Join(src, "mcp-servers.yaml"), Conflict: conflict})
	}
	return rows, nil
}

func scanSettings(src, dst string) ([]CatalogItem, error) {
	source, err := readImportText(filepath.Join(src, "settings.yaml"))
	if err != nil {
		return nil, err
	}
	target, err := readImportText(filepath.Join(dst, "settings.yaml"))
	if err != nil {
		return nil, err
	}
	from, err := splitSections(string(source))
	if err != nil {
		return nil, err
	}
	to, err := splitSections(string(target))
	if err != nil {
		return nil, err
	}
	var rows []CatalogItem
	for _, id := range settingSections {
		if _, ok := from.blocks[id]; !ok {
			continue
		}
		_, conflict := to.blocks[id]
		rows = append(rows, CatalogItem{ID: id, Label: id, Abs: filepath.Join(src, "settings.yaml"), Conflict: conflict})
	}
	if entryExists(filepath.Join(src, "AGENTS.md"), false) {
		rows = append(rows, CatalogItem{ID: "agents-md", Label: "AGENTS.md", Abs: filepath.Join(src, "AGENTS.md"), Conflict: entryExists(filepath.Join(dst, "AGENTS.md"), false)})
	}
	return rows, nil
}

func scanPresets(src, dst string) []CatalogItem {
	dir := filepath.Join(src, ".agent-presets")
	if !entryExists(dir, true) {
		alt := filepath.Join(src, "agent-presets")
		if entryExists(alt, true) {
			dir = alt
		} else {
			return nil
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []CatalogItem
	for _, e := range entries {
		if !e.IsDir() || !presetIDRe.MatchString(e.Name()) {
			continue
		}
		abs := filepath.Join(dir, e.Name())
		broken := !entryExists(filepath.Join(abs, "agent.cordis.yml"), false)
		out = append(out, CatalogItem{
			ID: e.Name(), Label: e.Name(), Abs: abs,
			Unsupported: broken,
			Conflict:    entryExists(filepath.Join(dst, ".agent-presets", e.Name()), true),
		})
	}
	return out
}

func yamlTopLevelIDs(raw string) []string {
	var ids []string
	for _, line := range strings.Split(raw, "\n") {
		if len(line) == 0 || line[0] == ' ' || line[0] == '\t' || line[0] == '#' || line[0] == '-' {
			continue
		}
		if i := strings.IndexByte(line, ':'); i > 0 {
			key := strings.TrimSpace(line[:i])
			if key != "" {
				ids = append(ids, key)
			}
		}
	}
	return ids
}

func extractJSONObjectKeys(raw, field string) []string {
	idx := strings.Index(raw, `"`+field+`"`)
	if idx < 0 {
		return nil
	}
	rest := raw[idx:]
	brace := strings.Index(rest, "{")
	if brace < 0 {
		return nil
	}
	depth := 0
	end := -1
	for i := brace; i < len(rest); i++ {
		switch rest[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end = i
				i = len(rest)
			}
		}
	}
	if end < 0 {
		return nil
	}
	body := rest[brace+1 : end]
	var keys []string
	for _, part := range strings.Split(body, ",") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, `"`) {
			continue
		}
		part = part[1:]
		q := strings.IndexByte(part, '"')
		if q <= 0 {
			continue
		}
		keys = append(keys, part[:q])
	}
	return keys
}
