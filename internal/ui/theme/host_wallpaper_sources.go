package theme

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
	"whaleisle/internal/desktop/wallpapers"
)

var hostCatalogID = regexp.MustCompile(`(?i)^catalog-[0-9a-z]+$`)

func hostWallpaperSources(raw json.RawMessage) []wallpapers.Source {
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	if source, ok := fields["wallpaperSources"]; ok && string(source) != "null" {
		var rows []wallpapers.Source
		_ = json.Unmarshal(source, &rows)
		return sanitizeHostSources(rows)
	}
	rows := wallpapers.Sources()
	var catalogs []string
	_ = json.Unmarshal(fields["wallpaperCatalogUrls"], &catalogs)
	for _, href := range catalogs {
		rows = append(rows, wallpapers.Source{Kind: "catalog", URL: href})
	}
	return sanitizeHostSources(rows)
}

// Pure boundary normalization follows ui-theme/theme-settings.ts; legacy
// catalogs are projected for reading, never automatically written or migrated.
func sanitizeHostSources(rows []wallpapers.Source) []wallpapers.Source {
	result := []wallpapers.Source{}
	ids, urls := map[string]bool{}, map[string]bool{}
	catalogs := 0
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		units := utf16.Encode([]rune(name))
		if len(units) > 40 {
			name = string(utf16.Decode(units[:40]))
		}
		switch row.Kind {
		case "bing", "wallhaven":
			if ids[row.Kind] {
				continue
			}
			if name == "" {
				if row.Kind == "bing" {
					name = "必应"
				} else {
					name = "Wallhaven"
				}
			}
			result = append(result, wallpapers.Source{ID: row.Kind, Kind: row.Kind, Name: name})
			ids[row.Kind] = true
		case "catalog":
			href := strings.TrimSpace(row.URL)
			parsed, err := url.Parse(href)
			if err != nil || parsed.Scheme != "https" || parsed.Host == "" || len(utf16.Encode([]rune(href))) > 500 || urls[href] || catalogs >= 5 {
				continue
			}
			if parsed.Path == "" {
				parsed.Path = "/"
			}
			href = parsed.String()
			if urls[href] {
				continue
			}
			if name == "" {
				name = parsed.Hostname()
			}
			id := strings.TrimSpace(row.ID)
			if !hostCatalogID.MatchString(id) || ids[id] {
				hash := uint32(2166136261)
				for _, unit := range utf16.Encode([]rune(href)) {
					hash ^= uint32(unit)
					hash *= 16777619
				}
				id = "catalog-" + strconv.FormatUint(uint64(hash), 36)
				base := id
				for suffix := 2; ids[id]; suffix++ {
					id = fmt.Sprintf("%s-%d", base, suffix)
				}
			}
			result = append(result, wallpapers.Source{ID: id, Kind: "catalog", Name: name, URL: href})
			ids[id], urls[href] = true, true
			catalogs++
		}
	}
	return result
}

func sanitizeHostFavorites(rows []wallpapers.Item) []wallpapers.Item {
	result := []wallpapers.Item{}
	seen := map[string]bool{}
	for _, row := range rows {
		row.ID = strings.TrimSpace(row.ID)
		row.SourceID = strings.TrimSpace(row.SourceID)
		row.Title = strings.TrimSpace(row.Title)
		row.ThumbURL = strings.TrimSpace(row.ThumbURL)
		row.ImageURL = strings.TrimSpace(row.ImageURL)
		if row.ID == "" || row.SourceID == "" || row.Title == "" || row.ThumbURL == "" || row.ImageURL == "" || seen[row.SourceID+":"+row.ID] {
			continue
		}
		seen[row.SourceID+":"+row.ID] = true
		result = append(result, row)
		if len(result) == 100 {
			break
		}
	}
	return result
}
