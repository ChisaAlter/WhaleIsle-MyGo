//go:build functionalacceptance

package pluginmanager_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	manager "whaleisle/internal/pluginmanager"
	marketstore "whaleisle/internal/store/market"
	schedstore "whaleisle/internal/store/schedule"
	settingsstore "whaleisle/internal/store/settings"
)

// This opt-in check exercises existing Go controllers against an owned, real
// Host. It never invokes session/prompt, account, credential or model methods.
func TestFunctionalPluginsSettingsTasks(t *testing.T) {
	stage, base, output := os.Getenv("WHALEISLE_FUNCTIONAL_RUNTIME"), os.Getenv("WHALEISLE_FUNCTIONAL_ROOT"), os.Getenv("WHALEISLE_FUNCTIONAL_OUTPUT")
	if !filepath.IsAbs(stage) || !filepath.IsAbs(base) || !filepath.IsAbs(output) {
		t.Fatal("explicit absolute stage, fresh-run parent and evidence output are required")
	}
	root, err := os.MkdirTemp(base, "plugins-settings-tasks-")
	if err != nil {
		t.Fatal(err)
	}
	type check struct {
		Name, Status, Detail string
		Count                int
	}
	report := struct {
		Started          string  `json:"started"`
		Root             string  `json:"ownedRoot"`
		Runtime          string  `json:"runtime"`
		Checks           []check `json:"checks"`
		HostStopped      bool    `json:"ownedHostStopped"`
		ListenerClosed   bool    `json:"ownedListenerClosed"`
		SubmittedPrompts int     `json:"submittedPrompts"`
		UserGUI          string  `json:"userGUI"`
		PNPMSource       string  `json:"pnpmSource,omitempty"`
		PNPMHash         string  `json:"pnpmSHA256,omitempty"`
	}{Started: time.Now().UTC().Format(time.RFC3339), Root: root, Runtime: stage, UserGUI: "not exercised; real Go stores/controllers and Host only"}
	write := func() {
		raw, _ := json.MarshalIndent(report, "", "  ")
		if e := os.WriteFile(output, append(raw, '\n'), 0600); e != nil {
			t.Errorf("cannot save evidence: %v", e)
		}
	}
	add := func(name, status, detail string, count int) {
		report.Checks = append(report.Checks, check{name, status, detail, count})
		write()
		t.Log(name, status, detail, count)
	}
	failure := func(err error) string {
		if err == nil {
			return ""
		}
		sum := sha256.Sum256([]byte(err.Error()))
		var rpc *dsh.Error
		code := "operation-error"
		if errors.As(err, &rpc) {
			code = "rpc:" + rpc.Code
		}
		var decode *json.UnmarshalTypeError
		if errors.As(err, &decode) {
			code = "decode:" + decode.Value + "->" + decode.Type.String() + ":" + decode.Field
		}
		for _, status := range []string{"HTTP 404", "HTTP 500", "HTTP 503"} {
			if strings.Contains(err.Error(), status) {
				code = status
			}
		}
		return code + " sha256:" + hex.EncodeToString(sum[:])
	}
	userData, workspace := filepath.Join(root, "user-data"), filepath.Join(root, "workspace")
	if err = os.MkdirAll(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv(paths.EnvUserData, userData)
	t.Setenv(paths.EnvVendor, filepath.Join(stage, "vendor"))
	t.Setenv("DSH_HOME", filepath.Join(userData, "dsh-home"))
	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "No external dependencies exist in this fixture", http.StatusNotFound)
	}))
	defer registry.Close()
	home, vendor, node := paths.DshHome(), paths.VendorDir(), filepath.Join(stage, "node", "node.exe")
	harnessRoot := filepath.Join(vendor, "deepseek-harness")
	extraEnv := []string{"npm_config_offline=true", "npm_config_registry=" + registry.URL, "COREPACK_ENABLE_NETWORK=0", "NO_UPDATE_NOTIFIER=1"}
	if source := os.Getenv("WHALEISLE_FUNCTIONAL_PNPM"); source != "" {
		// Reuse the existing native package manager without changing the source,
		// global PATH, user npm configuration, or any install/build approval policy.
		contents, e := os.ReadFile(source)
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(contents)
		report.PNPMSource, report.PNPMHash = source, hex.EncodeToString(sum[:])
		if report.PNPMHash != "51cc0ce092691ea0a9cf4f50b7512f8a202911696d291b2322c8be838c2fa829" {
			t.Fatal("existing pnpm provenance hash differs from verified native binary")
		}
		bin := filepath.Join(root, "owned-bin")
		if e = os.MkdirAll(bin, 0700); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(bin, "pnpm.exe"), contents, 0700); e != nil {
			t.Fatal(e)
		}
		config := filepath.Join(root, "empty-npmrc")
		if e = os.WriteFile(config, nil, 0600); e != nil {
			t.Fatal(e)
		}
		extraEnv = append(extraEnv, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "npm_config_userconfig="+config, "npm_config_globalconfig="+config, "npm_config_store_dir="+filepath.Join(root, "pnpm-store"), "PNPM_HOME="+bin)
	}
	sup := harness.New(harness.Options{
		Install:   harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}),
		Launch:    harness.LaunchOptions{NodeBin: node, HarnessRoot: harnessRoot, VendorDir: vendor, Host: "127.0.0.1", Port: 36780, Home: home, ExtraEnv: extraEnv},
		Prepare:   harness.PrepareOptions{VendorDir: vendor, HarnessRoot: harnessRoot, DshHome: home, ProfileDir: paths.WebProfileDir(), RemoteSet: true, Remote: false},
		Workspace: workspace, ReadyTimeout: 3 * time.Minute,
	})
	var endpoint harness.Endpoint
	stop := func() {
		sup.Stop()
		report.HostStopped = sup.Status().State == harness.Stopped
		if endpoint.Origin != "" {
			u, _ := url.Parse(endpoint.Origin)
			conn, e := net.DialTimeout("tcp", u.Host, time.Second)
			report.ListenerClosed = e != nil
			if conn != nil {
				conn.Close()
			}
		}
	}
	defer func() { stop(); write() }()
	start := func() (*dsh.Client, error) {
		sup.Start()
		deadline := time.Now().Add(195 * time.Second)
		for time.Now().Before(deadline) {
			st := sup.Status()
			if st.State == harness.Ready {
				endpoint = st.Endpoint
				return &dsh.Client{Endpoint: dsh.Endpoint{Origin: st.Endpoint.Origin, Cookie: st.Endpoint.Cookie}, HTTP: &http.Client{Timeout: 100 * time.Second}}, nil
			}
			if st.State == harness.Failed {
				return nil, st.Err
			}
			time.Sleep(50 * time.Millisecond)
		}
		return nil, context.DeadlineExceeded
	}
	client, err := start()
	if err != nil {
		add("owned-host-start", "blocked", failure(err), 0)
		t.Fatal("owned Host did not start; see hash-only evidence")
	}
	add("owned-host-start", "pass", "complete rc.2 + production Prepare, remote disabled, blank own profile", 0)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	wait := func(predicate func() (bool, error)) error {
		deadline := time.Now().Add(100 * time.Second)
		for time.Now().Before(deadline) {
			ok, e := predicate()
			if e != nil {
				return e
			}
			if ok {
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			time.Sleep(20 * time.Millisecond)
		}
		return context.DeadlineExceeded
	}
	controller := manager.New()
	controller.Attach(client, nil)
	defer controller.Detach()
	if err = controller.Load(ctx); err != nil {
		add("PLUG-01 inventory", "fail", failure(err), 0)
	} else {
		st := controller.Snapshot()
		add("PLUG-01 inventory", "pass", st.Status, len(st.Packages))
	}
	if _, err = (manager.Client{Caller: client}).Indexes(ctx); err != nil {
		add("PLUG-02 served indexes", "blocked", failure(err), 0)
	} else {
		add("PLUG-02 served indexes", "pass", "read only; no remote catalog fetch", 0)
	}
	market := marketstore.NewWithPersonalPath(filepath.Join(root, "market-personal.json"))
	market.Attach(client, nil)
	defer market.Detach()
	market.Refresh(ctx)
	if err = wait(func() (bool, error) {
		s := market.Snapshot()
		if s.Err != "" {
			return false, errors.New(s.Err)
		}
		return !s.Loading, nil
	}); err != nil {
		add("PLUG-04 installed market", "fail", failure(err), 0)
	} else {
		add("PLUG-04 installed market", "pass", "real Host bundle inventory", len(market.Snapshot().Rows))
	}
	settings := settingsstore.New()
	describe := func() error {
		settings.Attach(client, nil)
		settings.RefreshDescribe(ctx)
		return wait(func() (bool, error) {
			s := settings.Snapshot()
			if s.Err != "" {
				return false, errors.New(s.Err)
			}
			return !s.Loading && len(s.Namespaces) > 0, nil
		})
	}
	readBusy := func() (string, bool) {
		for _, ns := range settings.Snapshot().Namespaces {
			if ns.NS == "ui-conversation" {
				var value struct {
					BusyEnter string `json:"busyEnter"`
				}
				_ = json.Unmarshal(ns.Value, &value)
				for _, f := range ns.Fields {
					if f.Key == "busyEnter" && f.Kind == settingsstore.FieldEnum {
						return value.BusyEnter, true
					}
				}
			}
		}
		return "", false
	}
	writeBusy := func(value string) error {
		settings.PatchDescribedField(ctx, "ui-conversation", "busyEnter", value)
		return wait(func() (bool, error) {
			s := settings.Snapshot()
			if s.Err != "" {
				return false, errors.New(s.Err)
			}
			v, ok := readBusy()
			return !s.Saving && ok && v == value, nil
		})
	}
	settingsChanged, originalBusy, chosenBusy := false, "", ""
	if err = describe(); err != nil {
		add("SET-18 served describe", "fail", failure(err), 0)
	} else {
		add("SET-18 served describe", "pass", "Go schema decoder against real Host", len(settings.Snapshot().Namespaces))
		if originalBusy, settingsChanged = readBusy(); settingsChanged && (originalBusy == "queue" || originalBusy == "steer") {
			chosenBusy = "steer"
			if originalBusy == "steer" {
				chosenBusy = "queue"
			}
			if err = writeBusy(chosenBusy); err != nil {
				settingsChanged = false
				add("SET-03 preference save", "fail", failure(err), 0)
			} else {
				add("SET-03 preference save", "pass", "ui-conversation.busyEnter only; no prompt", 1)
			}
		} else {
			settingsChanged = false
			add("SET-03 preference save", "blocked", "busyEnter enum not served", 0)
		}
	}
	fixtureName := "whaleisle-functional-inert-fixture"
	fixtureDir := filepath.Join(root, "fixture-plugin")
	if err = os.MkdirAll(fixtureDir, 0700); err != nil {
		t.Fatal(err)
	}
	fixtureSession := "functional-fixture-" + filepath.Base(root)
	manifest := map[string]any{"name": fixtureName, "version": "1.0.0", "type": "module", "main": "index.js", "exports": map[string]string{".": "./index.js", "./package.json": "./package.json", "./patch.yml": "./patch.yml"}, "dsh": map[string]any{"bundle": map[string]string{"patch": "./patch.yml"}}}
	manifestRaw, _ := json.Marshal(manifest)
	fixtureJS := fmt.Sprintf("export const name = 'functional-inert-fixture';\nexport const inject = ['schedule'];\nexport async function apply(ctx) {\n const sessionId = %q;\n const existing = await ctx.schedule.catalog();\n if (!existing.some(row => row.sessionId === sessionId && row.title === 'Functional inert future fixture')) await ctx.schedule.create(sessionId, {title:'Functional inert future fixture',prompt:'Owned test fixture. Never deliver during this test.',at:'2099-12-31T00:00:00.000Z'});\n}\n", fixtureSession)
	for name, contents := range map[string][]byte{"package.json": manifestRaw, "index.js": []byte(fixtureJS), "patch.yml": []byte("- insert:\n    - id: functional-inert-fixture\n      name: '" + fixtureName + "'\n")} {
		if err = os.WriteFile(filepath.Join(fixtureDir, name), contents, 0600); err != nil {
			t.Fatal(err)
		}
	}
	pc := manager.Client{Caller: client}
	inspection, inspectErr := pc.Inspect(ctx, fixtureDir, &registry.URL)
	installed := false
	if inspectErr != nil || inspection.Status != "accepted" {
		add("PLUG-03 local inspection", "blocked", inspection.Problem+" "+failure(inspectErr), 0)
	} else {
		add("PLUG-03 local inspection", "pass", "own directory; no dependencies or install scripts", 1)
		a := controller.Actions(controller.Snapshot().Owner)
		a.OpenInstall()
		a.EditInstall(fixtureDir, registry.URL)
		a.Inspect()
		err = wait(func() (bool, error) {
			s := controller.Snapshot()
			if s.Install.Error != "" {
				return false, errors.New(s.Install.Error)
			}
			return s.Install.Inspection != nil, nil
		})
		if err == nil {
			a.RequestInstall()
			a.Confirm()
			err = wait(func() (bool, error) {
				s := controller.Snapshot()
				if s.Install.Error != "" {
					return false, errors.New(s.Install.Error)
				}
				return !s.Busy && s.Install.Result != nil, nil
			})
		}
		if err == nil {
			r := controller.Snapshot().Install.Result
			if r.Error != nil || r.Application == "failed" {
				code := "failed"
				if r.Error != nil {
					code = r.Error.Code
				}
				add("PLUG-03 local install", "blocked", code+" (real Host refusal retained)", 0)
			} else {
				installed = true
				add("PLUG-03 local install", "pass", r.Application, 1)
			}
		} else {
			add("PLUG-03 local install", "blocked", failure(err), 0)
		}
	}
	schedules := schedstore.New()
	schedules.Attach(client, nil)
	defer schedules.Detach()
	refreshTasks := func() error {
		schedules.Refresh(ctx, fixtureSession)
		return wait(func() (bool, error) {
			s := schedules.Snapshot()
			if s.Err != "" {
				return false, errors.New(s.Err)
			}
			return !s.Loading, nil
		})
	}
	var task schedstore.Entry
	if err = refreshTasks(); err != nil {
		add("TASK-03 catalog", "fail", failure(err), 0)
	} else {
		for _, row := range schedules.Snapshot().Rows {
			if row.SessionID == fixtureSession {
				task = row
				break
			}
		}
		add("TASK-03 catalog", "pass", "real Host catalog", len(schedules.Snapshot().Rows))
	}
	if installed && task.ID != "" {
		at, e := time.Parse(time.RFC3339Nano, task.ScheduledAt)
		if e != nil || at.Year() != 2099 {
			t.Fatal("fixture task is not safely in 2099")
		}
		add("TASK-04 create service", "pass", "original Schedule.create called by inert own fixture; UI creation not exercised", 1)
		page, e := schedules.DeliveryHistory(ctx, task, "")
		if e != nil {
			add("TASK-03 history", "fail", failure(e), 0)
		} else if len(page.Records) != 0 {
			add("TASK-03 history", "fail", "unexpected deliveries for owned 2099 task", len(page.Records))
		} else {
			add("TASK-03 history", "pass", "no deliveries", len(page.Records))
		}
		title := "Functional inert future fixture edited"
		if e = schedules.Update(ctx, task, &title, nil, nil); e != nil {
			add("TASK-04 edit", "fail", failure(e), 0)
		} else {
			add("TASK-04 edit", "pass", "captured expected record; future instant unchanged", 1)
		}
	} else {
		add("TASK-04 own fixture CRUD", "blocked", "local fixture was not installed/loaded; no alternative create RPC or model prompt attempted", 0)
	}
	if installed {
		change, e := pc.SetBundleEnabled(ctx, fixtureName, false)
		if e != nil || change.Error != nil {
			add("PLUG-03 disable", "fail", failure(e), 0)
		} else {
			add("PLUG-03 disable", "pass", change.Application, 1)
		}
	}
	stop()
	client, err = start()
	if err != nil {
		add("restart", "fail", failure(err), 0)
		t.Fatal("own restart failed")
	}
	pc = manager.Client{Caller: client}
	controller.Attach(client, nil)
	schedules.Attach(client, nil)
	if settingsChanged {
		if e := describe(); e != nil {
			add("SET-03 restart persistence", "fail", failure(e), 0)
		} else {
			value, ok := readBusy()
			if ok && value == chosenBusy {
				add("SET-03 restart persistence", "pass", "saved busyEnter restored after actual Host restart", 1)
			} else {
				add("SET-03 restart persistence", "fail", "accepted preference was not durable", 0)
			}
		}
	}
	if installed {
		bundles, e := pc.Bundles(ctx)
		foundDisabled := false
		for _, b := range bundles {
			if b.Name == fixtureName && b.Installed && !b.Enabled {
				foundDisabled = true
			}
		}
		if e == nil && foundDisabled {
			add("PLUG-03 restart persistence", "pass", "fixture installed and disabled", 1)
		} else {
			add("PLUG-03 restart persistence", "fail", failure(e), 0)
		}
	}
	if task.ID != "" {
		if e := refreshTasks(); e != nil {
			add("TASK-04 restart persistence", "fail", failure(e), 0)
		} else {
			for _, row := range schedules.Snapshot().Rows {
				if row.ID == task.ID {
					task = row
				}
			}
			if task.Title == "Functional inert future fixture edited" {
				add("TASK-04 restart persistence", "pass", "edited future record restored", 1)
			} else {
				add("TASK-04 restart persistence", "fail", "edited task was not restored", 0)
			}
			if e = schedules.DeleteEntry(ctx, task); e != nil {
				add("TASK-04 delete", "fail", failure(e), 0)
			} else {
				add("TASK-04 delete", "pass", "own fixture only", 1)
			}
		}
	}
	if settingsChanged {
		if e := writeBusy(originalBusy); e != nil {
			add("SET-03 restore", "fail", failure(e), 0)
		} else {
			add("SET-03 restore", "pass", "original effective preference restored", 1)
		}
	}
	if installed {
		result, e := pc.Remove(ctx, fixtureName)
		if e != nil || result.Error != nil {
			add("PLUG-03 remove", "fail", failure(e), 0)
		} else {
			add("PLUG-03 remove", "pass", result.Application, 1)
		}
	}
	stop()
	client, err = start()
	if err != nil {
		add("final restart", "fail", failure(err), 0)
		return
	}
	pc = manager.Client{Caller: client}
	schedules.Attach(client, nil)
	if settingsChanged {
		if e := describe(); e != nil {
			add("SET-03 restored persistence", "fail", failure(e), 0)
		} else {
			value, ok := readBusy()
			if ok && value == originalBusy {
				add("SET-03 restored persistence", "pass", "restored preference survived second restart", 1)
			} else {
				add("SET-03 restored persistence", "fail", "restore not durable", 0)
			}
		}
	}
	if installed {
		bundles, e := pc.Bundles(ctx)
		remains := false
		for _, b := range bundles {
			remains = remains || b.Name == fixtureName && b.Installed
		}
		if e == nil && !remains {
			add("PLUG-03 removed persistence", "pass", "fixture absent after second restart", 0)
		} else {
			add("PLUG-03 removed persistence", "fail", failure(e), 0)
		}
	}
	if task.ID != "" {
		if e := refreshTasks(); e != nil {
			add("TASK-04 deleted persistence", "fail", failure(e), 0)
		} else {
			remains := false
			for _, row := range schedules.Snapshot().Rows {
				remains = remains || row.SessionID == fixtureSession
			}
			if !remains {
				add("TASK-04 deleted persistence", "pass", "own task absent after second restart", 0)
			} else {
				add("TASK-04 deleted persistence", "fail", "task survived deletion", 0)
			}
		}
	}
	for _, c := range report.Checks {
		if c.Status == "fail" {
			t.Errorf("actual operation failed: %s %s", c.Name, c.Detail)
		}
	}
}
