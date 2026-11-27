package plugininventory

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"sync"
)

type DisplayRow struct {
	Row
	Title, Description string
	ProvidedBy         []string
}
type DisplayPreset struct {
	ID, Label, Broken string
	Default           bool
	Rows              []DisplayRow
}
type View struct {
	SelectedID       string
	Options          []DisplayPreset
	Presets          []DisplayPreset
	Global           []DisplayRow
	Searching, Empty bool
}

// parsedText holds a pre-decoded localized JSON field (either a string literal
// or a locale->text map) so rendering frames never invoke json.Unmarshal.
type parsedText struct {
	isLiteral bool
	literal   string
	shortName string
	variants  map[string]string
	fallback  string
}

var emptyParsedText = &parsedText{}

func parseLocalized(raw json.RawMessage) *parsedText {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return emptyParsedText
	}
	switch trimmed[0] {
	case '"':
		var text string
		if json.Unmarshal(trimmed, &text) == nil {
			return &parsedText{
				isLiteral: true,
				literal:   text,
				shortName: moduleShortName(text),
			}
		}
	case '{':
		var variants map[string]string
		if json.Unmarshal(trimmed, &variants) == nil && len(variants) > 0 {
			keys := make([]string, 0, len(variants))
			for key := range variants {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			fallback := ""
			for _, key := range keys {
				if variants[key] != "" {
					fallback = variants[key]
					break
				}
			}
			return &parsedText{
				variants: variants,
				fallback: fallback,
			}
		}
	}
	return emptyParsedText
}

func localeBase(locale string) string {
	if i := strings.IndexByte(locale, '-'); i >= 0 {
		locale = locale[:i]
	}
	for i := 0; i < len(locale); i++ {
		if c := locale[i]; c >= 'A' && c <= 'Z' {
			return strings.ToLower(locale)
		}
	}
	return locale
}

func (p *parsedText) resolve(locale string) string {
	if p == nil || p == emptyParsedText {
		return ""
	}
	if p.isLiteral {
		return p.literal
	}
	if len(p.variants) == 0 {
		return ""
	}
	if text := p.variants[locale]; text != "" {
		return text
	}
	if base := localeBase(locale); base != locale {
		if text := p.variants[base]; text != "" {
			return text
		}
	}
	if text := p.variants["en"]; text != "" {
		return text
	}
	if text := p.variants["zh"]; text != "" {
		return text
	}
	return p.fallback
}

func (p *parsedText) resolveTitle(locale string) string {
	if p == nil || p == emptyParsedText {
		return ""
	}
	if p.isLiteral {
		return p.shortName
	}
	return p.resolve(locale)
}

type viewCacheKey struct {
	host       uint64
	revision   uint64
	globalPtr  *Row
	globalLen  int
	presetsPtr *Preset
	presetsLen int
	selectedID string
	query      string
	locale     string
}

type viewCacheEntry struct {
	key  viewCacheKey
	view View
	ok   bool
}

type viewCache struct {
	mu      sync.Mutex
	entries [8]viewCacheEntry
	next    int
}

func (c *viewCache) get(key viewCacheKey) (View, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.entries {
		if c.entries[i].ok && c.entries[i].key == key {
			return c.entries[i].view, true
		}
	}
	return View{}, false
}

func (c *viewCache) put(key viewCacheKey, v View) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.entries {
		if c.entries[i].ok && c.entries[i].key == key {
			c.entries[i].view = v
			return
		}
	}
	c.entries[c.next] = viewCacheEntry{key: key, view: v, ok: true}
	c.next = (c.next + 1) % len(c.entries)
}

func trimLowerQuery(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	for i := 0; i < len(trimmed); i++ {
		if c := trimmed[i]; c >= 'A' && c <= 'Z' || c >= 0x80 {
			return strings.ToLower(trimmed)
		}
	}
	return trimmed
}

func selectedPreset(presets []Preset, chosen string) string {
	for _, p := range presets {
		if p.ID == chosen {
			return chosen
		}
	}
	for _, p := range presets {
		if p.Default {
			return p.ID
		}
	}
	if len(presets) > 0 {
		return presets[0].ID
	}
	return ""
}

func localized(raw json.RawMessage, locale string) string {
	return parseLocalized(raw).resolve(locale)
}

func rowMatchesQuery(query string, r Row, title, desc string) bool {
	if query == "" {
		return true
	}
	if strings.ContainsRune(query, '\n') {
		return strings.Contains(strings.ToLower(strings.Join([]string{r.Module, r.EntryID, title, desc, r.MetadataError}, "\n")), query)
	}
	for _, field := range [5]string{r.Module, r.EntryID, title, desc, r.MetadataError} {
		if field != "" && strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func displayRows(rows []Row, locale, query string, failuresFirst bool) []DisplayRow {
	out := make([]DisplayRow, 0, len(rows))
	for _, r := range rows {
		pt, pd := r.parsedTitle, r.parsedDesc
		if pt == nil {
			pt = parseLocalized(r.Title)
		}
		if pd == nil {
			pd = parseLocalized(r.Description)
		}
		title := pt.resolveTitle(locale)
		if title == "" {
			title = moduleShortName(r.Module)
		}
		desc := pd.resolve(locale)
		if !rowMatchesQuery(query, r, title, desc) {
			continue
		}
		out = append(out, DisplayRow{Row: r, Title: title, Description: desc})
	}
	if failuresFirst {
		sort.SliceStable(out, func(i, j int) bool { return out[i].Phase == "failed" && out[j].Phase != "failed" })
	}
	return out
}

func moduleShortName(name string) string {
	if strings.HasPrefix(name, "@") {
		name = name[strings.Index(name, "/")+1:]
	}
	name = strings.TrimPrefix(name, "cordis:")
	name = strings.TrimPrefix(name, "cordis-plugin-")
	if strings.HasPrefix(name, "dsh-") {
		name = strings.TrimPrefix(name, "dsh-")
		if strings.HasPrefix(name, "host-") {
			name = strings.TrimPrefix(name, "host-")
		} else {
			name = strings.TrimPrefix(name, "client-")
		}
	}
	return name
}

// View filters all planes without collapsing repeated module identities. A
// search also exposes matches in non-selected presets, as the original does.
// Snapshots published by Controller cache computed views by (snapshotRev,
// locale, query, selection) so repeated render frames allocate nothing.
func (s Snapshot) View(locale string) View {
	query := trimLowerQuery(s.Query)
	if s.views != nil {
		key := viewCacheKey{
			host:       s.Host,
			revision:   s.Revision,
			globalLen:  len(s.Global),
			presetsLen: len(s.Presets),
			selectedID: s.SelectedID,
			query:      query,
			locale:     locale,
		}
		if len(s.Global) > 0 {
			key.globalPtr = &s.Global[0]
		}
		if len(s.Presets) > 0 {
			key.presetsPtr = &s.Presets[0]
		}
		if cached, ok := s.views.get(key); ok {
			return cached
		}
		v := s.buildView(locale, query)
		s.views.put(key, v)
		return v
	}
	return s.buildView(locale, query)
}

func (s Snapshot) buildView(locale, query string) View {
	v := View{SelectedID: selectedPreset(s.Presets, s.SelectedID), Searching: query != ""}
	v.Global = displayRows(s.Global, locale, query, true)
	// Global disablement is preserved. Agent presets may still provide the same
	// module per session, so expose those enabling presets as separate provenance.
	for i := range v.Global {
		row := &v.Global[i]
		if row.Enabled != "disabled" || row.Phase == "failed" {
			continue
		}
		for _, preset := range s.Presets {
			for _, entry := range preset.Rows {
				if entry.Module == row.Module && entry.Enabled == "enabled" {
					label := preset.Name
					if label == "" {
						label = preset.ID
					}
					row.ProvidedBy = append(row.ProvidedBy, label)
					break
				}
			}
		}
	}
	count := len(v.Global)
	var others []DisplayPreset
	for _, p := range s.Presets {
		label := p.Name
		if label == "" || label == p.ID {
			label = p.ID
		} else {
			label += " (" + p.ID + ")"
		}
		option := DisplayPreset{ID: p.ID, Label: label, Broken: p.Broken, Default: p.Default}
		v.Options = append(v.Options, option)
		if p.ID == v.SelectedID || v.Searching {
			option.Rows = displayRows(p.Rows, locale, query, false)
		}
		if p.ID == v.SelectedID {
			v.Presets = append(v.Presets, option)
			count += len(option.Rows)
		} else if v.Searching && len(option.Rows) > 0 {
			others = append(others, option)
			count += len(option.Rows)
		}
	}
	v.Presets = append(v.Presets, others...)
	v.Empty = count == 0
	return v
}
