package terminal

import (
	"fmt"

	myterm "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/hostterminal"
	"whaleisle/internal/ui/theme"
)

func HostTitle() string { return terminalLabel("Host 终端", "Host terminal") }

// HostState owns only the native renderer. Host process and hold ownership live
// in the Store, independently of this view or the local PTY drawer.
// All HostState methods run on the UI thread.
type HostState struct {
	occurrences     map[string]*HostState
	terminal        *myterm.Terminal
	attachment      *hostterminal.Attachment
	closeTarget     string
	confirmClose    bool
	manageOpen      bool
	renderError     string
	selectionNotice string
}

func (h *HostState) drop() {
	if h.terminal != nil {
		_ = h.terminal.Close()
	}
	h.terminal = nil
	h.attachment = nil
	h.renderError = ""
	h.selectionNotice = ""
}
func (h *HostState) Observe(store *hostterminal.Store, session string, visible bool) {
	if store == nil {
		return
	}
	store.Show(session, visible)
	if !visible {
		h.hideOccurrences(store, nil)
		h.drop()
		h.confirmClose = false
		h.manageOpen = false
		h.closeTarget = ""
	}
}

// HostView offers list/create/connect/close actions. Only StartFresh, carried
// from an explicit new-tab action, creates on mount; ordinary recovery does not.
// Options.Conn prevents a local PTY fallback.
func HostView(c *ui.Context, h *HostState, store *hostterminal.Store, session, themeID string, dark bool, actionOptions ...HostActions) {
	var actions HostActions
	if len(actionOptions) > 0 {
		actions = actionOptions[0]
	}
	if h == nil || store == nil {
		return
	}
	h.Observe(store, session, true)
	v := store.Snapshot(session)
	if actions.StartFresh != nil && *actions.StartFresh && store.StartFresh(v.Owner, session) {
		*actions.StartFresh = false
		v = store.Snapshot(session)
	}
	colors := theme.Resolve(themeID).Active(dark)
	target := fmt.Sprintf("%d:%s:%s", v.Owner, session, v.ActiveID)
	if target != h.closeTarget {
		h.closeTarget = target
		h.confirmClose = false
	}
	if h.attachment != v.Attachment {
		h.drop()
		h.attachment = v.Attachment
	}
	if v.Render != nil && v.Attachment != nil {
		r := v.Render
		if r.Snapshot && h.terminal == nil {
			if err := prepareNativeRuntime(); err != nil {
				h.renderError = err.Error()
				store.RenderFailed(v.Attachment, err)
			} else {
				scrollback := -1
				if v.Environment.Scrollback > 0 {
					scrollback = min(20<<20, v.Environment.Scrollback*max(80, v.Environment.MaxCols)*4)
				}
				themed := terminalTheme(colors, dark)
				copyOwner, copyID, copyAttachment := v.Owner, v.ActiveID, v.Attachment
				opts := myterm.Options{Conn: v.Attachment, ReadOnly: true, CanCopy: func() bool { return store.CanCopySelection(copyOwner, session, copyID, copyAttachment) }, Theme: themed, DarkTheme: themed, Scrollback: scrollback, Font: terminalFont(theme.RenderPreferences().Terminal)}
				opts.LinkAt = TargetAt
				opts.OnLinkActivate = func(target string) bool {
					if !store.CanCopySelection(copyOwner, session, copyID, copyAttachment) {
						h.selectionNotice = "Terminal target is no longer available"
						return true
					}
					if actions.OpenTarget == nil {
						h.selectionNotice = "Terminal target opener is unavailable"
						return true
					}
					if err := actions.OpenTarget(target, store.Snapshot(session).Info.Cwd); err != nil {
						h.selectionNotice = err.Error()
					} else {
						h.selectionNotice = ""
					}
					return true
				}
				var err error
				h.terminal, err = myterm.New(opts)
				if err != nil {
					h.renderError = err.Error()
					store.RenderFailed(v.Attachment, err)
				}
			}
		}
		if h.terminal != nil {
			h.terminal.SetReadOnly(!v.Writable)
			if r.Snapshot {
				h.terminal.Resize(r.Cols, r.Rows)
			}
			h.terminal.Feed([]byte(r.Data))
			store.Acknowledge(v.Attachment, r.Revision)
		}
		v = store.Snapshot(session)
	}
	hostSurface(c, h, store, session, colors, v, actions)
}
