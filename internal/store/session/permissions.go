package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"whaleisle/internal/dsh/api"
)

// PermissionOption is one row in the composer permission preset picker.
type PermissionOption struct {
	Label string
	Value string
}

func flattenPermissionCatalog(cat api.PermissionCatalog) []PermissionOption {
	var out []PermissionOption
	seen := map[string]struct{}{}
	appendItems := func(items []api.PermissionCatalogOptionsItem) {
		for _, it := range items {
			if it.Value == "" {
				continue
			}
			if _, ok := seen[it.Value]; ok {
				continue
			}
			seen[it.Value] = struct{}{}
			label := it.Name
			if label == "" {
				label = it.Value
			}
			out = append(out, PermissionOption{Label: label, Value: it.Value})
		}
	}
	appendItems(cat.DefaultOptions)
	appendItems(cat.Options)
	return out
}

func labelForPresetValue(value string, options []PermissionOption) string {
	for _, o := range options {
		if o.Value == value {
			return o.Label
		}
	}
	if value != "" {
		return value
	}
	return ""
}

func permissionOptionByLabel(label string, options []PermissionOption) (PermissionOption, bool) {
	for _, o := range options {
		if o.Label == label {
			return o, true
		}
	}
	return PermissionOption{}, false
}

func parseDefaultPreset(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var v struct {
		DefaultPreset string `json:"defaultPreset"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return ""
	}
	return v.DefaultPreset
}

func findPermissionNamespace(desc api.SettingsDescribeValue) (ns string, revision float64, preset string) {
	for _, item := range desc.Namespaces {
		if !strings.Contains(strings.ToLower(item.Ns), "permission") {
			continue
		}
		return item.Ns, item.Revision, parseDefaultPreset(item.Value)
	}
	return "", 0, ""
}

// RefreshPermissionCatalog loads permissionPresets/catalog and caches the permission settings namespace from settings/describe.
func (s *Store) RefreshPermissionCatalog(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		var cat api.PermissionCatalog
		err := client.Call(ctx, "permissionPresets/catalog", map[string]any{}, &cat)
		var desc api.SettingsDescribeValue
		descErr := client.Call(ctx, "settings/describe", nil, &desc)

		s.mu.Lock()
		if err != nil {
			s.permissionsErr = err
			s.mu.Unlock()
			s.fire()
			return
		}
		s.permissionsErr = nil
		s.permissionOptions = flattenPermissionCatalog(cat)
		if descErr == nil {
			s.permSettingsWritable = desc.Writable
			if ns, rev, livePreset := findPermissionNamespace(desc); ns != "" {
				s.permSettingsNS = ns
				s.permSettingsRevision = rev
				if livePreset != "" {
					if lbl := labelForPresetValue(livePreset, s.permissionOptions); lbl != "" {
						s.selectedPermissionLabel = lbl
					}
				}
			}
		}
		if s.selectedPermissionLabel == "" {
			if lbl := labelForPresetValue(cat.DefaultPreset, s.permissionOptions); lbl != "" {
				s.selectedPermissionLabel = lbl
			} else if len(s.permissionOptions) > 0 {
				s.selectedPermissionLabel = s.permissionOptions[0].Label
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// SelectPermissionPreset applies settings/update on the permission namespace when writable; selection is committed only after Host persistence succeeds.
func (s *Store) SelectPermissionPreset(ctx context.Context, label string) {
	s.mu.Lock()
	opt, ok := permissionOptionByLabel(label, s.permissionOptions)
	client := s.client
	prev := s.selectedPermissionLabel
	ns := s.permSettingsNS
	rev := s.permSettingsRevision
	writable := s.permSettingsWritable
	s.mu.Unlock()
	if !ok || client == nil || label == prev {
		return
	}
	go func() {

		if ns == "" || !writable {
			s.mu.Lock()
			if ns == "" {
				s.lastAction = "permission preset not saved: no settings namespace"
			} else {
				s.lastAction = "permission preset not saved: settings read-only"
			}
			s.permissionsErr = fmt.Errorf("%s", s.lastAction)
			s.mu.Unlock()
			s.fire()
			return
		}
		rawPreset, mErr := json.Marshal(opt.Value)
		if mErr != nil {
			s.mu.Lock()
			s.lastAction = mErr.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		patch := map[string]json.RawMessage{
			"defaultPreset": rawPreset,
		}
		args := map[string]any{
			"ns":               ns,
			"patch":            patch,
			"expectedRevision": rev,
		}
		var out api.SettingsNamespaceView
		err := client.Call(ctx, "settings/update", args, &out)
		if err == nil && parseDefaultPreset(out.Value) == "" {
			err = fmt.Errorf("permission save returned no committed preset")
		}
		s.mu.Lock()
		if s.client != client || s.permSettingsNS != ns {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.permissionsErr = err
			s.lastAction = err.Error()
			s.mu.Unlock()
			s.fire()
			return
		}
		s.permSettingsRevision = out.Revision
		if live := parseDefaultPreset(out.Value); live != "" {
			if lbl := labelForPresetValue(live, s.permissionOptions); lbl != "" {
				s.selectedPermissionLabel = lbl
			}
		}
		s.permissionsErr = nil
		s.lastAction = "permission preset updated"
		s.mu.Unlock()
		s.fire()
	}()
}
