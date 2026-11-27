package chat

import (
	"context"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

const (
	fnvOffset64 uint64 = 14695981039346656037
	fnvPrime64  uint64 = 1099511628211
)

func hashU64(h, v uint64) uint64 {
	for i := 0; i < 8; i++ {
		h ^= uint64(byte(v))
		h *= fnvPrime64
		v >>= 8
	}
	return h
}

func hashStr(h uint64, s string) uint64 {
	h = hashU64(h, uint64(len(s)))
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= fnvPrime64
	}
	return h
}

func hashBool(h uint64, b bool) uint64 {
	if b {
		h ^= 1
	}
	return h * fnvPrime64
}

func hashF32(h uint64, v float32) uint64 {
	return hashU64(h, uint64(math.Float32bits(v)))
}

func hashColor(h uint64, c ui.Color) uint64 {
	return hashU64(h, uint64(c.R)<<24|uint64(c.G)<<16|uint64(c.B)<<8|uint64(c.A))
}

type chatContentMemoKey struct {
	id  string
	seq float64
}

// markdownBodyHasAsyncOrInteractiveBlocks reports whether a markdown body may
// contain async math rendering or interactive code-block toolbars (copy timer,
// line-wrap toggle) that cannot be safely frozen behind a static ui.Memo.
func markdownBodyHasAsyncOrInteractiveBlocks(body string) bool {
	if strings.Contains(body, "```") || strings.Contains(body, "~~~") ||
		strings.HasPrefix(body, "    ") || strings.HasPrefix(body, "\t") ||
		strings.Contains(body, "\n\n    ") || strings.Contains(body, "\n\n\t") {
		return true
	}
	if strings.Contains(body, "$$") || strings.Contains(body, `\(`) || strings.Contains(body, `\[`) {
		return true
	}
	if strings.Count(body, "$") >= 2 {
		return true
	}
	return false
}

// chatMarkdownMemoVersion returns a stable ui.Memo version for settled
// assistant markdown blocks that do not contain dynamic media/inspector/openui,
// math formulas, or code blocks. Keeping ContextMenu outside this memo boundary
// lets mygo reuse the markdown layout and text nodes without being marked live.
func chatMarkdownMemoVersion(ctx *ui.Context, m Model, msg session.Message, body string) (uint64, bool) {
	if msg.Streaming || m.Inspector != nil || body == "" {
		return 0, false
	}
	// Images and math formulas render asynchronously; OpenUI fences and code
	// blocks host interactive controls/timers.
	if strings.Contains(body, "![") || strings.Contains(body, "openui") || strings.Contains(body, "whale-ui") || strings.Contains(body, "dsh-ui") ||
		markdownBodyHasAsyncOrInteractiveBlocks(body) {
		return 0, false
	}
	prefs := theme.CurrentPreferences()
	uiPrefs := ctx.Preferences()
	tokens := design.For(ctx)
	backdropActive := chrome.BackdropActive()
	backdropPrefs := chrome.BackdropPreferences()
	h := fnvOffset64
	h = hashStr(h, msg.ID)
	h = hashStr(h, body)
	h = hashF32(h, m.ContentWidth)
	h = hashU64(h, theme.Revision())
	h = hashU64(h, design.Revision())
	h = hashBool(h, tokens.Glass)
	h = hashBool(h, backdropActive)
	h = hashBool(h, backdropPrefs.Transparent)
	h = hashF32(h, float32(backdropPrefs.Glass))
	h = hashF32(h, prefs.Chat.Size)
	h = hashStr(h, prefs.Chat.Family)
	h = hashF32(h, prefs.Code.Size)
	h = hashStr(h, prefs.Code.Family)
	h = hashF32(h, uiPrefs.TextScale)
	h = hashBool(h, uiPrefs.ReduceMotion)
	h = hashColor(h, tokens.Foreground)
	h = hashColor(h, tokens.Background)
	h = hashColor(h, tokens.Primary)
	h = hashColor(h, tokens.Code)
	h = hashColor(h, tokens.Card)
	h = hashColor(h, tokens.Border)
	h = hashColor(h, tokens.Muted)
	h = hashColor(h, tokens.MutedForeground)
	h = hashStr(h, m.ThemeID)
	h = hashBool(h, m.SystemDark)
	h = hashStr(h, string(i18n.Locale()))
	h = hashBool(h, m.OnOpenCitationURL != nil)
	h = hashBool(h, m.OnOpenDeliverable != nil)
	h = hashBool(h, projectedFileMentionResolver(m.Snapshot, msg) != nil)
	h = hashU64(h, projectedFileMentionRevision(m.Snapshot, msg))
	return h, true
}

type scheduleCardsCache struct {
	ready    bool
	projKey  session.ProjectionKey
	selected string
	binding  string
	msgCount int
	firstID  string
	lastID   string
	cards    map[float64][]schedstore.Entry
	cardIDs  []string
	hasStore bool
}

// prepareScheduleContextCached reuses ScheduleCards in O(1) while the store's
// DurableHistoryProjectionKey (or HistoryProjectionKey fallback) is unchanged,
// avoiding per-frame O(N) scans and map clones on streaming and long-history
// viewports.
func prepareScheduleContextCached(root ui.Element, m Model, memo *session.HistoryDerivedMemo) map[float64][]schedstore.Entry {
	projKey := m.Snapshot.DurableHistoryProjectionKey()
	if !projKey.Valid() {
		projKey = m.Snapshot.HistoryProjectionKey()
	}
	if !projKey.Valid() {
		return prepareScheduleContext(m, memo)
	}
	cache := ui.Local(root, "schedule-cards-cache", func() scheduleCardsCache { return scheduleCardsCache{} })
	n := len(m.Snapshot.Messages)
	var firstID, lastID string
	if n > 0 {
		firstID = m.Snapshot.Messages[0].ID
		lastID = m.Snapshot.Messages[n-1].ID
	}
	if cache.ready && cache.projKey == projKey && cache.selected == m.Snapshot.SelectedID &&
		cache.binding == m.Snapshot.PlanHistoryBinding &&
		cache.msgCount == n && cache.firstID == firstID && cache.lastID == lastID {
		if m.ScheduleStore != nil {
			m.ScheduleStore.EnsureContext(context.Background(), m.Snapshot.SelectedID, cache.cardIDs)
		}
		return cache.cards
	}
	cards := memo.ScheduleCards(m.Snapshot)
	var ids []string
	for _, rows := range cards {
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
	}
	if m.ScheduleStore != nil {
		m.ScheduleStore.EnsureContext(context.Background(), m.Snapshot.SelectedID, ids)
	}
	cache.ready = true
	cache.projKey = projKey
	cache.selected = m.Snapshot.SelectedID
	cache.binding = m.Snapshot.PlanHistoryBinding
	cache.msgCount = n
	cache.firstID = firstID
	cache.lastID = lastID
	cache.cards = cards
	cache.cardIDs = ids
	return cards
}
