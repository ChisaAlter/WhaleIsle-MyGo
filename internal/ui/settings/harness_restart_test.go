package settings

import (
	"errors"
	"github.com/egoist/mygo/ui"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
)

func TestSP06DesktopNativeEntry(t *testing.T) {
	for _, name := range []string{"HarnessAutoRestart", "HarnessRestartMaxAttempts", "HarnessRestartBaseDelayMs"} {
		if _, ok := reflect.TypeOf(DesktopFields{}).FieldByName(name); !ok {
			t.Fatalf("Desktop settings has no native restart policy field %s", name)
		}
	}
	if _, ok := reflect.TypeOf(DesktopAccess{}).FieldByName("RestartPolicy"); !ok {
		t.Fatal("Desktop settings has no committed restart policy status")
	}
}

func TestSP06DesktopRestartInputsPersistAndRetainFailedDraft(t *testing.T) {
	oldLocale := i18n.Locale()
	i18n.SetLocale("en")
	defer i18n.SetLocale(oldLocale)
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := store.Get()
	enabled := cfg.HarnessAutoRestart
	attempts := strconv.Itoa(cfg.HarnessRestartMaxAttempts)
	delay := strconv.Itoa(cfg.HarnessRestartBaseDelayMs)
	saved := config.RestartPreferencesFrom(cfg)
	s := harness.New(harness.Options{ReadRestartPolicy: func() harness.RestartPolicy {
		c := store.Get()
		return harness.RestartPolicy{AutoRestart: c.HarnessAutoRestart, MaxAttempts: c.HarnessRestartMaxAttempts, BaseDelay: time.Duration(c.HarnessRestartBaseDelayMs) * time.Millisecond}
	}})
	writes := 0
	failSave := false
	access := &DesktopAccess{Fields: DesktopFields{HarnessAutoRestart: &enabled, HarnessRestartMaxAttempts: &attempts, HarnessRestartBaseDelayMs: &delay}, RestartPolicy: &saved}
	access.OnSave = func() {
		draft, err := config.ParseRestartPreferences(enabled, attempts, delay)
		if err == nil && failSave {
			err = errors.New("synthetic write failure")
		}
		if err == nil {
			err = store.Update(draft.Apply)
		}
		if err != nil {
			access.Err = err.Error()
			access.LastAction = ""
			return
		}
		writes++
		saved = config.RestartPreferencesFrom(store.Get())
		access.Err = ""
		access.LastAction = "Local config saved"
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { desktopPane(c, Model{Desktop: access}, ui.RGB(20, 20, 20), ui.RGB(90, 90, 90)) })
	}, 1100, 1500)
	if err := tt.Click("Local runtime and diagnostics"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("already scheduled retry timers") {
		t.Fatal("pending timer contract missing from desktop UI")
	}
	edit := func(label, value string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		tt.Key(ui.Cmd, ui.KeyA)
		tt.Type(value)
	}
	edit("harness-restart-attempts", "6")
	edit("harness-restart-delay", "2500")
	if err := tt.Click("harness-restart-enabled"); err != nil {
		t.Fatal(err)
	}
	if enabled || saved.AutoRestart != true || writes != 0 {
		t.Fatal("draft toggle applied without save")
	}
	if _, ok := tt.Find("harness-restart-dirty"); !ok {
		t.Fatal("draft not distinguished from saved policy")
	}
	if err := tt.Click("save-desktop-config"); err != nil {
		t.Fatal(err)
	}
	if writes != 1 || saved != (config.RestartPreferences{AutoRestart: false, MaxAttempts: 6, BaseDelayMs: 2500}) || s.RestartPolicy().AutoRestart {
		t.Fatalf("save did not update future policy: %d %+v %+v", writes, saved, s.RestartPolicy())
	}
	if st := s.Status(); st.State != harness.Stopped || st.Generation != 0 {
		t.Fatalf("save changed process lifecycle: %+v", st)
	}
	edit("harness-restart-attempts", "11")
	if err := tt.Click("save-desktop-config"); err != nil {
		t.Fatal(err)
	}
	if access.Err == "" || writes != 1 || attempts != "11" || s.RestartPolicy().MaxAttempts != 6 {
		t.Fatal("invalid input accepted or draft lost")
	}
	edit("harness-restart-attempts", "4")
	failSave = true
	if err := tt.Click("save-desktop-config"); err != nil {
		t.Fatal(err)
	}
	if access.Err != "synthetic write failure" || writes != 1 || attempts != "4" || s.RestartPolicy().MaxAttempts != 6 {
		t.Fatal("failed save changed active state or lost draft")
	}
	failSave = false
	if err := tt.Click("save-desktop-config"); err != nil {
		t.Fatal(err)
	}
	if writes != 2 || s.RestartPolicy().MaxAttempts != 4 || access.Err != "" {
		t.Fatal("retained draft could not be saved")
	}
	if err := tt.Click("save-desktop-config"); err != nil {
		t.Fatal(err)
	}
	if writes != 3 || s.Status().Generation != 0 {
		t.Fatal("repeated save changed lifecycle")
	}
	reopened, err := config.Open(filepath.Dir(store.Path()))
	if err != nil {
		t.Fatal(err)
	}
	if config.RestartPreferencesFrom(reopened.Get()) != saved {
		t.Fatal("saved policy did not reopen")
	}
}

func TestSP06DesktopSaveAndProviderWiring(t *testing.T) {
	raw, err := os.ReadFile("../../../cmd/whaleisle/main.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	start := strings.Index(source, "func (s *mainState) saveDesktopConfig()")
	end := strings.Index(source[start:], "\n// pickAndApplyWorkspace")
	if start < 0 || end < 0 {
		t.Fatal("desktop save seam missing")
	}
	save := source[start : start+end]
	parse := strings.Index(save, "config.ParseRestartPreferences(")
	update := strings.Index(save, "s.cfgStore.Update(")
	receipt := strings.Index(save, "config.RestartPreferencesFrom(cfg) != restartPolicy")
	publish := strings.Index(save, "s.desktopLastAction = i18n.T(i18n.KeySettingsDesktopSaved)")
	if parse < 0 || update < parse || receipt < update || publish < receipt || !strings.Contains(save, "restartPolicy.Apply(c)") {
		t.Fatal("desktop save does not validate, commit and verify policy in order")
	}
	for _, forbidden := range []string{".Start()", ".Stop()", ".Restart()", "startHarness(", "refreshPhoneRemote("} {
		if strings.Contains(save, forbidden) {
			t.Fatalf("save invokes forbidden lifecycle/network action %s", forbidden)
		}
	}
	providerStart := strings.Index(source, "ReadRestartPolicy: func() harness.RestartPolicy {")
	if providerStart < 0 {
		t.Fatal("real supervisor has no live policy provider")
	}
	providerEnd := strings.Index(source[providerStart:], "\n\t\t},")
	if providerEnd < 0 {
		t.Fatal("provider seam missing")
	}
	provider := source[providerStart : providerStart+providerEnd]
	if strings.Count(provider, "cfgStore.Get()") != 1 || strings.Contains(provider, "state.mu.Lock()") {
		t.Fatal("provider must read one committed snapshot without mainState lock")
	}
	for _, field := range []string{"current.HarnessAutoRestart", "current.HarnessRestartMaxAttempts", "current.HarnessRestartBaseDelayMs"} {
		if !strings.Contains(provider, field) {
			t.Fatalf("provider omitted %s", field)
		}
	}
}
