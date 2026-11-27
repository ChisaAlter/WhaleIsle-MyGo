package frame

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"maps"
	"strings"
	"sync"
	"whaleisle/internal/shortcuts"
)

type resolvedPageCommand struct {
	id         string
	identity   string
	region     string
	mods       ui.Modifiers
	code       string
	secondCode string
}

var pageCommandsMemo struct {
	mu      sync.Mutex
	valid   bool
	input   map[string]string
	entries []resolvedPageCommand
}

func resolvedPageCommands(overrides map[string]string) []resolvedPageCommand {
	pageCommandsMemo.mu.Lock()
	if pageCommandsMemo.valid && maps.Equal(pageCommandsMemo.input, overrides) {
		entries := pageCommandsMemo.entries
		pageCommandsMemo.mu.Unlock()
		return entries
	}
	pageCommandsMemo.mu.Unlock()

	bindings := shortcuts.Resolve(overrides)
	accepted, _ := json.Marshal(overrides)
	acceptedStr := string(accepted)
	cmds := shortcuts.Commands()
	entries := make([]resolvedPageCommand, 0, len(cmds))
	for _, command := range cmds {
		chord, ok := bindings[command.ID]
		if !ok {
			continue
		}
		mods, code, secondCode, valid := physicalBindingSpec(chord)
		if !valid {
			continue
		}
		region := shortcuts.NativeCommandScope(command.ID)
		entries = append(entries, resolvedPageCommand{
			id:         command.ID,
			identity:   command.ID + acceptedStr,
			region:     region,
			mods:       mods,
			code:       code,
			secondCode: secondCode,
		})
	}

	pageCommandsMemo.mu.Lock()
	pageCommandsMemo.valid = true
	pageCommandsMemo.input = maps.Clone(overrides)
	pageCommandsMemo.entries = entries
	pageCommandsMemo.mu.Unlock()
	return entries
}

func physicalBindingSpec(chord shortcuts.Chord) (ui.Modifiers, string, string, bool) {
	b := chord.PhysicalBinding()
	if b == nil {
		return 0, "", "", false
	}
	mods := ui.Modifiers(0)
	for _, m := range b.Modifiers {
		switch m {
		case "primary":
			mods |= ui.Cmd
		case "control":
			mods |= ui.Ctrl
		case "alt":
			mods |= ui.Alt
		case "shift":
			mods |= ui.Shift
		case "meta":
			mods |= ui.Super
		}
	}
	return mods, b.Code, b.SecondCode, true
}

// registerPageCommands reuses existing actions. PageShortcut retains text/IME,
// terminal, guest, popup and modal ownership for both defaults and custom keys.
func registerPageCommands(c *ui.Context, m Model) {
	registerPageCommandsPtr(c, &m)
}

func registerPageCommandsPtr(c *ui.Context, m *Model) {
	mainChildActive := m.Sizes != nil && m.Sizes.Workspace != nil && m.Sizes.Workspace.currentMainChildPtr(m) != nil
	for _, entry := range resolvedPageCommands(m.PageShortcutBindings) {
		if !pageCommandEnabled(m, mainChildActive, entry.id) {
			continue
		}
		if c.PhysicalShortcut(entry.mods, entry.code, entry.secondCode, entry.identity, entry.region) {
			dispatchPageCommand(c, m, entry.id)
		}
	}
}

func pageCommandEnabled(m *Model, mainChildActive bool, id string) bool {
	if mainChildActive {
		switch id {
		case "session.rename", "session.fork", "session.archive":
			return false
		}
	}
	switch id {
	case "sidebar.toggle":
		return m.Sizes != nil
	case "session.new":
		return m.OnCreateSession != nil || m.Store != nil
	case "workspace.open", "workspace.add":
		return m.OnBrowseWorkspace != nil
	case "settings.open":
		return m.OnOpenSettings != nil
	case "tools.toggle", "tools.toggle.alternate", "sidebar.right.toggle":
		return m.OnToggleTools != nil
	case "terminal.toggle":
		return m.OnToggleTerminal != nil
	case "page.close", "page.refresh", "pane.split", "pane.splitDown", "pane.fullscreen.toggle", "session.search":
		return true
	case "workspace.openLocal":
		return m.OnOpenWorkspaceLocal != nil
	case "browser.open":
		return m.OnOpenBrowserPreview != nil
	case "terminal.host":
		return m.OnOpenHostTerminal != nil
	case "terminal.new":
		return m.OnNewHostTerminal != nil
	case "market.open":
		return m.OnOpenMarket != nil
	case "schedule.open":
		return m.OnOpenSchedule != nil
	case "assistant.open":
		return m.OnOpenAssistant != nil
	case "launcher.open":
		return m.OnOpenDesktopSettings != nil
	case "workspace.files":
		return m.OnOpenWorkspaceFiles != nil
	case "sessions.refresh", "session.rename", "session.fork", "session.archive", "session.previous", "session.next":
		return m.Store != nil
	default:
		return false
	}
}

func dispatchPageCommand(c *ui.Context, m *Model, id string) {
	switch id {
	case "sidebar.toggle":
		if m.Sizes != nil {
			m.Sizes.ToggleSidebar()
		}
	case "session.new":
		if m.OnCreateSession != nil {
			m.OnCreateSession()
		} else if m.Store != nil {
			m.Store.CreateSession(context.Background())
		}
	case "workspace.open", "workspace.add":
		if m.OnBrowseWorkspace != nil {
			m.OnBrowseWorkspace()
		}
	case "settings.open":
		if m.OnOpenSettings != nil {
			m.OnOpenSettings()
		}
	case "tools.toggle", "tools.toggle.alternate", "sidebar.right.toggle":
		if m.OnToggleTools != nil {
			m.OnToggleTools()
		}
	case "terminal.toggle":
		if m.OnToggleTerminal != nil {
			m.OnToggleTerminal()
		}
	case "page.close":
		closeFocusedWorkspacePage(c, *m)
	case "page.refresh":
		refreshFocusedWorkspacePage(c, *m)
	case "pane.split":
		splitFocusedWorkspacePage(c, *m)
	case "pane.splitDown":
		SplitWorkspacePane(*m, true)
	case "pane.fullscreen.toggle":
		maximizeFocusedWorkspacePage(c, *m)
	case "workspace.openLocal":
		if m.OnOpenWorkspaceLocal != nil {
			m.OnOpenWorkspaceLocal()
		}
	case "browser.open":
		if m.OnOpenBrowserPreview != nil {
			m.OnOpenBrowserPreview()
		}
	case "terminal.host":
		if m.OnOpenHostTerminal != nil {
			m.OnOpenHostTerminal()
		}
	case "terminal.new":
		if m.OnNewHostTerminal != nil {
			m.OnNewHostTerminal()
		}
	case "market.open":
		if m.OnOpenMarket != nil {
			m.OnOpenMarket()
		}
	case "schedule.open":
		if m.OnOpenSchedule != nil {
			m.OnOpenSchedule()
		}
	case "assistant.open":
		if m.OnOpenAssistant != nil {
			m.OnOpenAssistant()
		}
	case "launcher.open":
		if m.OnOpenDesktopSettings != nil {
			m.OnOpenDesktopSettings()
		}
	case "workspace.files":
		if m.OnOpenWorkspaceFiles != nil {
			m.OnOpenWorkspaceFiles()
		}
	case "session.search":
		if m.Sizes != nil && m.Sizes.sidebarPreference() <= 0 {
			m.Sizes.ToggleSidebar()
		}
		if m.SessionSearchFocus != nil {
			*m.SessionSearchFocus = true
		}
		c.Invalidate()
	case "sessions.refresh":
		if m.Store != nil {
			m.Store.RefreshList(context.Background())
		}
	case "session.rename":
		if m.Store != nil {
			snap := m.Store.Published().Value
			for _, item := range snap.Items {
				if item.SessionID == snap.SelectedID {
					beginConversationRenamePtr(m, item.SessionID, item.Title)
					return
				}
			}
		}
	case "session.fork":
		if m.Store != nil {
			id := m.Store.SelectedSessionID()
			if item, ok := conversationActionItemPtr(m, id); ok && !routeProjectConversationActionPtr(m, item, "fork") {
				m.Store.ForkSession(context.Background(), id)
			}
		}
	case "session.archive":
		if m.Store != nil {
			id := m.Store.SelectedSessionID()
			archiveConversationPtr(m, id, false)
		}
	case "session.previous":
		selectNeighborSession(m, -1)
	case "session.next":
		selectNeighborSession(m, 1)
	}
}

func NativeCommandPressed(c *ui.Context, chord shortcuts.Chord, identity ...string) bool {
	mods, code, secondCode, ok := physicalBindingSpec(chord)
	if !ok {
		return false
	}
	return c.PhysicalShortcut(mods, code, secondCode, identity...)
}

func NativeCommandChord(chord shortcuts.Chord) (ui.Modifiers, ui.Key) {
	mods := ui.Cmd
	if chord.Shift {
		mods |= ui.Shift
	}
	if chord.Alt {
		mods |= ui.Alt
	}
	letters := []ui.Key{ui.KeyA, ui.KeyB, ui.KeyC, ui.KeyD, ui.KeyE, ui.KeyF, ui.KeyG, ui.KeyH, ui.KeyI, ui.KeyJ, ui.KeyK, ui.KeyL, ui.KeyM, ui.KeyN, ui.KeyO, ui.KeyP, ui.KeyQ, ui.KeyR, ui.KeyS, ui.KeyT, ui.KeyU, ui.KeyV, ui.KeyW, ui.KeyX, ui.KeyY, ui.KeyZ}
	digits := []ui.Key{ui.Key0, ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9}
	if len(chord.Key) == 1 {
		if i := strings.Index("ABCDEFGHIJKLMNOPQRSTUVWXYZ", chord.Key); i >= 0 {
			return mods, letters[i]
		}
		if i := strings.Index("0123456789", chord.Key); i >= 0 {
			return mods, digits[i]
		}
	}
	functionKeys := []ui.Key{ui.KeyF1, ui.KeyF2, ui.KeyF3, ui.KeyF4, ui.KeyF5, ui.KeyF6, ui.KeyF7, ui.KeyF8, ui.KeyF9, ui.KeyF10, ui.KeyF11, ui.KeyF12}
	for index, key := range functionKeys {
		if chord.Key == fmt.Sprintf("F%d", index+1) {
			return mods, key
		}
	}
	keys := map[string]ui.Key{"ENTER": ui.KeyEnter, ",": ui.KeyComma, "\\": ui.KeyBackslash, "`": ui.KeyBackquote, ".": ui.KeyPeriod, ";": ui.KeySemicolon, "'": ui.KeyQuote, "[": ui.KeyBracketLeft, "]": ui.KeyBracketRight, "-": ui.KeyMinus, "=": ui.KeyEqual, "/": ui.KeySlash}
	return mods, keys[chord.Key]
}
