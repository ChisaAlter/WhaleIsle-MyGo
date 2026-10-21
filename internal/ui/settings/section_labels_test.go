package settings

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

// The settings rail uses the reference names, not shorter paraphrases: the
// original labels are 通用设置, 界面设置, 内置插件, 鲸鱼娘 and 用量统计.
func TestSettingsSectionLabelsMatchTheReference(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	// English first, then the Chinese copy.
	wantEN := map[SectionID]string{
		SectionGeneral:    "General settings",
		SectionInterface:  "Interface settings",
		SectionPlugins:    "Built-in plugins",
		SectionPet:        "Whale",
		SectionUsageStats: "Usage stats",
	}
	for id, want := range wantEN {
		if got := sectionLabel(id); got != want {
			t.Errorf("EN label %s = %q, want %q", id, got, want)
		}
	}

	i18n.SetLocale("zh")
	wantZH := map[SectionID]string{
		SectionGeneral:    "通用设置",
		SectionInterface:  "界面设置",
		SectionPlugins:    "内置插件",
		SectionPet:        "鲸鱼娘",
		SectionUsageStats: "用量统计",
	}
	for id, want := range wantZH {
		if got := sectionLabel(id); got != want {
			t.Errorf("ZH label %s = %q, want %q", id, got, want)
		}
	}
}
