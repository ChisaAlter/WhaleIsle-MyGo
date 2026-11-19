package settings

import (
	"strings"
	"testing"

	"whaleisle/internal/ui/i18n"
)

// TestProviderHelpAnswersBeforeItListsVariables covers the third-party provider
// form: the live pass found the help text was nothing but eight AWS variable
// names and two internal paths. The answer comes first; the names are the detail
// a person copies when the saved credential is not the one to use.
func TestProviderHelpAnswersBeforeItListsVariables(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	names := []string{"AWS_BEARER_TOKEN_BEDROCK", "AWS_PROFILE", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN", "AWS_REGION", "AWS_DEFAULT_REGION", "AWS_WEB_IDENTITY_TOKEN_FILE"}
	help := providerCredentialHelp(names)
	answer, details, split := strings.Cut(help, "详情")
	if !split {
		t.Fatalf("no details disclosure in the help: %q", help)
	}
	if strings.Contains(answer, "AWS_") {
		t.Fatalf("the answer to what the field does is still a variable list: %q", answer)
	}
	if !strings.Contains(answer, "已保存的凭据") || !strings.Contains(answer, "补齐") {
		t.Fatalf("the answer does not say what happens: %q", answer)
	}
	for _, name := range names {
		if !strings.Contains(details, name) {
			t.Fatalf("the names disappeared, so a person cannot supply the key: %q", details)
		}
	}
	for _, raw := range []string{"$DSH_HOME", "$cwd"} {
		if strings.Contains(help, raw) {
			t.Fatalf("the help kept %q: %q", raw, help)
		}
	}
	if help := providerCredentialHelp(nil); help != "" {
		t.Fatalf("no names must produce no help line: %q", help)
	}
}
