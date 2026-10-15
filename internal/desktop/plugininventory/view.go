package plugininventory

import (
	"encoding/json"
	"sort"
	"strings"
)

type DisplayRow struct {
	Row
	Title, Description string
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
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var variants map[string]string
	if json.Unmarshal(raw, &variants) != nil {
		return ""
	}
	base := strings.Split(strings.ToLower(locale), "-")[0]
	for _, key := range []string{locale, base, "en", "zh"} {
		if text = variants[key]; text != "" {
			return text
		}
	}
	keys := make([]string, 0, len(variants))
	for key := range variants {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if variants[key] != "" {
			return variants[key]
		}
	}
	return ""
}
func displayRows(rows []Row, locale, query string, failuresFirst bool) []DisplayRow {
	out := make([]DisplayRow, 0, len(rows))
	for _, r := range rows {
		d := DisplayRow{Row: r, Title: localized(r.Title, locale), Description: localized(r.Description, locale)}
		if d.Title == "" {
			d.Title = r.Module
		}
		if query != "" && !strings.Contains(strings.ToLower(strings.Join([]string{r.Module, r.EntryID, d.Title, d.Description, r.MetadataError}, "\n")), query) {
			continue
		}
		out = append(out, d)
	}
	if failuresFirst {
		sort.SliceStable(out, func(i, j int) bool { return out[i].Phase == "failed" && out[j].Phase != "failed" })
	}
	return out
}

// View filters all planes without collapsing repeated module identities. A
// search also exposes matches in non-selected presets, as the original does.
func (s Snapshot) View(locale string) View {
	query := strings.ToLower(strings.TrimSpace(s.Query))
	v := View{SelectedID: selectedPreset(s.Presets, s.SelectedID), Searching: query != ""}
	v.Global = displayRows(s.Global, locale, query, true)
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
		option.Rows = displayRows(p.Rows, locale, query, false)
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
