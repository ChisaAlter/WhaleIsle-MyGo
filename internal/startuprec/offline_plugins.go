package startuprec

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
)

// OfflinePlugins uses the desktop config and the actual CLI web profile. No
// healthy Host or pluginManager RPC is needed to repair a broken startup.
type OfflinePlugins struct {
	ProfileDir   string
	LoadDisabled func() []string
	SaveDisabled func([]string) error
	Uninstall    func(string) error
}

func immutablePlugin(name string) bool {
	switch name {
	case "@deepseek-ai/dsh-base", "@deepseek-ai/dsh-web-app", "dshmarket", "@xmanrui/dsh-im", "dsh-im", "xmanrui-dsh-im", "dshbot", "dsh-bot", "dsh-whale", "dshwhale", "dsh-remote", "dsh-usage-panel":
		return true
	}
	return isBuiltin(name)
}

func (p OfflinePlugins) manifest() (map[string]any, error) {
	raw, err := os.ReadFile(filepath.Join(p.ProfileDir, "package.json"))
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err = json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errors.New("invalid web profile")
	}
	return m, nil
}
func profileParts(m map[string]any) (map[string]any, map[string]any, []any) {
	dsh, _ := m["dsh"].(map[string]any)
	if dsh == nil {
		dsh = map[string]any{}
		m["dsh"] = dsh
	}
	profile, _ := dsh["profile"].(map[string]any)
	if profile == nil {
		profile = map[string]any{}
		dsh["profile"] = profile
	}
	bundles, _ := profile["bundles"].([]any)
	return dsh, profile, bundles
}
func (p OfflinePlugins) write(m map[string]any) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	file := filepath.Join(p.ProfileDir, "package.json")
	tmp, err := os.CreateTemp(p.ProfileDir, ".plugin-profile-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(append(raw, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), file)
}
func (p OfflinePlugins) Inventory() ([]Plugin, error) {
	m, err := p.manifest()
	if os.IsNotExist(err) {
		return []Plugin{}, nil
	}
	if err != nil {
		return nil, err
	}
	deps, _ := m["dependencies"].(map[string]any)
	_, _, bundles := profileParts(m)
	disabled := []string{}
	if p.LoadDisabled != nil {
		disabled = p.LoadDisabled()
	}
	rows := []Plugin{}
	for name := range deps {
		enabled := false
		for _, b := range bundles {
			if b == name {
				enabled = true
			}
		}
		mutable := packageName.MatchString(name) && !immutablePlugin(name)
		restriction := ""
		if !mutable {
			restriction = "Built-in / template"
		}
		rows = append(rows, Plugin{Name: name, Enabled: enabled && !slices.Contains(disabled, name), Mutable: mutable, Removable: mutable && name != "dsh-project", Restriction: restriction})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows, nil
}
func (p OfflinePlugins) validate(name string, remove bool) error {
	rows, err := p.Inventory()
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.Name == name {
			if !row.Mutable || (remove && !row.Removable) {
				return errors.New("preset plugin cannot be changed")
			}
			return nil
		}
	}
	return fmt.Errorf("plugin %q is not in the web profile", name)
}

// ApplyDisabled is the original applyDisabledBundles runtime consumer. It
// removes disabled user bundles while preserving dependency entries and all
// unrelated profile fields, including immutable template bundles.
func (p OfflinePlugins) ApplyDisabled(disabled []string) error {
	if len(disabled) == 0 {
		return nil
	}
	m, err := p.manifest()
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	_, profile, bundles := profileParts(m)
	next := make([]any, 0, len(bundles))
	for _, b := range bundles {
		name, ok := b.(string)
		if !ok || immutablePlugin(name) || !slices.Contains(disabled, name) {
			next = append(next, b)
		}
	}
	if len(next) == len(bundles) {
		return nil
	}
	profile["bundles"] = next
	return p.write(m)
}
func (p OfflinePlugins) SetEnabled(name string, enabled bool) error {
	if err := p.validate(name, false); err != nil {
		return err
	}
	if p.LoadDisabled == nil || p.SaveDisabled == nil {
		return errors.New("desktop plugin config unavailable")
	}
	disabled := slices.Clone(p.LoadDisabled())
	if enabled {
		disabled = slices.DeleteFunc(disabled, func(v string) bool { return v == name })
		m, err := p.manifest()
		if err != nil {
			return err
		}
		_, profile, bundles := profileParts(m)
		found := false
		for _, b := range bundles {
			if b == name {
				found = true
			}
		}
		if !found {
			profile["bundles"] = append(bundles, name)
			if err = p.write(m); err != nil {
				return err
			}
		}
	} else if !slices.Contains(disabled, name) {
		disabled = append(disabled, name)
	}
	if err := p.ApplyDisabled(disabled); err != nil {
		return err
	}
	return p.SaveDisabled(disabled)
}
func (p OfflinePlugins) Remove(name string) error {
	if err := p.validate(name, true); err != nil {
		return err
	}
	if p.Uninstall == nil || p.LoadDisabled == nil || p.SaveDisabled == nil {
		return errors.New("bundled plugin uninstaller unavailable")
	}
	if err := p.Uninstall(name); err != nil {
		return err
	}
	return p.SaveDisabled(slices.DeleteFunc(slices.Clone(p.LoadDisabled()), func(v string) bool { return v == name }))
}
