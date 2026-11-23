// Package dshbot persists native dshbot roster snapshots under userData/assistant/dshbot/.
package dshbot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/state"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/store/internal/storefs"
)

const rosterFile = "roster.json"

// Row is one cached bot or room list entry.
type Row struct {
	MemberBotIDs []string `json:"memberBotIds,omitempty"`
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Title        string   `json:"title,omitempty"`
	SessionID    string   `json:"sessionId"`
	Hidden       bool     `json:"hidden,omitempty"`
	Pinned       bool     `json:"pinned,omitempty"`
	MemberCount  int      `json:"memberCount,omitempty"`
	DisplayLabel string   `json:"displayLabel"`
	Detail       string   `json:"detail,omitempty"` // activity preview or group/runtime summary
}

// Snapshot is the on-disk roster cache.
type Snapshot struct {
	Origin    string `json:"origin,omitempty"`
	Revision  int    `json:"revision,omitempty"`
	FetchedAt string `json:"fetchedAt"`
	Rows      []Row  `json:"rows"`
}

// Store holds the last describe-derived roster. The mutable document stays
// under mu; readers get a published deep-copied version. Disk writes run
// outside mu, serialized by persistMu so the last write wins.
type Store struct {
	mu        sync.Mutex
	persistMu sync.Mutex
	root      string
	snap      Snapshot
	pub       state.Published[Snapshot]
}

// Dir returns userData/assistant/dshbot.
func Dir(userData string) string {
	return filepath.Join(assiststore.Dir(userData), "dshbot")
}

// Open loads or creates the roster cache directory.
func Open(userData string) (*Store, error) {
	root := Dir(userData)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	s := &Store{root: root}
	if _, err := storefs.ReadJSON("dshbot", filepath.Join(root, rosterFile), &s.snap, false); err != nil {
		return nil, err
	}
	s.pub.Publish(cloneSnapshot(s.snap))
	return s, nil
}

func cloneSnapshot(v Snapshot) Snapshot {
	out := v
	out.Rows = append([]Row(nil), v.Rows...)
	for i := range out.Rows {
		out.Rows[i].MemberBotIDs = append([]string(nil), out.Rows[i].MemberBotIDs...)
	}
	return out
}

// Snapshot returns the published roster version; unchanged reads share it.
func (s *Store) Snapshot() *state.Version[Snapshot] {
	if v := s.pub.Load(); v != nil {
		return v
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pub.Load() == nil {
		s.pub.Publish(cloneSnapshot(s.snap))
	}
	return s.pub.Load()
}

// SetFromDescribe replaces the cache from a describe result.
func (s *Store) SetFromDescribe(res pluginrpc.DshbotDescribeResult, includeHidden bool) {
	s.SetFromDescribeForHost(res, includeHidden, "")
}

// SetFromDescribeForHost binds cached identities to the Host that supplied them.
func (s *Store) SetFromDescribeForHost(res pluginrpc.DshbotDescribeResult, includeHidden bool, origin string) {
	rows := pluginrpc.RosterRowsFromDescribe(res, includeHidden)
	converted := make([]Row, 0, len(rows))
	for _, r := range rows {
		converted = append(converted, Row{
			ID:           r.ID,
			Kind:         r.Kind,
			Name:         r.Name,
			Title:        r.Title,
			SessionID:    r.SessionID,
			Hidden:       r.Hidden,
			Pinned:       r.Pinned,
			MemberCount:  r.MemberCount,
			MemberBotIDs: append([]string(nil), r.MemberBotIDs...),
			DisplayLabel: r.DisplayLabel,
		})
	}
	s.mu.Lock()
	s.snap = Snapshot{
		Origin:    strings.TrimSpace(origin),
		Revision:  res.View.Revision,
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
		Rows:      converted,
	}
	s.pub.Publish(cloneSnapshot(s.snap))
	s.mu.Unlock()
	s.persist()
}

// MergeActivity overlays bot/activity previews onto matching bot rows.
func (s *Store) MergeActivity(act pluginrpc.DshbotActivityResult) {
	s.mu.Lock()
	byID := map[string]string{}
	for _, e := range act.Entries {
		id := strings.TrimSpace(e.BotID)
		if id == "" {
			continue
		}
		preview := strings.TrimSpace(e.Preview)
		if preview == "" && e.Attention {
			preview = "attention"
		}
		if err := strings.TrimSpace(e.Error); err != "" {
			if preview != "" {
				preview += " · "
			}
			preview += err
		}
		if preview == "" {
			continue
		}
		if len(preview) > 80 {
			preview = preview[:80] + "…"
		}
		byID[id] = preview
	}
	if len(byID) == 0 {
		return
	}
	for i := range s.snap.Rows {
		if d, ok := byID[s.snap.Rows[i].ID]; ok {
			s.snap.Rows[i].Detail = d
		}
	}
	s.pub.Publish(cloneSnapshot(s.snap))
	s.mu.Unlock()
	s.persist()
}

// ApplyGroupRuntime updates a room row from group/runtime (member count + status).
func (s *Store) ApplyGroupRuntime(rt pluginrpc.DshbotGroupRuntimeResult) {
	roomID := strings.TrimSpace(rt.RoomID)
	if roomID == "" {
		return
	}
	s.mu.Lock()
	for i := range s.snap.Rows {
		row := &s.snap.Rows[i]
		if row.Kind != "room" || row.ID != roomID {
			continue
		}
		row.MemberCount = len(rt.Members)
		if sid := strings.TrimSpace(rt.SessionID); sid != "" {
			row.SessionID = sid
		}
		statuses := make([]string, 0, len(rt.Members))
		for _, m := range rt.Members {
			st := strings.TrimSpace(m.Status)
			if st == "" {
				st = "?"
			}
			statuses = append(statuses, st)
		}
		if len(statuses) > 0 {
			row.Detail = "runtime: " + strings.Join(statuses, ",")
			if len(row.Detail) > 100 {
				row.Detail = row.Detail[:100] + "…"
			}
		}
		s.pub.Publish(cloneSnapshot(s.snap))
		s.mu.Unlock()
		s.persist()
		return
	}
	s.mu.Unlock()
}

// persist re-reads the newest snapshot under a brief lock so a slower earlier
// write cannot overwrite a later one.
func (s *Store) persist() {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.mu.Lock()
	snap := cloneSnapshot(s.snap)
	s.mu.Unlock()
	if s.root == "" {
		return
	}
	raw, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return
	}
	_ = storefs.WriteFileAtomic(filepath.Join(s.root, rosterFile), raw, 0o644)
}

// RowSessionID returns a trimmed session id for a roster id, if present.
func (s *Store) RowSessionID(id string) string {
	id = strings.TrimSpace(id)
	for _, row := range s.Snapshot().Value.Rows {
		if row.ID == id {
			return strings.TrimSpace(row.SessionID)
		}
	}
	return ""
}
