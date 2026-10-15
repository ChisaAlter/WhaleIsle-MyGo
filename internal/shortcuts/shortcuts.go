// Package shortcuts owns native application and page bindings, not system hotkeys.
package shortcuts

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
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
	{"tools.toggle.alternate", "切换右侧面板（备用）", "Toggle tools pane (alternate)", "Mod+Alt+B"},
	{"terminal.toggle", "切换终端", "Toggle terminal", "Mod+`"},
	{"session.search", "搜索对话", "Search conversations", "Mod+Shift+K"},
	{"session.rename", "重命名当前对话", "Rename conversation", "Mod+Shift+R"},
	{"session.fork", "分支当前对话", "Fork conversation", "Mod+Shift+J"},
	{"session.archive", "归档当前对话", "Archive conversation", "Mod+Shift+E"},
	{"session.previous", "上一个对话", "Previous conversation", "Mod+Alt+["},
	{"session.next", "下一个对话", "Next conversation", "Mod+Alt+]"},
	{"workspace.add", "添加工作区", "Add workspace", "Mod+Shift+O"},
	{"workspace.openLocal", "在文件管理器中打开工作区", "Reveal workspace", "Mod+Alt+O"},
	{"workspace.files", "打开工作区文件", "Open workspace files", "Mod+Shift+B"},
	{"browser.open", "打开浏览器面板", "Open browser pane", "Mod+Shift+U"},
	{"terminal.host", "打开 Host 终端面板", "Open Host terminal pane", "Mod+Alt+T"},
	{"sessions.refresh", "刷新对话列表", "Refresh conversations", "Mod+Alt+R"},
	{"market.open", "打开市场", "Open marketplace", "Mod+Shift+P"},
	{"schedule.open", "打开日程", "Open schedules", "Mod+Shift+D"},
	{"assistant.open", "打开助手", "Open assistant", "Mod+Shift+G"},
	{"launcher.open", "打开启动器", "Open launcher", "Mod+Shift+L"},
	{"shortcuts.open", "打开快捷键设置", "Open shortcut settings", "Mod+/"},
	{"window.zoomIn", "放大应用界面", "Zoom in", "Mod+="},
	{"window.zoomOut", "缩小应用界面", "Zoom out", "Mod+-"},
	{"window.actualSize", "应用界面实际大小", "Actual size", "Mod+0"},
	{"window.fullscreen", "切换应用全屏", "Toggle fullscreen", "Mod+Shift+I"},
	{"pane.split", "向右拆分当前面板", "Split pane right", "Mod+Shift+\\"},
	{"pane.splitDown", "向下拆分当前面板", "Split pane down", "Mod+Alt+\\"},
	{"pane.fullscreen.toggle", "切换面板全屏", "Toggle pane fullscreen", "Mod+Alt+Enter"},
}

func Commands() []Command { return commandsForPlatform(runtime.GOOS) }
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
func IDs() []string {
	out := make([]string, 0, len(commands))
	for _, c := range Commands() {
		out = append(out, c.ID)
	}
	return out
}
func Defaults() map[string]string {
	out := map[string]string{}
	for _, c := range Commands() {
		out[c.ID] = c.Default
	}
	return out
}
func effective(overrides map[string]string) map[string]string {
	out := Defaults()
	for _, c := range Commands() {
		if value, ok := overrides[c.ID]; ok {
			out[c.ID] = value
		}
	}
	return out
}

// Mod means Ctrl on Windows/Linux and Command on macOS. Ctrl and Cmd are
// accepted aliases for this portable modifier, never a global registration.
type Chord struct {
	Key        string
	Shift, Alt bool
}

func (c Chord) String() string {
	if c.Key == "" {
		return ""
	}
	s := "Mod"
	if c.Shift {
		s += "+Shift"
	}
	if c.Alt {
		s += "+Alt"
	}
	return s + "+" + c.Key
}
func Parse(raw string) (Chord, error) { return parseForPlatform(raw, runtime.GOOS) }
func parseForPlatform(raw, platform string) (Chord, error) {
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
			if i != len(parts)-1 || (token != "ENTER" && (len(token) != 1 || !strings.Contains("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789,\\`.;'[]-=/", token))) {
				return Chord{}, errors.New("无效按键；支持字母、数字、标点或 Enter / Invalid key; use a letter, digit, punctuation or Enter")
			}
			chord.Key = token
		}
	}
	if !primary || chord.Key == "" {
		return Chord{}, errors.New("必须包含 Mod 和一个按键 / Use Mod plus one key")
	}
	// Keep editing, file-save and application/window-management keys with their
	// existing owner. Other system keys (Super, Space, Tab, Escape, F-keys) are
	// outside the supported grammar altogether. No OS hook is installed.
	if strings.Contains("ACFHMQSVWXYZ", chord.Key) {
		return Chord{}, errors.New("此按键保留给系统、编辑或文件操作 / Reserved for system, editing or file actions")
	}
	if platform == "darwin" && ((chord.Key == "`" && !chord.Alt) || (chord.Shift && strings.Contains("3456", chord.Key)) || (chord.Alt && chord.Key == "D")) {
		return Chord{}, errors.New("macOS 系统保留快捷键 / Reserved macOS system shortcut")
	}
	return chord, nil
}

// Validate marks every row involved in a collision, including defaults that a
// custom binding collides with. The UI must not save any invalid set.
func Validate(values map[string]string) map[string]string {
	values = effective(values)
	issues := map[string]string{}
	seen := map[string]string{}
	for _, c := range Commands() {
		chord, err := Parse(values[c.ID])
		if err != nil {
			issues[c.ID] = err.Error()
			continue
		}
		if chord.Key == "" {
			continue
		}
		canonical := chord.String()
		if previous, ok := seen[canonical]; ok {
			issues[c.ID] = fmt.Sprintf("快捷键冲突 / Conflicts with %s", previous)
			issues[previous] = fmt.Sprintf("快捷键冲突 / Conflicts with %s", c.ID)
		} else {
			seen[canonical] = c.ID
		}
	}
	return issues
}

// Resolve fails closed per invalid/conflicting row. Unknown IDs cannot become
// commands. An explicit empty string disables only that particular binding.
func Resolve(values map[string]string) map[string]Chord {
	values = effective(values)
	issues := Validate(values)
	out := map[string]Chord{}
	for _, c := range Commands() {
		if _, bad := issues[c.ID]; bad {
			continue
		}
		chord, _ := Parse(values[c.ID])
		if chord.Key != "" {
			out[c.ID] = chord
		}
	}
	return out
}

// Editor keeps uncommitted pointers stable across native UI frames. Only a
// successful persistence call replaces active bindings. It is UI-thread owned.
type Editor struct {
	Query                  string
	Err, Status, ReadError string
	ResetPending           bool
	drafts                 map[string]*string
	active                 map[string]string
	persist                func(map[string]string) error
}

func NewEditor(saved map[string]string, loadErr error, persist func(map[string]string) error) *Editor {
	e := &Editor{active: effective(saved), drafts: map[string]*string{}, persist: persist}
	if loadErr != nil {
		e.ReadError = "快捷键配置无法读取；请修复后重启 / Cannot read shortcut settings; repair and restart: " + loadErr.Error()
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
func (e *Editor) Issues() map[string]string { return Validate(e.Values()) }
func (e *Editor) Search(query string) []Command {
	query = strings.ToLower(strings.TrimSpace(query))
	out := []Command{}
	for _, c := range Commands() {
		if query == "" || strings.Contains(strings.ToLower(c.ID+" "+c.ZH+" "+c.EN+" "+c.Default+" "+*e.Draft(c.ID)), query) {
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
		e.Err = "快捷键保存不可用 / Shortcut persistence is unavailable"
		return false
	}
	values := e.Values()
	for id, v := range values {
		chord, _ := Parse(v)
		values[id] = chord.String()
	}
	if err := e.persist(values); err != nil {
		e.Err = "保存失败，草稿已保留 / Save failed; draft retained: " + err.Error()
		return false
	}
	e.active = values
	for id, v := range values {
		*e.drafts[id] = v
	}
	e.Status = "已保存；返回主页后生效 / Saved; active on the main page"
	e.ResetPending = false
	return true
}
func (e *Editor) RequestReset() { e.ResetPending = true }
func (e *Editor) CancelReset()  { e.ResetPending = false }
func (e *Editor) ConfirmReset() bool {
	if !e.ResetPending {
		return false
	}
	e.ResetPending = false
	for id, v := range Defaults() {
		*e.drafts[id] = v
	}
	return e.Save()
}
