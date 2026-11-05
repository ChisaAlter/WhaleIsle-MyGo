package theme

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var themeSlug = regexp.MustCompile(`[^a-z0-9]+`)

func familyID(value string) string {
	value = strings.Trim(themeSlug.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "-"), "-")
	if value == "" {
		return "custom-theme"
	}
	return value
}

func (p Preferences) UniqueFamilyID(base string) string {
	base = familyID(base)
	seen := map[string]bool{}
	for _, f := range p.Families() {
		seen[f.ID] = true
	}
	if !seen[base] {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !seen[candidate] {
			return candidate
		}
	}
}

func (p Preferences) DuplicateFamily(f Family) Family {
	f.Light, f.Dark = f.Light.Clone(), f.Dark.Clone()
	f.ID = p.UniqueFamilyID(f.ID + "-copy")
	f.Name += " Copy"
	return f
}

func canonicalFamily(f Family) Family {
	if f.Skin == SkinDefault {
		f.Skin = ""
	}
	f.Light, f.Dark = f.Light.Clone(), f.Dark.Clone()
	for _, s := range []*Seed{&f.Light, &f.Dark} {
		if s.Contrast == nil {
			value := float64(46)
			s.Contrast = &value
		}
		for key, value := range s.Overrides {
			if value == "" {
				delete(s.Overrides, key)
			}
		}
		if len(s.Overrides) == 0 {
			s.Overrides = nil
		}
	}
	return f
}

func SerializeFamily(f Family) string {
	f = canonicalFamily(f)
	value := struct {
		Family
		Origin string `json:"origin"`
	}{f, "custom"}
	data, _ := json.MarshalIndent(value, "", "  ")
	return string(data) + "\n"
}

func (p Preferences) ImportFamily(raw string) (Family, error) {
	var f Family
	if err := json.Unmarshal([]byte(raw), &f); err != nil {
		return f, err
	}
	if f.ID == "" {
		f.ID = f.Name
	}
	f.ID = p.UniqueFamilyID(f.ID)
	f = canonicalFamily(f)
	check := p.Clone()
	check.Custom = append(check.Custom, f)
	if err := check.Validate(); err != nil {
		return Family{}, err
	}
	return f, nil
}

func validFamilyWallpaper(value string) bool {
	if len(value) > 16<<20 {
		return false
	}
	header, data, ok := strings.Cut(value, ",")
	if !ok {
		return false
	}
	switch header {
	case "data:image/png;base64", "data:image/jpeg;base64", "data:image/webp;base64", "data:image/gif;base64":
	default:
		return false
	}
	_, err := base64.StdEncoding.DecodeString(data)
	return err == nil && data != ""
}
