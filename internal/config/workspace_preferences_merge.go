package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// ReadWorkspaceNavigationPreferences reads a fresh view baseline without
// replacing the baseline of unrelated Config.Update callers.
func (s *Store) ReadWorkspaceNavigationPreferences(hostKey string) (WorkspaceNavigationPreferences, error) {
	s.mu.Lock()
	base := s.baseline
	cached, cachedErr := s.workspaceNavigationPreferencesLocked(hostKey)
	s.mu.Unlock()

	commitMu.Lock()
	cur, err := readBaseline(s.path)
	commitMu.Unlock()
	if err != nil {
		return WorkspaceNavigationPreferences{}, err
	}
	if cur.exists == base.exists && bytes.Equal(cur.data, base.data) {
		return cached, cachedErr
	}
	if !cur.exists {
		return WorkspaceNavigationPreferences{}, nil
	}
	data := bytes.TrimPrefix(cur.data, []byte{0xEF, 0xBB, 0xBF})
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return WorkspaceNavigationPreferences{}, fmt.Errorf("config: %s: %w", s.path, err)
	}
	var out WorkspaceNavigationPreferences
	root := map[string]json.RawMessage{}
	if navRaw := raw[workspaceNavigationKey]; len(navRaw) > 0 {
		if err := json.Unmarshal(navRaw, &root); err != nil {
			return out, err
		}
	}
	if hostRaw := root[hostKey]; len(hostRaw) > 0 {
		if err := json.Unmarshal(hostRaw, &out); err != nil {
			return out, err
		}
	}
	return out, nil
}

// MergeWorkspaceNavigationPreferences applies only base->next view changes to
// Merge refreshes the baseline from disk when config.json was updated by
// another process (such as the shared desktop pet) before applying fn and
// writing the merged configuration atomically.
func (s *Store) Merge(fn func(*Config)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	unlock, err := s.lockCommit()
	if err != nil {
		return err
	}
	defer unlock()
	var latest *Store
	if ok, matchErr := s.baseline.matches(s.path); matchErr == nil && ok {
		latest = &Store{
			path:     s.path,
			raw:      s.raw,
			cfg:      s.cfg,
			baseline: s.baseline,
		}
	} else {
		latest, err = Open(filepath.Dir(s.path))
		if err != nil {
			return err
		}
	}
	if s.baseline.exists && !latest.baseline.exists {
		return fmt.Errorf("%w: configuration file was removed", ErrConflict)
	}
	candidate := cloneConfig(latest.cfg)
	if fn != nil {
		fn(&candidate)
	}
	if candidate.SkipUserPlugins != latest.cfg.SkipUserPlugins {
		candidate.SkipUserPlugins = latest.cfg.SkipUserPlugins
	}
	next := normalize(cloneConfig(candidate))
	encoded, err := json.Marshal(next)
	if err != nil {
		return err
	}
	var known map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &known); err != nil {
		return err
	}
	merged := mergeKnownConfig(latest.raw, known)
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	if err := latest.checkBaseline(); err != nil {
		return err
	}
	rename := s.rename
	if rename == nil {
		rename = os.Rename
	}
	guardedRename := func(oldpath, newpath string) error {
		if err := latest.checkBaseline(); err != nil {
			return err
		}
		return rename(oldpath, newpath)
	}
	if err := writeAtomic(s.path, data, guardedRename); err != nil {
		return err
	}
	s.raw = merged
	s.setCfgLocked(next)
	s.baseline = fileBaseline{exists: true, data: data}
	return nil
}

// MergeWorkspaceNavigationPreferences merges one Host's view preferences onto
// a fresh configuration. Concurrent changes to the same field are conflicts;
// unrelated fields, other Hosts, and unknown configuration keys survive.
// The caller retains base on failure and acknowledges next on success, since
// its in-memory view may not yet contain independently merged disk changes.
func (s *Store) MergeWorkspaceNavigationPreferences(hostKey string, base, next WorkspaceNavigationPreferences) (WorkspaceNavigationPreferences, error) {
	if strings.TrimSpace(hostKey) == "" {
		return WorkspaceNavigationPreferences{}, fmt.Errorf("workspace preference Host is missing")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	unlock, err := s.lockCommit()
	if err != nil {
		return WorkspaceNavigationPreferences{}, err
	}
	defer unlock()
	var latest *Store
	if ok, matchErr := s.baseline.matches(s.path); matchErr == nil && ok {
		latest = &Store{
			path:     s.path,
			raw:      s.raw,
			cfg:      s.cfg,
			baseline: s.baseline,
		}
	} else {
		latest, err = Open(filepath.Dir(s.path))
		if err != nil {
			return WorkspaceNavigationPreferences{}, err
		}
	}
	if s.baseline.exists && !latest.baseline.exists {
		return WorkspaceNavigationPreferences{}, fmt.Errorf("%w: configuration file was removed", ErrConflict)
	}
	root := map[string]json.RawMessage{}
	if raw := latest.raw[workspaceNavigationKey]; len(raw) > 0 {
		if err = json.Unmarshal(raw, &root); err != nil {
			return WorkspaceNavigationPreferences{}, err
		}
	}
	if root == nil {
		root = map[string]json.RawMessage{}
	}
	host := map[string]json.RawMessage{}
	if raw := root[hostKey]; len(raw) > 0 {
		if err = json.Unmarshal(raw, &host); err != nil {
			return WorkspaceNavigationPreferences{}, err
		}
	}
	if host == nil {
		host = map[string]json.RawMessage{}
	}
	current, err := latest.WorkspaceNavigationPreferences(hostKey)
	if err != nil {
		return WorkspaceNavigationPreferences{}, err
	}
	before, requested, disk := workspacePreferenceValues(base), workspacePreferenceValues(next), workspacePreferenceValues(current)
	for _, field := range []string{"groupBy", "orderBy", "archivedFilter", "collapsed", "sessionOrders"} {
		// Missing/empty archive values mean default. Preserve their original
		// representation when this view did not edit the archive choice.
		if field == "archivedFilter" && sameWorkspacePreferenceValue(before[field], requested[field]) {
			continue
		}
		host[field], err = mergeWorkspacePreferenceValue(field, before[field], requested[field], disk[field])
		if err != nil {
			return WorkspaceNavigationPreferences{}, err
		}
	}
	hostRaw, err := json.Marshal(host)
	if err != nil {
		return WorkspaceNavigationPreferences{}, err
	}
	var result WorkspaceNavigationPreferences
	if err = json.Unmarshal(hostRaw, &result); err != nil {
		return WorkspaceNavigationPreferences{}, err
	}
	root[hostKey] = hostRaw
	rootRaw, err := json.Marshal(root)
	if err != nil {
		return WorkspaceNavigationPreferences{}, err
	}
	merged := make(map[string]json.RawMessage, len(latest.raw)+1)
	for key, value := range latest.raw {
		merged[key] = value
	}
	merged[workspaceNavigationKey] = rootRaw
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return WorkspaceNavigationPreferences{}, err
	}
	// Keep the ordinary final comparison, now against the exact bytes just read.
	// Another process changing the file during merge must cause a retry, never
	// an unchecked replacement of that process's settings.
	if err = latest.checkBaseline(); err != nil {
		return WorkspaceNavigationPreferences{}, err
	}
	rename := s.rename
	if rename == nil {
		rename = os.Rename
	}
	guardedRename := func(oldpath, newpath string) error {
		if err := latest.checkBaseline(); err != nil {
			return err
		}
		return rename(oldpath, newpath)
	}
	if err = writeAtomic(s.path, data, guardedRename); err != nil {
		return WorkspaceNavigationPreferences{}, err
	}
	s.raw = merged
	s.setCfgLocked(latest.cfg)
	s.baseline = fileBaseline{exists: true, data: data}
	return result, nil
}

func workspacePreferenceValues(p WorkspaceNavigationPreferences) map[string]json.RawMessage {
	// Session view state materializes these defaults on its first edit. An
	// absent serialized value is the same user choice, not a grouping/order edit.
	if p.GroupBy == "" {
		p.GroupBy = "workspace"
	}
	if p.OrderBy == "" {
		p.OrderBy = "manual"
	}
	if p.ArchivedFilter != "show" && p.ArchivedFilter != "only" {
		p.ArchivedFilter = "default"
	}
	if p.Collapsed == nil {
		p.Collapsed = map[string]bool{}
	}
	if p.SessionOrders == nil {
		p.SessionOrders = map[string][]string{}
	}
	values := map[string]json.RawMessage{}
	// These concrete JSON types cannot fail to marshal.
	values["groupBy"], _ = json.Marshal(p.GroupBy)
	values["orderBy"], _ = json.Marshal(p.OrderBy)
	values["archivedFilter"], _ = json.Marshal(p.ArchivedFilter)
	values["collapsed"], _ = json.Marshal(p.Collapsed)
	values["sessionOrders"], _ = json.Marshal(p.SessionOrders)
	return values
}

func sameWorkspacePreferenceValue(a, b json.RawMessage) bool {
	if len(a) == 0 || len(b) == 0 {
		return len(a) == len(b)
	}
	var left, right any
	if json.Unmarshal(a, &left) != nil || json.Unmarshal(b, &right) != nil {
		return false
	}
	return reflect.DeepEqual(left, right)
}

func mergeWorkspacePreferenceValue(field string, base, next, current json.RawMessage) (json.RawMessage, error) {
	if sameWorkspacePreferenceValue(base, next) {
		return current, nil
	}
	if sameWorkspacePreferenceValue(base, current) || sameWorkspacePreferenceValue(next, current) {
		return next, nil
	}
	var before, requested, disk map[string]json.RawMessage
	if json.Unmarshal(base, &before) == nil && before != nil && json.Unmarshal(next, &requested) == nil && requested != nil && json.Unmarshal(current, &disk) == nil && disk != nil {
		keys := map[string]bool{}
		for key := range before {
			keys[key] = true
		}
		for key := range requested {
			keys[key] = true
		}
		for key := range disk {
			keys[key] = true
		}
		ordered := make([]string, 0, len(keys))
		for key := range keys {
			ordered = append(ordered, key)
		}
		sort.Strings(ordered)
		merged := map[string]json.RawMessage{}
		for _, key := range ordered {
			value, err := mergeWorkspacePreferenceValue(field+"."+key, before[key], requested[key], disk[key])
			if err != nil {
				return nil, err
			}
			if len(value) > 0 {
				merged[key] = value
			}
		}
		return json.Marshal(merged)
	}
	return nil, fmt.Errorf("%w: view preference %s was also changed in another window", ErrConflict, field)
}
