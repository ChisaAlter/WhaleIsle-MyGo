// Package shortcuts owns native application and page bindings, not system hotkeys.
package shortcuts

import (
	"errors"
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strings"
	"sync"
)

// IDs are native binding identifiers. The tools alternate intentionally targets
// the same action as tools.toggle; these are seven bindings and six actions.
type Command struct{ ID, ZH, EN, Default string }

var commands = []Command{
	{"sidebar.toggle", "切换左侧栏", "Toggle left sidebar", "Mod+B"},
	{"session.new", "新建对话", "New conversation", "Mod+N"},
	{"workspace.open", "浏览工作区", "Browse workspace", "Mod+O"},
	{"settings.open", "打开设置", "Open settings", "Mod+,"},
	{"tools.toggle", "切换右侧面板", "Toggle tools pane", "Mod+\\"},
	{"tools.toggle.alternate", "切换右侧面板（备用）", "Toggle tools pane (alternate)", ""},
	{"sidebar.right.toggle", "展开右侧栏", "Toggle right sidebar", "Mod+Alt+B"},
	{"page.close", "关闭当前页", "Close current page", "Mod+W"},
	{"page.refresh", "刷新当前页", "Refresh current page", "Mod+R"},
	{"terminal.toggle", "切换终端", "Toggle terminal", "Mod+`"},
	{"session.search", "搜索对话", "Search conversations", "Mod+Shift+K"},
	{"session.rename", "重命名当前对话", "Rename conversation", "Mod+Shift+R"},
	{"session.fork", "分支当前对话", "Fork conversation", "Mod+Shift+J"},
	{"session.archive", "归档当前对话", "Archive conversation", "Mod+Shift+E"},
	{"session.previous", "上一个对话", "Previous conversation", "Mod+Alt+["},
	{"session.next", "下一个对话", "Next conversation", "Mod+Alt+]"},
	{"workspace.add", "添加工作区", "Add workspace", "Mod+Shift+O"},
	{"workspace.openLocal", "在本地打开", "Open locally", "Mod+Alt+O"},
	{"workspace.files", "打开工作区文件", "Open workspace files", "Mod+P"},
	{"browser.open", "打开浏览器面板", "Open browser pane", "Mod+T"},
	{"terminal.host", "打开 Host 终端面板", "Open Host terminal pane", "Mod+Alt+T"},
	{"terminal.new", "新建 Host 终端", "New Host terminal", ""},
	{"sessions.refresh", "刷新对话列表", "Refresh conversations", "Mod+Alt+R"},
	{"market.open", "打开市场", "Open marketplace", "Mod+Shift+M"},
	{"schedule.open", "打开日程", "Open schedules", "Mod+Shift+D"},
	{"assistant.open", "打开助手", "Open assistant", "Mod+Shift+G"},
	{"launcher.open", "打开桌面设置", "Open desktop settings", "Mod+Shift+L"},
	{"harness.restart", "重启 Harness", "Restart Harness", "Mod+Alt+F5"},
	{"window.refresh", "重新绘制界面", "Refresh native interface", "Mod+F5"},
	{"settings.mcp", "打开 MCP 设置", "Open MCP settings", ""},
	{"settings.skills", "打开技能设置", "Open skills settings", ""},
	{"settings.about", "打开关于", "Open about", ""},
	{"shortcuts.open", "打开快捷键设置", "Open shortcut settings", "Mod+/"},
	{"window.zoomIn", "放大应用界面", "Zoom in", ""},
	{"window.zoomOut", "缩小应用界面", "Zoom out", ""},
	{"window.actualSize", "应用界面实际大小", "Actual size", "Mod+0"},
	{"window.fullscreen", "切换应用全屏", "Toggle fullscreen", "Mod+Shift+I"},
	{"pane.split", "向右拆分当前面板", "Split pane right", "Mod+Shift+\\"},
	{"pane.splitDown", "向下拆分当前面板", "Split pane down", "Mod+Alt+\\"},
	{"pane.fullscreen.toggle", "切换面板全屏", "Toggle pane fullscreen", "Mod+Alt+Enter"},
}

var (
	platformCommands      []Command
	platformIDs           []string
	platformDefaults      map[string]string
	platformDefaultChords map[string]Chord
)

func init() {
	platformCommands = commandsForPlatform(runtime.GOOS)
	platformIDs = make([]string, len(platformCommands))
	platformDefaults = make(map[string]string, len(platformCommands))
	platformDefaultChords = make(map[string]Chord, len(platformCommands))
	for i, c := range platformCommands {
		platformIDs[i] = c.ID
		platformDefaults[c.ID] = c.Default
		if chord, err := Parse(c.Default); err == nil {
			platformDefaultChords[c.ID] = chord
		}
	}
}

func Commands() []Command { return slices.Clone(platformCommands) }
func commandsForPlatform(platform string) []Command {
	out := append([]Command(nil), commands...)
	if platform == "darwin" {
		for i := range out {
			if out[i].ID == "terminal.toggle" {
				out[i].Default = ""
			}
		}
	}
	return out
}
func IDs() []string { return slices.Clone(platformIDs) }

// Defaults returns a fresh mutable copy of the cached platform default bindings.
func Defaults() map[string]string { return maps.Clone(platformDefaults) }

// DefaultFor returns the platform default binding for one command ID with zero
// allocations.
func DefaultFor(id string) string { return platformDefaults[id] }

func effective(overrides map[string]string) map[string]string {
	out := maps.Clone(platformDefaults)
	if len(overrides) == 0 {
		return out
	}
	for id, value := range overrides {
		if _, known := platformDefaults[id]; known {
			out[id] = value
		}
	}
	return out
}

// Mod means Ctrl on Windows/Linux and Command on macOS. Ctrl and Cmd are
// accepted aliases for this portable modifier, never a global registration.
type Chord struct {
	Key        string
	Shift, Alt bool
	SecondKey  string
	// Nil retains the legacy portable Mod spelling. A non-nil slice is an
	// exact physical modifier set, including an explicitly unmodified key.
	Modifiers []string
}

func (c Chord) String() string {
	if c.Key == "" {
		return ""
	}
	if c.Modifiers != nil {
		keys := c.Key
		if c.SecondKey != "" {
			keys += " & " + c.SecondKey
		}
		return strings.Join(append(append([]string{}, c.Modifiers...), keys), "+")
	}
	s := "Mod"
	if c.Shift {
		s += "+Shift"
	}
	if c.Alt {
		s += "+Alt"
	}
	if c.SecondKey != "" {
		return s + "+" + c.Key + " & " + c.SecondKey
	}
	return s + "+" + c.Key
}
func Parse(raw string) (Chord, error) { return parseForPlatform(raw, runtime.GOOS) }
func parseForPlatform(raw, platform string) (Chord, error) {
	if platform == "windows" || platform == "darwin" {
		return parseDesktopChord(raw)
	}
	var chord Chord
	if strings.TrimSpace(raw) == "" {
		return chord, nil
	}
	primary := false
	parts := strings.Split(raw, "+")
	for i, part := range parts {
		token := strings.ToUpper(strings.TrimSpace(part))
		switch token {
		case "MOD", "CTRL", "CONTROL", "CMD", "COMMAND", "CTRLORCMD", "COMMANDORCONTROL":
			if primary {
				return Chord{}, errors.New("修饰键重复 / Duplicate modifier")
			}
			primary = true
		case "SHIFT":
			if chord.Shift {
				return Chord{}, errors.New("修饰键重复 / Duplicate modifier")
			}
			chord.Shift = true
		case "ALT", "OPTION":
			if chord.Alt {
				return Chord{}, errors.New("修饰键重复 / Duplicate modifier")
			}
			chord.Alt = true
		default:
			if i != len(parts)-1 || (token != "ENTER" && !functionKey(token) && (len(token) != 1 || !strings.Contains("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789,\\`.;'[]-=/", token))) {
				return Chord{}, errors.New("暂不支持这个按键 / This key is not supported")
			}
			chord.Key = token
		}
	}
	if !primary || chord.Key == "" {
		return Chord{}, errors.New("请同时按下 Command、Ctrl 或 Alt 修饰键 / Include Command, Ctrl, or Alt in the combination")
	}
	// Keep editing, file-save and application/window-management keys with their
	// existing owner. Other system keys (Super, Space, Tab, Escape) are
	// outside the supported grammar altogether; modified F1–F12 are supported. No OS hook is installed.
	// The original native marketplace action owns Mod+Shift+M; plain
	// Mod+M remains the system minimize chord.
	if strings.Contains("ACFHMQSVWXYZ", chord.Key) && !((chord.Key == "M" || chord.Key == "A") && chord.Shift && !chord.Alt) && !(chord.Key == "F" && chord.Alt && !chord.Shift) {
		return Chord{}, errors.New("此组合由系统或文本编辑操作保留 / This combination is reserved for system or text editing actions")
	}
	if platform == "darwin" && ((chord.Key == "`" && !chord.Alt) || (chord.Shift && strings.Contains("3456", chord.Key)) || (chord.Alt && chord.Key == "D")) {
		return Chord{}, errors.New("此组合由系统或文本编辑操作保留 / This combination is reserved for system or text editing actions")
	}
	return chord, nil
}

// Validate marks every row involved in a collision, including defaults that a
// custom binding collides with. The UI must not save any invalid set.
func Validate(values map[string]string) map[string]string {
	issues, _ := validateAndParse(values, false)
	return issues
}

func validateAndParse(values map[string]string, wantParsed bool) (map[string]string, map[string]Chord) {
	issues := map[string]string{}
	var parsed map[string]Chord
	if wantParsed {
		parsed = make(map[string]Chord, len(platformCommands))
	}
	type acceptedChord struct {
		id         string
		zh         string
		en         string
		modMask    uint8
		code       string
		secondCode string
	}
	seen := make([]acceptedChord, 0, len(platformCommands))
	for _, c := range platformCommands {
		raw, overridden := values[c.ID]
		if !overridden {
			raw = c.Default
		}
		var chord Chord
		var err error
		if raw == c.Default {
			if cached, ok := platformDefaultChords[c.ID]; ok {
				chord = cached
			} else {
				chord, err = Parse(raw)
			}
		} else {
			chord, err = Parse(raw)
		}
		if err != nil {
			issues[c.ID] = err.Error()
			continue
		}
		if chord.Key == "" {
			continue
		}
		mask, code, secondCode := chordOverlapKey(chord)
		for _, previous := range seen {
			if overlapKeysCollide(previous.modMask, previous.code, previous.secondCode, mask, code, secondCode) {
				issues[c.ID] = fmt.Sprintf("已被「%s」占用 / Already used by “%s”", previous.zh, previous.en)
				issues[previous.id] = fmt.Sprintf("已被「%s」占用 / Already used by “%s”", c.ZH, c.EN)
			}
		}
		seen = append(seen, acceptedChord{c.ID, c.ZH, c.EN, mask, code, secondCode})
		if wantParsed {
			parsed[c.ID] = chord
		}
	}
	return issues, parsed
}

// Resolve fails closed per invalid/conflicting row. Unknown IDs cannot become
// commands. An explicit empty string disables only that particular binding.
func Resolve(values map[string]string) map[string]Chord {
	resolveMemo.Lock()
	if resolveMemo.out != nil && maps.Equal(resolveMemo.values, values) {
		out := cloneChords(resolveMemo.out)
		resolveMemo.Unlock()
		return out
	}
	resolveMemo.Unlock()
	out := resolve(values)
	resolveMemo.Lock()
	resolveMemo.values, resolveMemo.out = maps.Clone(values), cloneChords(out)
	resolveMemo.Unlock()
	return out
}

// cloneChords keeps a nil Modifiers nil and an empty one empty.
func cloneChords(in map[string]Chord) map[string]Chord {
	out := make(map[string]Chord, len(in))
	for id, c := range in {
		if c.Modifiers != nil {
			c.Modifiers = append(make([]string, 0, len(c.Modifiers)), c.Modifiers...)
		}
		out[id] = c
	}
	return out
}

// resolveMemo remembers the last Resolve. The page asks on every render pass,
// and validating every pair of chords each time showed in hover frames.
var resolveMemo struct {
	sync.Mutex
	values map[string]string
	out    map[string]Chord
}

func resolve(values map[string]string) map[string]Chord {
	issues, parsed := validateAndParse(values, true)
	out := make(map[string]Chord, len(parsed))
	for id, chord := range parsed {
		if _, bad := issues[id]; !bad {
			out[id] = chord
		}
	}
	return out
}

// Editor keeps uncommitted pointers stable across native UI frames. Only a
// successful persistence call replaces active bindings. It is UI-thread owned.
type Editor struct {
	ReferenceOpen          bool
	Query                  string
	Err, Status, ReadError string
	ResetPending           bool
	Recording              string
	drafts                 map[string]*string
	active                 map[string]string
	persist                func(map[string]string) error
	shared                 bool
	defaults               map[string]string
	resetDrafts            map[string]bool
	reset                  func() error
	reload                 func() (map[string]string, error)
	describe               func(string, string) string
}

func NewEditor(saved map[string]string, loadErr error, persist func(map[string]string) error) *Editor {
	e := &Editor{active: effective(saved), drafts: map[string]*string{}, persist: persist}
	if loadErr != nil {
		e.ReadError = "无法读取 userData/keybindings.json。请检查访问权限后重新启动 Harness / Could not read userData/keybindings.json. Check access permissions, then restart Harness: " + loadErr.Error()
		for id := range e.active {
			e.active[id] = ""
		}
	}
	for id, v := range e.active {
		value := v
		e.drafts[id] = &value
	}
	return e
}
func (e *Editor) Draft(id string) *string { return e.drafts[id] }
func (e *Editor) Values() map[string]string {
	out := map[string]string{}
	for id, p := range e.drafts {
		out[id] = *p
	}
	return out
}
func (e *Editor) Active() map[string]string {
	out := map[string]string{}
	for id, v := range e.active {
		out[id] = v
	}
	return out
}
func (e *Editor) Issues() map[string]string {
	if !e.shared {
		return Validate(e.Values())
	}
	issues := map[string]string{}
	for id, v := range e.Values() {
		if v != e.active[id] {
			if _, err := Parse(v); err != nil {
				issues[id] = err.Error()
			} else if e.describe != nil {
				if issue := e.describe(id, v); issue != "" {
					issues[id] = issue
				}
			}
		}
	}
	return issues
}

// Shared snapshots retain dirty drafts until the caller resolves a stale save.
func (e *Editor) Dirty() bool {
	if e.Recording != "" || e.ResetPending {
		return true
	}
	if len(e.resetDrafts) > 0 {
		return true
	}
	for id, p := range e.drafts {
		if *p != e.active[id] {
			return true
		}
	}
	return false
}
func (e *Editor) UseShared(reset func() error) {
	e.shared = true
	e.reset = reset
	e.defaults = SharedDefaults()
	e.resetDrafts = map[string]bool{}
}
func (e *Editor) Default(id string) string {
	if e.shared {
		return e.defaults[id]
	}
	return platformDefaults[id]
}
func (e *Editor) ResetDraft(id string) {
	if p := e.Draft(id); p != nil {
		*p = e.Default(id)
		if e.shared {
			e.resetDrafts[id] = true
		}
	}
}
func (e *Editor) Discard() {
	for id, p := range e.drafts {
		*p = e.active[id]
	}
	clear(e.resetDrafts)
	e.Err = ""
	e.Recording = ""
}

func (e *Editor) SetReload(reload func() (map[string]string, error)) { e.reload = reload }
func (e *Editor) SetDescribe(describe func(string, string) string)   { e.describe = describe }

// Reload is the explicit discard gesture. A failed read retains the accepted
// bindings; a repaired source clears the previous read error.
func (e *Editor) Reload() bool {
	e.Discard()
	e.ResetPending = false
	e.Status = ""
	if e.reload == nil {
		return true
	}
	values, err := e.reload()
	if err != nil {
		e.ReadError = "Cannot reload shortcut settings: " + err.Error()
		e.Err = e.ReadError
		return false
	}
	e.ReadError, e.Err = "", ""
	e.Reconcile(values)
	return true
}

// Review adopts a newly read revision while retaining the candidate. A later
// retry is a separate explicit save, matching the original stale editor flow.
func (e *Editor) Review() bool {
	if e.reload == nil {
		return false
	}
	values, err := e.reload()
	if err != nil {
		e.Err = "Cannot review shortcut settings: " + err.Error()
		return false
	}
	e.active = values
	e.ReadError, e.Err = "", ""
	e.Status = "Reviewed current shortcuts; retry to save the retained candidate"
	return true
}
func (e *Editor) Reconcile(values map[string]string) bool {
	if e.Dirty() {
		return false
	}
	e.active = values
	for id, v := range values {
		if p := e.drafts[id]; p != nil {
			*p = v
		}
	}
	return true
}
func (e *Editor) Search(query string) []Command {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return slices.Clone(platformCommands)
	}
	out := make([]Command, 0, len(platformCommands))
	for _, c := range platformCommands {
		if strings.Contains(strings.ToLower(c.ID+" "+c.ZH+" "+c.EN+" "+c.Default+" "+*e.Draft(c.ID)), query) {
			out = append(out, c)
		}
	}
	return out
}
func (e *Editor) Save() bool {
	e.Status = ""
	e.Err = ""
	if e.ReadError != "" {
		e.Err = e.ReadError
		return false
	}
	if len(e.Issues()) > 0 {
		e.Err = "请先修复无效按键或冲突 / Fix invalid or conflicting bindings before saving"
		return false
	}
	if e.persist == nil {
		e.Err = "快捷键尚未准备好，请稍后重试 / Shortcuts are not ready. Please try again"
		return false
	}
	values := e.Values()
	patch := map[string]string{}
	for id, v := range values {
		if e.shared && e.resetDrafts[id] && v == e.Default(id) {
			patch[id] = "\x00reset"
			continue
		}
		if e.shared && v == e.active[id] {
			continue
		}
		chord, _ := Parse(v)
		values[id] = chord.String()
		patch[id] = values[id]
	}
	if !e.shared {
		patch = values
	}
	if err := e.persist(patch); err != nil {
		e.Err = "保存失败。原快捷键和当前草稿已保留，请重试 / Could not save. Your previous shortcuts and current draft are preserved. Please retry: " + err.Error()
		return false
	}
	clear(e.resetDrafts)
	e.active = values
	for id, v := range values {
		*e.drafts[id] = v
	}
	e.Status = "已修改 / Modified"
	e.ResetPending = false
	return true
}
func (e *Editor) RequestReset() { e.CancelRecording(); e.ResetPending = true }
func (e *Editor) CancelReset()  { e.ResetPending = false }
func (e *Editor) ConfirmReset() bool {
	if !e.ResetPending {
		return false
	}
	if e.shared && e.reset != nil {
		if err := e.reset(); err != nil {
			e.Err = err.Error()
			return false
		}
		for id, p := range e.drafts {
			*p = e.active[id]
		}
		clear(e.resetDrafts)
		e.ResetPending = false
		e.Err = ""
		e.Status = "已恢复默认快捷键 / Default shortcuts restored"
		return true
	}
	for id, v := range platformDefaults {
		*e.drafts[id] = v
	}
	return e.Save()
}
