package dirpick

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestDirPickLocaleCatalogZHAndEN(t *testing.T) {
	prev := i18n.Locale()
	t.Cleanup(func() {
		i18n.SetLocale(prev)
	})

	i18n.SetLocale(i18n.LocaleZH)
	if got := CrumbNavigationAriaLabel(); got != "目录导航" {
		t.Fatalf("CrumbNavigationAriaLabel() zh = %q", got)
	}
	if got := DrillFolderAriaLabel(); got != "进入目录" {
		t.Fatalf("DrillFolderAriaLabel() zh = %q", got)
	}
	if got := DirPickLocaleText("folderError.title"); got != "无法打开文件夹" {
		t.Fatalf("DirPickLocaleText(folderError.title) zh = %q", got)
	}

	i18n.SetLocale(i18n.LocaleEN)
	if got := CrumbNavigationAriaLabel(); got != "Folder navigation" {
		t.Fatalf("CrumbNavigationAriaLabel() en = %q", got)
	}
	if got := DrillFolderAriaLabel(); got != "Browse folder" {
		t.Fatalf("DrillFolderAriaLabel() en = %q", got)
	}
	if got := DirPickLocaleText("shortcut.noPicker"); got != "Directory picker unavailable" {
		t.Fatalf("DirPickLocaleText(shortcut.noPicker) en = %q", got)
	}
}
