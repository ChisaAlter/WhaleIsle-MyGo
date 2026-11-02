package main

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"whaleisle/internal/harness/installctrl"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Read only the new independently owned catalog, so this regression does not
// dilute another migration's declared scope or duplicate its strings in cmd/.
func TestCmdInlineAuditCatalogFormatting(t *testing.T) {
	path := filepath.Join("..", "..", "internal", "ui", "i18n", "catalog_cmd_inline.go")
	fs := token.NewFileSet()
	file, err := parser.ParseFile(fs, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	verbs := regexp.MustCompile(`%[+#.0-9\-\[\]]*[a-zA-Z]`)
	count := 0
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		fn, ok := call.Fun.(*ast.Ident)
		if !ok || fn.Name != "Register" || len(call.Args) != 3 {
			return true
		}
		keyID, ok := call.Args[0].(*ast.Ident)
		if !ok {
			return true
		}
		obj := file.Scope.Lookup(keyID.Name)
		if obj == nil {
			t.Fatalf("catalog key %s missing", keyID.Name)
		}
		decl := obj.Decl.(*ast.ValueSpec)
		lit := decl.Values[0].(*ast.BasicLit)
		key := strings.Trim(lit.Value, "\"")
		zh, en := i18n.Lookup(i18n.LocaleZH, key), i18n.Lookup(i18n.LocaleEN, key)
		if !i18n.Registered(key) || zh == "" || en == "" || zh == en {
			t.Errorf("missing locale pair %s: %q / %q", key, zh, en)
		}
		for _, r := range en {
			if unicode.Is(unicode.Han, r) {
				t.Errorf("English copy still Chinese for %s: %q", key, en)
				break
			}
		}
		if !reflect.DeepEqual(verbs.FindAllString(zh, -1), verbs.FindAllString(en, -1)) {
			t.Errorf("format arguments differ for %s: %q / %q", key, zh, en)
		}
		count++
		return true
	})
	if count == 0 {
		t.Fatal("no audited catalog entries checked")
	}
}

func TestCmdInlineAuditImportAndPreviewUseCurrentLocale(t *testing.T) {
	old := i18n.Locale()
	defer i18n.SetLocale(old)
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		t.Run(locale, func(t *testing.T) {
			i18n.SetLocale(locale)
			for _, tc := range []struct {
				name, key string
				result    installctrl.Result
				approve   func([]string) bool
			}{
				{"empty", i18n.KeyCmdImportBuildListInvalid, installctrl.Result{NeedsAllowBuilds: true}, nil},
				{"declined", i18n.KeyCmdImportBuildDeclined, installctrl.Result{NeedsAllowBuilds: true, AllowBuilds: []string{"esbuild"}}, func([]string) bool { return false }},
				{"repeated", i18n.KeyCmdImportBuildRepeated, installctrl.Result{NeedsAllowBuilds: true, AllowBuilds: []string{"esbuild"}}, func([]string) bool { return true }},
			} {
				t.Run(tc.name, func(t *testing.T) {
					calls := 0
					out := installImportPlugin(context.Background(), "fixture", func([]string) installctrl.Result { calls++; return tc.result }, tc.approve)
					if out.Error != i18n.T(tc.key) || out.OK {
						t.Fatalf("wrong locale error %+v", out)
					}
					want := 1
					if tc.name == "repeated" {
						want = 2
					}
					if calls != want {
						t.Fatalf("copy edit changed retry behavior: %d", calls)
					}
				})
			}
			if got := officePreviewFailure(errors.New("response exceeds bounded wire limit")); got != i18n.T(i18n.KeyCmdOfficeTransportTooLarge) {
				t.Fatalf("size error %q", got)
			}
			if got := officePreviewFailure(errors.New("fixture detail")); got != fmt.Sprintf(i18n.T(i18n.KeyCmdOfficeRetryFailure), "fixture detail") {
				t.Fatalf("detail argument changed: %q", got)
			}
			s := &mainState{}
			s.openSettingsSkillCreate()
			if s.settingsSkillCreate.err != i18n.T(i18n.KeyCmdRuntimeDisconnected) {
				t.Fatalf("create error %q", s.settingsSkillCreate.err)
			}
			s.openSettingsSkillHub()
			if s.settingsSkillHub.err != i18n.T(i18n.KeyCmdRuntimeDisconnected) {
				t.Fatalf("hub error %q", s.settingsSkillHub.err)
			}
		})
	}
}

func TestCmdInlineAuditVisibleLocalFailuresUseLocale(t *testing.T) {
	old := i18n.Locale()
	defer i18n.SetLocale(old)
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		t.Run(locale, func(t *testing.T) {
			i18n.SetLocale(locale)
			wantError := func(err error, key string) {
				t.Helper()
				if err == nil || err.Error() != i18n.T(key) {
					t.Fatalf("locale=%s key=%s got=%v", locale, key, err)
				}
			}
			sentinel := errLocalDeliveryPermission
			wrappedPermission := fmt.Errorf("preview: %w", sentinel)
			wantError(errLocalDeliveryPermission, i18n.KeyCmdLocalPreviewPermissionEnded)
			if errLocalDeliveryPermission != sentinel || !errors.Is(wrappedPermission, errLocalDeliveryPermission) {
				t.Fatal("locale switch changed permission sentinel identity")
			}
			s := &mainState{}
			wantError(s.setAppearanceTheme("dark"), i18n.KeyCmdConfigStoreNotReady)
			s.hostAppearanceAccepted = true
			wantError(s.setAppearanceTheme("dark"), i18n.KeyCmdHostAppearanceUnavailable)
			wantError(s.saveAppearancePreferences(theme.Preferences{}), i18n.KeyCmdHostAppearanceUnavailable)
			_, err := projectQueueChange(projectflow.RequestPage{}, nil, nil, "fixture", "enqueue")
			wantError(err, i18n.KeyCmdProjectQueueUnavailable)
			wantError(submitProjectRecoveryDraft(&composer.DraftState{Text: "retained"}, "replacement", func() { t.Fatal("pending draft sent") }), i18n.KeyCmdProjectLeadDraftPending)
			_, err = insertNativeComposerFiles(&composer.DraftState{Sending: true}, "", nil, true)
			wantError(err, i18n.KeyCmdComposerWaitSendFiles)
			limits := &session.ComposerImageLimits{MediaTypes: []string{"image/png"}, MaxImagesPerMessage: 0, MaxImageBytes: 4, MaxMessageImageBytes: 4}
			err = validateComposerImageIntake([]session.ComposerAttachment{{ImageData: "YQ==", MediaType: "image/png"}}, nil, limits)
			if err == nil || err.Error() != fmt.Sprintf(i18n.T(i18n.KeyCmdComposerImageCountLimit), 0) {
				t.Fatalf("image format argument/locale changed: %v", err)
			}
			original := errors.New("fixture cause")
			wrapped := fmt.Errorf(i18n.T(i18n.KeyCmdCreatedBotOpenFailed), original)
			if !errors.Is(wrapped, original) {
				t.Fatal("localized %w lost its cause")
			}
		})
	}
}

func TestCmdInlineAuditPreservesIntentionalLocaleAndProtocolLiterals(t *testing.T) {
	// The migration should never translate these Host method/field discriminants
	// or locale-aware helper pairs a second time.
	for file, needles := range map[string][]string{
		"market_catalog.go":  {`marketMessage("运行时未连接，无法安装。", "The runtime is not connected; installation is unavailable.")`},
		"office_document.go": {`strings.Contains(err.Error(), "response exceeds bounded wire limit")`, `strings.Contains(err.Error(), "harness not ready")`},
		"import_plugins.go":  {`ExpectedPackageName: row.ID`, `NormalizeAllowBuilds(result.AllowBuilds)`, `ctx.Err()`},
	} {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, needle := range needles {
			if !strings.Contains(string(body), needle) {
				t.Fatalf("%s lost invariant %q", file, needle)
			}
		}
	}
}
