package components

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"whaleisle/internal/store/storelocale"
)

var genericID = regexp.MustCompile(`(?i)^[a-z0-9][a-z0-9._-]{0,63}$`)

type genericManifest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
	Entry       string `json:"entry"`
}

type genericRecord struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Previous  string `json:"previous"`
	State     string `json:"state"`
	PID       int    `json:"pid"`
	URL       string `json:"url"`
	Source    string `json:"source"`
	LastError string `json:"lastError"`
}

func genericEntry(dir, entry string) string {
	entry = strings.ReplaceAll(entry, `\`, "/")
	if entry == "" || filepath.IsAbs(entry) || strings.HasPrefix(entry, "/") || (len(entry) > 1 && entry[1] == ':') {
		return ""
	}
	base, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	target := filepath.Join(base, filepath.FromSlash(entry))
	rel, err := filepath.Rel(base, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return target
}

func readGenericManifest(dir string) (genericManifest, bool) {
	var m genericManifest
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil || json.Unmarshal(data, &m) != nil || !genericID.MatchString(m.ID) || m.Version == "" {
		return m, false
	}
	entry := genericEntry(dir, m.Entry)
	info, err := os.Stat(entry)
	if entry == "" || err != nil || !info.Mode().IsRegular() {
		return m, false
	}
	if m.Name == "" {
		m.Name = m.ID
	}
	if m.Kind != "tool" {
		m.Kind = "service"
	}
	return m, true
}

// Read the original registry.json plus bundled samples/<id>/<version> catalog.
// This path never starts Node or a component simply to render the launcher.
func genericDiskRows(userData, samples string) []Row {
	rows := map[string]Row{}
	var reg struct {
		Components map[string]genericRecord `json:"components"`
	}
	if data, err := os.ReadFile(filepath.Join(userData, "components", "registry.json")); err == nil {
		_ = json.Unmarshal(data, &reg)
	}
	for id, rec := range reg.Components {
		if !genericID.MatchString(id) || rec.Version == "" || id == "whalebridge" || id == HostComponentID {
			continue
		}
		state := rec.State
		switch state {
		case "installed", "stopped", "running", "error":
		default:
			state = "installed"
		}
		if state == "running" && !genericPIDAlive(rec.PID) {
			state = "stopped"
			rec.PID = 0
			rec.URL = ""
			rec.LastError = storelocale.T(storelocale.KeyStoreComponentsTheProcessExitedLauncherRestartOrExter)
		}
		name := rec.Name
		if name == "" {
			name = id
		}
		source := rec.Source
		if source == "" {
			source = "installed"
		}
		rows[id] = Row{ID: id, Name: name, Version: rec.Version, InstalledVersion: rec.Version, PreviousVersion: rec.Previous, Kind: "service", Source: source, State: state, Message: rec.LastError, PID: rec.PID, URL: rec.URL, Orphan: true}
	}
	ids, _ := os.ReadDir(samples)
	for _, idDir := range ids {
		id := idDir.Name()
		if !idDir.IsDir() || !genericID.MatchString(id) || id == "whalebridge" || id == HostComponentID {
			continue
		}
		versions, _ := os.ReadDir(filepath.Join(samples, id))
		var latest genericManifest
		for _, v := range versions {
			if !v.IsDir() {
				continue
			}
			m, ok := readGenericManifest(filepath.Join(samples, id, v.Name()))
			if !ok || m.ID != id || m.Version != v.Name() {
				continue
			}
			if latest.Version == "" || compareVersions(m.Version, latest.Version) > 0 {
				latest = m
			}
		}
		if latest.Version == "" {
			continue
		}
		row, exists := rows[id]
		if !exists {
			row = Row{ID: id, State: "available"}
		}
		row.Name = latest.Name
		row.Version = latest.Version
		row.Description = latest.Description
		row.Kind = latest.Kind
		row.Source = "bundled"
		row.Orphan = false
		row.UpdateAvailable = row.InstalledVersion != "" && compareVersions(row.Version, row.InstalledVersion) > 0
		rows[id] = row
	}
	keys := make([]string, 0, len(rows))
	for id := range rows {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]Row, 0, len(rows))
	for _, id := range keys {
		out = append(out, rows[id])
	}
	return out
}
