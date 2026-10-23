package terminal

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestFriendlyTerminalTitleNamesTheShell(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	cases := map[string]string{
		`C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe`: "PowerShell",
		`C:\Windows\System32\cmd.exe`:                                "Command Prompt",
		`/bin/bash`:                                                  "Bash",
		`C:\WINDOWS\System32\wsl.exe`:                                "WSL",
		"Terminal":                                                   "Terminal",
	}
	for title, want := range cases {
		if got := friendlyTerminalTitle(title); got != want {
			t.Fatalf("friendlyTerminalTitle(%q)=%q, want %q", title, got, want)
		}
	}
}
