// Package components discovers launcher-managed component rows for the
// native launcher UI (Electron shell:components-list parity, read-only first).
package components

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"whaleisle/internal/store/storelocale"
)

// Row is one catalog/install entry shown in launcher 组件 / 我的组件.
type Row struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Version          string `json:"version"`
	InstalledVersion string `json:"installedVersion"`
	PreviousVersion  string `json:"previousVersion"`
	Kind             string `json:"kind"`
	Source           string `json:"source"`
	State            string `json:"state"`
	UpdateAvailable  bool   `json:"updateAvailable"`
	Message          string `json:"message"`
	PID              int    `json:"pid"`
	URL              string `json:"url"`
	Orphan           bool   `json:"orphan"`
}

type catalogFile struct {
	Manifest struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	} `json:"manifest"`
}

type installedFile struct {
	Version  string `json:"version"`
	Previous string `json:"previous"`
}

// ListUserData scans <userData>/components for known component roots.
// Whalebridge is the first official component (Electron whalebridge.js).
func ListUserData(userData string) []Row {
	root := filepath.Join(userData, "components")
	var out []Row
	if HostBundleSupported() {
		out = append(out, hostBundleRow(userData))
	}
	if row, ok := whalebridgeRow(filepath.Join(root, "whalebridge")); ok {
		out = append(out, row)
	}
	generic := listGeneric(userData)
	out = append(out, generic...)
	seen := map[string]bool{}
	for _, row := range generic {
		seen[row.ID] = true
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "whalebridge" || e.Name() == HostComponentID || seen[e.Name()] {
			continue
		}
		if row, ok := genericRow(filepath.Join(root, e.Name()), e.Name()); ok {
			out = append(out, row)
		}
	}
	return out
}

func whalebridgeRow(dir string) (Row, bool) {
	cat := readCatalog(filepath.Join(dir, "catalog.json"))
	inst := readInstalled(filepath.Join(dir, "installed.json"))

	name := cat.Manifest.Name
	if name == "" {
		name = storelocale.T(storelocale.KeyStoreComponentsWhalebridge)
	}
	desc := cat.Manifest.Description
	if desc == "" {
		desc = storelocale.T(storelocale.KeyStoreComponentsConnectApiProvidersAndSubscriptionAccou)
	}
	ver := cat.Manifest.Version
	if ver == "" || compareVersions(inst.Version, ver) > 0 {
		ver = inst.Version
	}
	state := "available"
	if inst.Version != "" {
		state = "stopped"
		if running := readStatePID(filepath.Join(dir, "data", "state.json")); running {
			state = "running"
		}
	}
	update := inst.Version != "" && ver != "" && compareVersions(ver, inst.Version) > 0
	return Row{
		ID:               "whalebridge",
		Name:             name,
		Description:      desc,
		Version:          ver,
		InstalledVersion: inst.Version,
		PreviousVersion:  inst.Previous,
		Kind:             "service",
		Source:           "official",
		State:            state,
		UpdateAvailable:  update,
	}, true
}

// newestVersionDir returns the highest semver-ish subdirectory name under versions/.
func newestVersionDir(dir string) string {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	best := ""
	for _, e := range ents {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		name := e.Name()
		if best == "" || compareVersions(name, best) > 0 {
			best = name
		}
	}
	return best
}

func genericRow(dir, id string) (Row, bool) {
	cat := readCatalog(filepath.Join(dir, "catalog.json"))
	inst := readInstalled(filepath.Join(dir, "installed.json"))
	manPath := filepath.Join(dir, "manifest.json")
	if cat.Manifest.ID == "" {
		if b, err := os.ReadFile(manPath); err == nil {
			_ = json.Unmarshal(b, &cat.Manifest)
		}
	}
	if cat.Manifest.ID == "" && inst.Version == "" {
		return Row{}, false
	}
	name := cat.Manifest.Name
	if name == "" {
		name = id
	}
	ver := cat.Manifest.Version
	if ver == "" {
		ver = inst.Version
	}
	state := "available"
	if inst.Version != "" {
		state = "stopped"
	}
	return Row{
		ID:               id,
		Name:             name,
		Description:      cat.Manifest.Description,
		Version:          ver,
		InstalledVersion: inst.Version,
		Kind:             "tool",
		Source:           "bundled",
		State:            state,
		UpdateAvailable:  inst.Version != "" && ver != "" && compareVersions(ver, inst.Version) > 0,
	}, true
}

func readCatalog(path string) catalogFile {
	var c catalogFile
	b, err := os.ReadFile(path)
	if err != nil {
		return c
	}
	_ = json.Unmarshal(b, &c)
	return c
}

func readInstalled(path string) installedFile {
	var i installedFile
	b, err := os.ReadFile(path)
	if err != nil {
		return i
	}
	_ = json.Unmarshal(b, &i)
	return i
}

func readStatePID(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var st struct {
		Ready bool `json:"ready"`
		PID   int  `json:"pid"`
	}
	if json.Unmarshal(b, &st) != nil || !st.Ready || st.PID <= 0 {
		return false
	}
	proc, err := os.FindProcess(st.PID)
	if err != nil {
		return false
	}
	// On Windows FindProcess always succeeds; Signal(0) is not portable.
	// Treat ready+pid as running for UI until a cross-platform probe lands.
	_ = proc
	return true
}

func compareVersions(a, b string) int {
	pa := versionParts(a)
	pb := versionParts(b)
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}

func versionParts(s string) []int {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if s == "" {
		return nil
	}
	bits := strings.Split(s, ".")
	out := make([]int, len(bits))
	for i, b := range bits {
		n := 0
		for _, ch := range b {
			if ch < '0' || ch > '9' {
				break
			}
			n = n*10 + int(ch-'0')
		}
		out[i] = n
	}
	return out
}
