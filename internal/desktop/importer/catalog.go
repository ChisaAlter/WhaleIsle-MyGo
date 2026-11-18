package importer

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"whaleisle/internal/store/storelocale"
)

// Catalog is the Electron scanImport-shaped inventory for the launcher import tab.
type Catalog struct {
	SourceHome      string
	DestHome        string
	DestEmpty       bool
	SourceHasData   bool
	HasAttachments  bool
	SkillRoots      []string
	ExtraSkillRoots []string
	Sessions        []CatalogItem
	Skills          []CatalogItem
	Plugins         []CatalogItem
	MCP             []CatalogItem
	Settings        []CatalogItem
	Presets         []CatalogItem
}

// CatalogItem is one selectable row (sessions use Rel as ID; skills use prefix:name).
type CatalogItem struct {
	ID       string
	Label    string
	Group    string // sessions: cwd; skills: source prefix
	Rel      string
	Abs      string
	Spec     string // plugins: original dependency spec, never a local copy source
	Conflict bool
	Blocker  Blocker
	Selected bool // UI default: true when importable
}

// Blocker says why a row cannot be imported. Supported is the zero value; every
// other value is a category the launcher turns into a sentence in the reader's
// language, so a row never carries the scanner's own words to the screen.
type Blocker int

const (
	Supported Blocker = iota
	// BlockedUnreadable: the session directory could not be read at all.
	BlockedUnreadable
	// BlockedMixedEncoding: the same session holds plaintext and compressed logs.
	BlockedMixedEncoding
	// BlockedLegacyDatabase: a SQLite-era session directory this build cannot read.
	BlockedLegacyDatabase
	// BlockedRejected: the item's own scanner refused it.
	BlockedRejected
)

// Importable reports whether the row can be copied.
func (b Blocker) Importable() bool { return b == Supported }

// String names the blocker for the log and for an error a caller wraps with
// errcopy. The launcher translates the category instead of using this.
func (b Blocker) String() string {
	switch b {
	case BlockedUnreadable:
		return "the session directory could not be read"
	case BlockedMixedEncoding:
		return "the session holds both plaintext and compressed logs"
	case BlockedLegacyDatabase:
		return "the session is a legacy SQLite database"
	case BlockedRejected:
		return "the item's scanner rejected it"
	default:
		return "importable"
	}
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
	return BuildCatalogContext(context.Background(), sourceHome, destHome)
}

// BuildCatalogContext is the cancellable inventory used by interactive scans.
// Existing non-interactive callers retain BuildCatalog's original contract.
func BuildCatalogContext(ctx context.Context, sourceHome, destHome string) (Catalog, error) {
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
	info, lerr := os.Lstat(src)
	if lerr != nil {
		if !os.IsNotExist(lerr) {
			return cat, lerr
		}
		return cat, nil
	}
	if info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
		return cat, fmt.Errorf("unsafe import source directory: %s", src)
	}
	if !info.IsDir() {
		return cat, nil
	}

	var err error
	cat.Sessions, err = scanSessionsContext(ctx, src, dst)
	if err != nil {
		return cat, err
	}
	cat.Skills, cat.SkillRoots = scanSkills(src, dst)
	cat.Plugins = scanPlugins(src, dst)
	if err := ctx.Err(); err != nil {
		return cat, err
	}
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
	destSessions, err := walkSessionRelsContext(ctx, filepath.Join(dst, "sessions"))
	if err != nil {
		return cat, err
	}
	cat.DestEmpty = len(destSessions) == 0
	cat.SourceHasData = len(cat.Sessions)+len(cat.Skills)+len(cat.Plugins)+len(cat.MCP)+len(cat.Settings)+len(cat.Presets) > 0 || cat.HasAttachments

	// Default-select importable items (Electron: all choosable rows checked).
	for i := range cat.Sessions {
		cat.Sessions[i].Selected = cat.Sessions[i].Blocker.Importable()
	}
	for i := range cat.Skills {
		cat.Skills[i].Selected = true
	}
	for i := range cat.Plugins {
		cat.Plugins[i].Selected = cat.Plugins[i].Blocker.Importable()
	}
	for i := range cat.MCP {
		cat.MCP[i].Selected = true
	}
	for i := range cat.Settings {
		cat.Settings[i].Selected = true
	}
	for i := range cat.Presets {
		cat.Presets[i].Selected = cat.Presets[i].Blocker.Importable()
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
	knownExtra := false
	for _, root := range cat.ExtraSkillRoots {
		if filepath.Clean(root) == dir {
			knownExtra = true
			break
		}
	}
	if !knownExtra {
		cat.ExtraSkillRoots = append(cat.ExtraSkillRoots, dir)
	}
	prefix := "extra:" + filepath.ToSlash(dir)
	added := 0
	if entryExists(filepath.Join(dir, "SKILL.md"), false) {
		name := filepath.Base(dir)
		id := prefix + ":" + name
		if !skillIDExists(cat.Skills, id) {
			cat.Skills = append(cat.Skills, CatalogItem{
				ID: id, Label: name, Group: dir, Abs: dir,
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
		if !isSafeDirEntry(e) {
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
			ID: id, Label: name, Group: dir, Abs: abs,
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
			if it.Selected && it.Blocker.Importable() {
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
	rows, _ := scanSessionsContext(context.Background(), src, dst)
	return rows
}

func scanSessionsContext(ctx context.Context, src, dst string) ([]CatalogItem, error) {
	srcRoot := filepath.Join(src, "sessions")
	dstRoot := filepath.Join(dst, "sessions")
	rels, err := walkSessionRelsContext(ctx, srcRoot)
	if err != nil {
		return nil, err
	}
	out := make([]CatalogItem, 0, len(rels))
	type logSource struct {
		item              int
		plain, compressed string
	}
	sources := make([]logSource, 0, len(rels))
	var compressed []string
	for _, rel := range rels {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if strings.HasPrefix(rel, "_no-cwd/preset-") || strings.HasPrefix(rel, `_no-cwd\preset-`) {
			continue
		}
		abs := filepath.Join(srcRoot, filepath.FromSlash(rel))
		plain, packed := sessionLogFiles(abs)
		if packed != "" {
			compressed = append(compressed, packed)
		}
		sources = append(sources, logSource{item: len(out), plain: plain, compressed: packed})
		blocker, readErr := sessionEncoding(abs)
		if readErr != nil {
			// The row carries a category and the launcher writes the sentence;
			// the scanner's own words are worth the log and nothing else.
			log.Printf("import catalog: session %s: %v", abs, readErr)
		}
		conflict := entryExists(filepath.Join(dstRoot, filepath.FromSlash(rel)), true)
		out = append(out, CatalogItem{
			ID: filepath.Base(abs), Rel: rel, Abs: abs,
			Conflict: conflict, Blocker: blocker,
		})
	}
	// The directories are known before any log is opened, so one process
	// decodes every compressed log instead of one process per session.
	packed := readCompressedSessionMeta(ctx, compressed)
	if len(compressed) > 0 && len(packed) == 0 {
		log.Printf("import catalog: %d compressed session logs had no decoder; listing them by id", len(compressed))
	}
	for _, source := range sources {
		var meta sessionMeta
		switch {
		case source.plain != "":
			meta = readPlainSessionLog(ctx, source.plain)
		case source.compressed != "":
			meta = packed[source.compressed]
		}
		item := &out[source.item]
		item.Label = meta.Title
		if item.Label == "" {
			item.Label = item.ID
		}
		item.Group = meta.Cwd
		if item.Group == "" {
			item.Group = storelocale.T(storelocale.KeyStoreImporterNoWorkspace)
		}
	}
	return out, ctx.Err()
}

// sessionLogFiles picks the log a session row shows: the first plain session
// log, else the last compressed one.
func sessionLogFiles(dir string) (plain, compressed string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", ""
	}
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		if strings.HasPrefix(name, "session") && strings.HasSuffix(name, ".jsonl") && !strings.Contains(name, ".zstd") {
			return filepath.Join(dir, e.Name()), ""
		}
		if strings.HasPrefix(name, "session") && strings.HasSuffix(name, ".jsonl.zstd") {
			compressed = filepath.Join(dir, e.Name())
		}
	}
	return "", compressed
}

// Display metadata follows the log in order: the last session/title wins.
func readPlainSessionLog(ctx context.Context, path string) sessionMeta {
	var meta sessionMeta
	f, err := os.Open(path)
	if err != nil {
		return meta
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		if ctx.Err() != nil {
			return meta
		}
		line, err := r.ReadBytes('\n')
		var row struct {
			Type, Title, Cwd string
			Data             struct{ Title string }
		}
		if json.Unmarshal(line, &row) == nil {
			if row.Type == "session" {
				if row.Cwd != "" {
					meta.Cwd = row.Cwd
				}
				if meta.Title == "" && strings.TrimSpace(row.Title) != "" {
					meta.Title = strings.TrimSpace(row.Title)
				}
			}
			if row.Type == "session/title" {
				v := row.Data.Title
				if v == "" {
					v = row.Title
				}
				if strings.TrimSpace(v) != "" {
					meta.Title = strings.TrimSpace(v)
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}
	return meta
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
	rows, _ := walkSessionRelsContext(context.Background(), sessionsRoot)
	return rows
}

func isSafeDirEntry(d os.DirEntry) bool {
	return d != nil && d.Type()&(os.ModeSymlink|os.ModeIrregular) == 0 && d.IsDir()
}

func walkSessionRelsContext(ctx context.Context, sessionsRoot string) ([]string, error) {
	var found []string
	if !entryExists(sessionsRoot, true) {
		return found, ctx.Err()
	}
	err := filepath.WalkDir(sessionsRoot, func(path string, d os.DirEntry, err error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err != nil || d == nil {
			return nil
		}
		if d.Type()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path == sessionsRoot {
			return nil
		}
		if hasSessionLog(path) || sessionUnsupported(path) {
			rel, err := filepath.Rel(sessionsRoot, path)
			if err != nil {
				return nil
			}
			found = append(found, filepath.ToSlash(rel))
			return filepath.SkipDir
		}
		return nil
	})
	return found, err
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
	blocker, _ := sessionEncoding(dir)
	return !blocker.Importable()
}

// sessionEncoding classifies a session directory without deciding what to say
// about it. The launcher owns the sentence; a directory that could not be read
// also returns the OS error so the caller can put it in the log.
func sessionEncoding(dir string) (Blocker, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return BlockedUnreadable, err
	}
	hasPlain, hasCompressed, hasDB := false, false, false
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		if strings.HasSuffix(name, ".db") {
			hasDB = true
		}
		if strings.HasPrefix(name, "session") {
			hasPlain = hasPlain || strings.HasSuffix(name, ".jsonl")
			hasCompressed = hasCompressed || strings.HasSuffix(name, ".jsonl.zstd")
		}
	}
	if hasPlain && hasCompressed {
		return BlockedMixedEncoding, nil
	}
	if hasDB && !hasPlain && !hasCompressed {
		return BlockedLegacyDatabase, nil
	}
	return Supported, nil
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
			if !isSafeDirEntry(e) {
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
		if !isSafeDirEntry(e) || !presetIDRe.MatchString(e.Name()) {
			continue
		}
		abs := filepath.Join(dir, e.Name())
		broken := Supported
		if !entryExists(filepath.Join(abs, "agent.cordis.yml"), false) {
			broken = BlockedRejected
		}
		out = append(out, CatalogItem{
			ID: e.Name(), Label: e.Name(), Abs: abs,
			Blocker:  broken,
			Conflict: entryExists(filepath.Join(dst, ".agent-presets", e.Name()), true),
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
