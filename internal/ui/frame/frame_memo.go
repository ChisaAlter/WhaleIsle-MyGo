package frame

import (
	"math"
	"runtime"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/shortcuts"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

const (
	frameFNVOffset64 uint64 = 14695981039346656037
	frameFNVPrime64  uint64 = 1099511628211
)

func frameHashU64(h, v uint64) uint64 {
	for i := 0; i < 8; i++ {
		h ^= uint64(byte(v))
		h *= frameFNVPrime64
		v >>= 8
	}
	return h
}

func frameHashStr(h uint64, s string) uint64 {
	h = frameHashU64(h, uint64(len(s)))
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= frameFNVPrime64
	}
	return h
}

func frameHashBool(h uint64, b bool) uint64 {
	if b {
		h ^= 1
	}
	return h * frameFNVPrime64
}

func frameHashF32(h uint64, v float32) uint64 {
	return frameHashU64(h, uint64(math.Float32bits(v)))
}

var (
	defaultMenuShortcutsOnce sync.Once
	defaultMenuShortcuts     map[string]string

	customMenuShortcutsMu  sync.RWMutex
	customMenuShortcutsSrc map[string]string
	customMenuShortcutsVal map[string]string
)

func buildMenuShortcutLabels(bindings map[string]string) map[string]string {
	resolved := shortcuts.Resolve(bindings)
	labels := make(map[string]string, len(resolved))
	for id, chord := range resolved {
		label := chord.String()
		switch runtime.GOOS {
		case "windows":
			label = strings.Replace(label, "Mod", "Ctrl", 1)
		case "darwin":
			label = strings.Replace(label, "Mod", "Cmd", 1)
		}
		labels[id] = label
	}
	return labels
}

func sameStringMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

// resolvedMenuShortcutLabels caches platform-formatted shortcut labels so
// navigationView avoids calling shortcuts.Resolve / shortcuts.Defaults on every
// frame. The returned map is read-only.
func resolvedMenuShortcutLabels(bindings map[string]string) map[string]string {
	if len(bindings) == 0 {
		defaultMenuShortcutsOnce.Do(func() {
			defaultMenuShortcuts = buildMenuShortcutLabels(nil)
		})
		return defaultMenuShortcuts
	}
	customMenuShortcutsMu.RLock()
	if customMenuShortcutsVal != nil && sameStringMap(customMenuShortcutsSrc, bindings) {
		hit := customMenuShortcutsVal
		customMenuShortcutsMu.RUnlock()
		return hit
	}
	customMenuShortcutsMu.RUnlock()

	labels := buildMenuShortcutLabels(bindings)
	owned := make(map[string]string, len(bindings))
	for k, v := range bindings {
		owned[k] = v
	}
	customMenuShortcutsMu.Lock()
	customMenuShortcutsSrc = owned
	customMenuShortcutsVal = labels
	customMenuShortcutsMu.Unlock()
	return labels
}

type heroTrackState struct {
	initialized bool
	lastColW    float32
	lastOuterW  float32
	lastBottom  bool
	animUntil   time.Time
}

func heroHeadlineMemoVersion(m Model, outerW float32) uint64 {
	h := frameFNVOffset64
	h = frameHashStr(h, m.ThemeID)
	h = frameHashBool(h, m.SystemDark)
	h = frameHashStr(h, string(i18n.Locale()))
	h = frameHashStr(h, m.HostPlaceholder)
	h = frameHashU64(h, theme.Revision())
	h = frameHashU64(h, design.Revision())
	h = frameHashBool(h, chrome.BackdropActive())
	h = frameHashF32(h, outerW)
	return h
}

func settlingSkeletonMemoVersion(m Model, outerW float32) uint64 {
	h := frameFNVOffset64
	h = frameHashStr(h, m.ThemeID)
	h = frameHashBool(h, m.SystemDark)
	h = frameHashStr(h, string(i18n.Locale()))
	h = frameHashU64(h, theme.Revision())
	h = frameHashU64(h, design.Revision())
	h = frameHashBool(h, chrome.BackdropActive())
	h = frameHashF32(h, outerW)
	return h
}

func frameHashBytes(h uint64, b []byte) uint64 {
	h = frameHashU64(h, uint64(len(b)))
	for i := 0; i < len(b); i++ {
		h ^= uint64(b[i])
		h *= frameFNVPrime64
	}
	return h
}

func hashInspectorRecord(h uint64, r *session.InspectorRecord) uint64 {
	h = frameHashStr(h, r.Key)
	h = frameHashStr(h, r.Parent)
	h = frameHashStr(h, r.Type)
	h = frameHashStr(h, r.Identity)
	h = frameHashStr(h, r.Location)
	h = frameHashU64(h, math.Float64bits(r.Seq))
	h = frameHashU64(h, math.Float64bits(r.Time))
	h = frameHashBytes(h, r.Raw)
	h = frameHashStr(h, r.Disclosure)
	h = frameHashStr(h, r.CollapsedSummary)
	h = frameHashStr(h, r.ChatID)
	h = frameHashStr(h, r.ChatPickID)
	h = frameHashStr(h, r.ChatTarget.ID)
	h = frameHashU64(h, math.Float64bits(r.ChatTarget.Seq))
	h = frameHashStr(h, r.ChatTarget.CallID)
	return h
}

type inspectorSourceCacheKey struct {
	binding     string
	sessionID   string
	recordCount int
	recordHash  uint64
	nodeCount   int
	nodeHash    uint64
	groupCount  int
	groupHash   uint64
}

func makeInspectorSourceCacheKey(source session.InspectorSnapshot, nav *chat.InspectorNavigation) inspectorSourceCacheKey {
	rh := frameFNVOffset64
	for i := range source.Records {
		rh = hashInspectorRecord(rh, &source.Records[i])
	}
	k := inspectorSourceCacheKey{
		binding:     source.Binding,
		sessionID:   source.SessionID,
		recordCount: len(source.Records),
		recordHash:  rh,
	}
	if nav != nil {
		nh := frameFNVOffset64
		for i := range nav.Nodes {
			n := &nav.Nodes[i]
			nh = frameHashStr(nh, n.ID)
			nh = frameHashU64(nh, math.Float64bits(n.Seq))
			nh = frameHashStr(nh, string(n.Kind))
			nh = frameHashStr(nh, n.Role)
			nh = frameHashStr(nh, n.EventType)
			nh = frameHashStr(nh, n.Body)
			nh = frameHashBytes(nh, n.Raw)
			nh = frameHashBool(nh, n.Streaming)
			nh = frameHashU64(nh, math.Float64bits(n.TurnSeq))
			nh = frameHashStr(nh, n.SourceMessageID)
		}
		var gh uint64
		for gk, members := range nav.Groups {
			eh := frameHashStr(frameFNVOffset64, gk)
			eh = frameHashU64(eh, uint64(len(members)))
			for _, member := range members {
				eh = frameHashStr(eh, member)
			}
			gh ^= eh
		}
		k.nodeCount = len(nav.Nodes)
		k.nodeHash = nh
		k.groupCount = len(nav.Groups)
		k.groupHash = gh
	}
	return k
}

type inspectorTableCacheKey struct {
	rowCount     int
	rowHash      uint64
	filter       string
	collapsedRev uint64
}

func makeInspectorTableCacheKey(rows []session.InspectorRecord, filter string, collapsedRev uint64) inspectorTableCacheKey {
	rh := frameFNVOffset64
	for i := range rows {
		rh = hashInspectorRecord(rh, &rows[i])
	}
	return inspectorTableCacheKey{
		rowCount:     len(rows),
		rowHash:      rh,
		filter:       filter,
		collapsedRev: collapsedRev,
	}
}

type inspectorRenderCache struct {
	sourceValid bool
	sourceKey   inspectorSourceCacheKey
	chatRows    []session.InspectorRecord
	objects     map[string]inspectorObject

	tableValid bool
	tableKey   inspectorTableCacheKey
	depths     map[string]int
	filtered   []session.InspectorRecord
	branches   map[string]bool
	shown      []session.InspectorRecord
}
