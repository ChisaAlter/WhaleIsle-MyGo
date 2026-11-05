package settings

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/desktop/pluginui"
)

// A settings item list is interleaved at its real builtin rows, not appended
// below the page. Hidden builtins do not swallow an external owner's bucket.
type settingsItemContributionScope struct {
	contributions pluginui.PublicContributions
	slot          string
	entries       []pluginui.PublicSlotEntry
	next          int
 visible bool
}

var currentSettingsItems *settingsItemContributionScope

func beginSettingsItemContributions(c *ui.Context, contributions pluginui.PublicContributions, section SectionID) func() {
	previous := currentSettingsItems
	slot := "settings." + string(section) + ".item"
	switch section {
	case SectionGeneral, SectionInterface, SectionAppearance, SectionConversation, SectionFeatures, SectionPet:
	default:
		return func() {}
	}
	scope := &settingsItemContributionScope{contributions: contributions, slot: slot, visible:true}
	if contributions.Entries != nil && contributions.RenderEntry != nil {
		scope.entries = contributions.Entries(slot, "")
	}
	currentSettingsItems = scope
	return func() { scope.renderUntil(c, len(scope.entries)); currentSettingsItems = previous }
}
func (scope *settingsItemContributionScope) renderUntil(c *ui.Context, end int) {
	for scope.next < end {
		entry := scope.entries[scope.next]
		scope.next++
		if entry.Native || !scope.visible {
			continue
		}
		// Rendering the external owner cannot accidentally consume native anchors.
		previous := currentSettingsItems
		currentSettingsItems = nil
		row := ui.Column(c.Key("public-settings-item-" + entry.Handle)).FillWidth().MinWidth(0)
		settingsSearchTarget(c, "public-settings-item-"+entry.Handle, "", row)
		row.Children(func() { scope.contributions.RenderEntry(c, scope.slot, "", entry.Handle) })
		currentSettingsItems = previous
	}
}
// settingsItemBefore returns whether this original builtin is still the
// elected winner. External replacements stay in the source-order cursor and
// are consumed at the next surviving native anchor (or the final tail).
func settingsItemBefore(c *ui.Context, key string) bool {
 scope := currentSettingsItems
 if scope == nil { return true }
 id := settingsItemBuiltinID(scope.slot, key)
 if id == "" { return true }
 if !scope.contributions.NativeWinner(scope.slot, "", id) { return false }
 for i := scope.next; i < len(scope.entries); i++ {
  if scope.entries[i].Native && scope.entries[i].ID == id { scope.renderUntil(c, i); scope.next = i+1; break }
 }
 return true
}
func settingsItemBuiltinID(slot, key string) string {
	var ids map[string]string
	switch slot {
	case "settings.general.item":
		ids = map[string]string{"general-language": "language", "ui-settings.enabled": "developer-tools", "general-shortcuts-row": "shortcuts", "general-close": "close-behavior", "general-autostart": "auto-start-desktop", "ui-chat.performanceUsage": "performance-usage", "session-log-deepseek.enabled": "session-log-deepseek", "general-version": "current-version", "harness-auto-recovery": "harness-restart"}
	case "settings.interface.item":
		ids = map[string]string{"ui-git.titlebarGit": "titlebar-git", "ui-titlebar.terminalToggle": "terminal-toggle", "ui-titlebar.surfacesToggle": "surfaces-toggle", "ui-conversation.composerResize": "composer-resize", "ui-conversation.statsLine": "stats-line", "ui-conversation.viewTabs": "view-tabs", "interface-archive": "show-archived-list"}
	case "settings.appearance.item":
		ids = map[string]string{"general-font-settings": "font-size", "general-size-text": "font-size", "general-family-interface": "font-size", "general-size-interface": "font-size", "general-family-code": "font-size", "general-size-code": "font-size", "effect-row-typingFx": "composer-typing-fx", "effect-row-composerBeam": "composer-beam"}
	case "settings.conversation.item":
		ids = map[string]string{"permission-row": "permission", "ui-conversation.busyEnter": "composer-enter", "ui-chat.transcriptView": "transcript-view", "ui-chat.collapseTiming": "transcript-view", "ui-chat.linkOpening": "link-opening", "ui-conversation.nextPrompt": "next-prompt", "visual-replies.enabled": "visual-replies", "custom-instructions": "custom-instructions"}
	case "settings.features.item":
		ids = map[string]string{"features-assistant": "whale-assistant", "features-project": "project", "features-remote-workspace": "remote-workspace"}
	case "settings.pet.item":
		ids = map[string]string{"pet-assistant": "whale"}
	}
	return ids[key]
}

// Only visible labels and prose enter search. Input values and password fields
// never become an index of an external owner's draft or credentials.
func settingsContributionSearchText(nodes []pluginui.NativeNode) (string, string) {
	title := ""
	details := []string{}
	var visit func([]pluginui.NativeNode)
	visit = func(rows []pluginui.NativeNode) {
		for _, node := range rows {
			if node.Password {
				continue
			}
			for _, text := range []string{node.Label, node.Text} {
				if text = strings.TrimSpace(text); text != "" {
					if title == "" {
						title = text
					} else {
						details = append(details, text)
					}
				}
			}
			visit(node.Children)
		}
	}
	visit(nodes)
	return title, strings.Join(details, " · ")
}

// Pet's child list belongs to its chat tab, not the desktop-behavior tab.
func settingsItemVisibility(slot string, visible bool) {
 if scope := currentSettingsItems; scope != nil && scope.slot == slot { scope.visible = visible }
}

func settingsSearchEntryBuiltinID(entry searchEntry, slot string) string {
 if strings.HasPrefix(entry.target, "public-settings-item-") { return "" }
 if id := settingsItemBuiltinID(slot, entry.target); id != "" { return id }
 for _, key := range entry.reveal {
  if id := settingsItemBuiltinID(slot, key); id != "" { return id }
  if key == "typingFx-settings" { return "composer-typing-fx" }
  if key == "composerBeam-settings" { return "composer-beam" }
 }
 switch entry.section {
 case SectionGeneral:
  ids := map[string]string{providerCopy("语言","Language"):"language",providerCopy("关闭窗口时","When closing the window"):"close-behavior",providerCopy("启动时","On launch"):"auto-start-desktop",providerCopy("Harness 自动恢复","Harness auto-recovery"):"harness-restart"}
  if id := ids[entry.title]; id != "" { return id }
  if entry.target == "harness-restart-attempts-picker" || entry.target == "harness-restart-delay-picker" { return "harness-restart" }
 case SectionInterface:
  if entry.title == providerCopy("显示底部归档管理区","Show the archive manager") { return "show-archived-list" }
 case SectionAppearance:
  if entry.title == providerCopy("字号大小","Font size") && entry.target == entry.title { return "font-size" }
 case SectionFeatures:
  return map[string]string{providerCopy("鲸鱼娘助理","Whale assistant"):"whale-assistant","Project":"project",providerCopy("远程工作区（SSH）","Remote workspace (SSH)"):"remote-workspace"}[entry.title]
 case SectionUsageStats:
  return map[string]string{providerCopy("会话累计费用","Session cost"):"session-cost",providerCopy("官方峰谷时","Official peak and valley hours"):"official-peak-valley"}[entry.title]
 }
 return ""
}
