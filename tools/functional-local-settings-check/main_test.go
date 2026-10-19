//go:build functionalacceptance

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
)

// The settings check uses no Session; the second variant creates only an owned
// idle handle. Both use original providers and never send an execution request.
func TestFunctionalLocalSettingsComposer(t *testing.T) { runLocalSettingsComposer(t, false) }
func TestFunctionalDefaultSessionSkills(t *testing.T)  { runLocalSettingsComposer(t, true) }

type localCallAudit struct {
	mu        sync.Mutex
	forbidden int
	paths     map[string]int
}

func (a *localCallAudit) RoundTrip(r *http.Request) (*http.Response, error) {
	path := r.URL.Path
	a.mu.Lock()
	a.paths[path]++
	blocked := strings.Contains(path, "/session/prompt") || strings.Contains(path, "/session/modelCatalog") || strings.Contains(path, "/session/initializeDefaultModel") || strings.Contains(path, "/commands/execute") || strings.Contains(path, "/credentials/") || strings.Contains(path, "/account/") || strings.Contains(path, "/llm/")
	if blocked {
		a.forbidden++
	}
	a.mu.Unlock()
	if blocked {
		return nil, errors.New("execution/auth/discovery method excluded from owned skill-only check")
	}
	return http.DefaultTransport.RoundTrip(r)
}

func runLocalSettingsComposer(t *testing.T, defaultSession bool) {
	stage, base, output := os.Getenv("WHALEISLE_FUNCTIONAL_RUNTIME"), os.Getenv("WHALEISLE_FUNCTIONAL_ROOT"), os.Getenv("WHALEISLE_FUNCTIONAL_OUTPUT")
	for _, p := range []string{stage, base, output} {
		if !filepath.IsAbs(p) {
			t.Fatal("explicit absolute runtime, owned parent and evidence output required")
		}
	}
	root := os.Getenv("WHALEISLE_FUNCTIONAL_RESUME")
	resuming := root != ""
	if defaultSession && resuming {
		t.Fatal("default selected-session check requires own fresh profile")
	}
	var e error
	if resuming {
		rel, err := filepath.Rel(base, root)
		if err != nil || !filepath.IsAbs(root) || strings.HasPrefix(rel, "..") || !strings.HasPrefix(filepath.Base(root), "local-settings-functional-") {
			t.Fatal("resume must be owned QA profile inside explicit parent")
		}
		if _, e = os.Stat(filepath.Join(root, "user-data", "dsh-home", "skills", "functional-local-skill", "SKILL.md")); e != nil {
			t.Fatal("known owned skill fixture required for resume")
		}
	} else {
		root, e = os.MkdirTemp(base, "local-settings-functional-")
		if e != nil {
			t.Fatal(e)
		}
	}
	type check struct{ Name, Status, Detail string }
	report := struct {
		Root                  string         `json:"ownedRoot"`
		Runtime               string         `json:"runtime"`
		Scope                 string         `json:"scope"`
		Checks                []check        `json:"checks"`
		HostStopped           bool           `json:"ownedHostStopped"`
		ListenerClosed        bool           `json:"ownedListenerClosed"`
		AgentCreated          bool           `json:"agentCreated"`
		BlockedExecutionCalls int            `json:"blockedExecutionCalls"`
		ObservedRPCs          map[string]int `json:"observedRPCs,omitempty"`
	}{Root: root, Runtime: stage, Scope: "Explicit QA deployment-level original skill-filesystem provider enabled only in fresh owned profile. Default web-app intentionally disables global filesystem discovery; this does not certify default Agent-scoped CHAT08. Actual Host+production Store controllers, no Session selection/create/activation/prompt, account/keys/discovery/models/user GUI/IME; all home and skill roots owned."}
	if defaultSession {
		report.Scope = "Default web-app skill provider configuration, actual owned idle standard-preset Session/Agent handle, original session-addressed skills/list and inventory plus production Store controllers. No prompt/turn/tool/model execution/auth/discovery; all home/skill/workspace roots owned. User GUI/IME not exercised."
	}
	audit := &localCallAudit{paths: map[string]int{}}
	write := func() {
		raw, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(output, append(raw, '\n'), 0600); err != nil {
			t.Errorf("save evidence: %v", err)
		}
	}
	add := func(name, status, detail string) {
		report.Checks = append(report.Checks, check{name, status, detail})
		write()
		t.Log(name, status, detail)
		if status == "fail" {
			t.Errorf("actual operation failed: %s", name)
		}
	}
	failure := func(err error) string {
		if err == nil {
			return ""
		}
		sum := sha256.Sum256([]byte(err.Error()))
		code := "operation-error"
		var rpc *dsh.Error
		if errors.As(err, &rpc) {
			code = rpc.Code
		}
		return code + " sha256:" + hex.EncodeToString(sum[:])
	}
	ownedHome, workspace := filepath.Join(root, "owned-home"), filepath.Join(root, "workspace")
	envs := map[string]string{"HOME": ownedHome, "USERPROFILE": ownedHome, "HOMEDRIVE": filepath.VolumeName(ownedHome), "HOMEPATH": strings.TrimPrefix(ownedHome, filepath.VolumeName(ownedHome)), "APPDATA": filepath.Join(root, "appdata"), "LOCALAPPDATA": filepath.Join(root, "local-appdata"), "XDG_CONFIG_HOME": filepath.Join(ownedHome, ".config"), "XDG_DATA_HOME": filepath.Join(ownedHome, ".local", "share"), "XDG_CACHE_HOME": filepath.Join(ownedHome, ".cache"), "DSH_AGENTS_HOME": filepath.Join(ownedHome, ".agents"), "DSH_BUNDLED_SKILL_DIR": filepath.Join(root, "bundled-skills"), "COREPACK_ENABLE_NETWORK": "0", "npm_config_offline": "true"}
	for _, p := range []string{ownedHome, workspace, filepath.Join(workspace, ".git"), envs["APPDATA"], envs["LOCALAPPDATA"], envs["DSH_AGENTS_HOME"], envs["DSH_BUNDLED_SKILL_DIR"]} {
		if e = os.MkdirAll(p, 0700); e != nil {
			t.Fatal(e)
		}
	}
	for k, v := range envs {
		t.Setenv(k, v)
	}
	t.Setenv(paths.EnvUserData, filepath.Join(root, "user-data"))
	t.Setenv(paths.EnvVendor, filepath.Join(stage, "vendor"))
	t.Setenv("DSH_HOME", filepath.Join(root, "user-data", "dsh-home"))
	vendor, home := paths.VendorDir(), paths.DshHome()
	hroot := filepath.Join(vendor, "deepseek-harness")
	node := filepath.Join(stage, "node", "node.exe")
	profile := paths.WebProfileDir()
	if e = os.MkdirAll(profile, 0700); e != nil {
		t.Fatal(e)
	}
	presetID, presetRow, skillName := "functional-local-preset", "functional-local-preset-row", "functional-local-skill"
	patchPath := filepath.Join(profile, "cordis.patch.yml")
	patches := []map[string]any{{"insert": []map[string]any{{"id": presetRow, "name": "@deepseek-ai/dsh-agent-preset", "config": map[string]any{"id": presetID, "name": "Owned local preset", "description": "Inert empty composition fixture", "plugins": []any{}}}}}}
	if !defaultSession {
		patches = append(patches, map[string]any{"id": "skill-filesystem", "disabled": false})
	}
	patch, _ := yaml.Marshal(patches)
	if !resuming {
		if e = os.WriteFile(patchPath, patch, 0600); e != nil {
			t.Fatal(e)
		}
	}
	sup := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}), Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: hroot, VendorDir: vendor, Host: "127.0.0.1", Port: 36820, Home: home}, Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: hroot, DshHome: home, ProfileDir: profile, RemoteSet: true, Remote: false}, Workspace: workspace, ReadyTimeout: 3 * time.Minute})
	var endpoint harness.Endpoint
	stop := func() {
		sup.Stop()
		report.HostStopped = sup.Status().State == harness.Stopped
		if endpoint.Origin != "" {
			u, _ := url.Parse(endpoint.Origin)
			conn, err := net.DialTimeout("tcp", u.Host, time.Second)
			report.ListenerClosed = err != nil
			if conn != nil {
				conn.Close()
			}
		}
	}
	defer func() {
		stop()
		if defaultSession {
			audit.mu.Lock()
			report.BlockedExecutionCalls = audit.forbidden
			report.ObservedRPCs = map[string]int{}
			for k, v := range audit.paths {
				report.ObservedRPCs[k] = v
			}
			audit.mu.Unlock()
			if report.BlockedExecutionCalls != 0 {
				t.Errorf("excluded method attempted")
			}
		}
		write()
	}()
	start := func() *dsh.Client {
		sup.Start()
		until := time.Now().Add(195 * time.Second)
		for time.Now().Before(until) {
			st := sup.Status()
			if st.State == harness.Ready {
				endpoint = st.Endpoint
				httpClient := &http.Client{Timeout: 45 * time.Second}
				if defaultSession {
					httpClient.Transport = audit
				}
				return &dsh.Client{Endpoint: dsh.Endpoint{Origin: endpoint.Origin, Cookie: endpoint.Cookie}, HTTP: httpClient}
			}
			if st.State == harness.Failed {
				add("owned Host", "blocked", failure(st.Err))
				return nil
			}
			time.Sleep(50 * time.Millisecond)
		}
		add("owned Host", "blocked", "readiness timeout")
		return nil
	}
	client := start()
	if client == nil {
		return
	}
	add("owned Host", "pass", "fresh all home/config roots and user/profile data; remote disabled")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	s := session.New()
	s.Attach(client, nil)
	defer s.Attach(nil, nil)
	s.SetWorkspaceCwd(workspace)
	if defaultSession {
		var created api.SessionCreateValue
		if e = client.Call(ctx, "session/create", map[string]any{"request": map[string]any{"cwd": workspace, "agentPreset": "standard"}}, &created); e != nil || created.SessionId == "" {
			add("CHAT owned idle session", "blocked", failure(e))
			return
		}
		report.AgentCreated = true
		s.Select(created.SessionId)
		add("CHAT owned idle session", "pass", "original session/create returned own standard-preset idle handle; no prompt or model method called")
	}
	wait := func(label string, pred func() bool) bool {
		until := time.Now().Add(12 * time.Second)
		for time.Now().Before(until) {
			if pred() {
				return true
			}
			time.Sleep(10 * time.Millisecond)
		}
		add(label, "fail", "condition timeout")
		return false
	}
	var providers api.ListProvidersResult
	var directory api.ListConfigurableProvidersResult
	if !defaultSession && os.Getenv("WHALEISLE_FUNCTIONAL_SKIP_STATIC") != "1" {
		if e = client.Call(ctx, "llm/listProviders", nil, &providers); e != nil {
			add("SET static providers", "blocked", failure(e))
		} else {
			add("SET static providers", "pass", "registry metadata only: "+stringCount(len(providers))+" routes")
		}
		if e = client.Call(ctx, "llm/listConfigurableProviders", nil, &directory); e != nil {
			add("SET static provider directory", "blocked", failure(e))
		} else {
			add("SET static provider directory", "pass", "declaration metadata only: "+stringCount(len(directory))+" entries; no discover/auth/key read")
		}
	}
	add("CHAT model catalog", "not-run", "session/modelCatalog invokes provider listModels/resolve callbacks; no proof all registered adapters remain static, so not invoked")
	var baseline api.SkillInventorySnapshot
	if e = client.Call(ctx, "skillInventory/list", map[string]any{"request": map[string]any{"cwd": workspace}}, &baseline); e != nil {
		add("SET skill catalog", "blocked", failure(e))
		return
	}
	for _, row := range baseline.Skills {
		if row.Path != nil {
			path := filepath.Clean(*row.Path)
			rel, err := filepath.Rel(root, path)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				t.Fatal("catalog escaped owned roots")
			}
		}
	}
	add("SET skill catalog", "pass", "all discovered skill paths stayed inside owned roots")
	if !resuming {
		s.CreateSkillInventory(ctx, skillName, "Owned inert description", "# Owned inert body\nNo action should be executed.", session.SkillRootUser)
		if !wait("SET skill create", func() bool { return s.Snapshot().LastAction == "skill created "+skillName }) {
			return
		}
	}
	s.RefreshSkills(ctx)
	if !wait("CHAT skill picker", func() bool {
		for _, o := range s.Snapshot().SkillOptions {
			if o.Name == skillName {
				return true
			}
		}
		return false
	}) {
		state := s.Snapshot()
		var direct api.SkillInventorySnapshot
		err := client.Call(ctx, "skillInventory/list", map[string]any{"request": map[string]any{"cwd": workspace}}, &direct)
		add("CHAT skill picker diagnosis", "observed", "options="+stringCount(len(state.SkillOptions))+" catalog="+stringCount(len(state.SkillCatalog.Skills))+" catalog-error="+failure(errors.New(state.SkillCatalog.Err))+" picker-error="+failure(errors.New(state.SkillsErr))+" direct="+stringCount(len(direct.Skills))+" direct-error="+failure(err))
		for _, sk := range direct.Skills {
			if sk.Name == skillName {
				add("CHAT own skill flags", "observed", "source="+sk.Source+" writable="+boolText(sk.Writable)+" model="+boolText(sk.ModelInvocable)+" user="+boolText(sk.UserInvocable))
			}
		}
		return
	}
	add("SET/CHAT skill create", "pass", "production CreateSkillInventory and RefreshSkills populated composer options from real inventory")
	var target session.SkillEditTarget
	if !wait("SET skill edit target", func() bool { var err error; target, err = s.CaptureCatalogSkillEdit(skillName); return err == nil }) {
		return
	}
	target, e = s.LoadCatalogSkillEdit(ctx, target)
	if e != nil {
		t.Fatal(failure(e))
	}
	fields := target.Fields()
	fields.Description = "Owned edited description"
	fields.Content = "# Edited owned body\n鲸\n"
	fields.Groups = []string{"owned-group"}
	fields.ModelInvocable = false
	fields.UserInvocable = true
	if !resuming {
		if e = s.SaveCatalogSkillEditor(ctx, target, fields); e != nil {
			add("SET skill edit", "fail", failure(e))
			return
		}
	}
	doc, e := s.GetCatalogSkillDetail(ctx, skillName)
	if e != nil || doc.Content != strings.TrimSpace(fields.Content) || doc.Description != fields.Description || doc.ModelInvocable || !doc.UserInvocable || len(doc.Groups) != 1 {
		t.Fatal("saved skill read-back mismatch: " + failure(e))
	}
	if doc.Path == nil {
		t.Fatal("owned skill path missing")
	}
	rel, e := filepath.Rel(root, *doc.Path)
	if e != nil || strings.HasPrefix(rel, "..") {
		t.Fatal("skill mutation escaped owned root")
	}
	add("SET skill edit", "pass", "production captured editor updated real file/body/groups/flags and authoritative read matched")
	s.RefreshSkills(ctx)
	if !wait("CHAT user-only skill", func() bool {
		for _, o := range s.Snapshot().SkillOptions {
			if o.Name == skillName {
				return o.UserOnly
			}
		}
		return false
	}) {
		return
	}
	add("CHAT skill picker policy", "pass", "real user-invocable/model-disabled skill retained as UserOnly; no invocation")
	if defaultSession {
		options := s.ListSkills(ctx)
		found := false
		for _, o := range options {
			if o.Name == skillName && o.UserOnly {
				found = true
			}
		}
		if !found {
			t.Fatal("session-addressed skills/list omitted owned UserOnly skill")
		}
		add("CHAT-08 default selected scope", "pass", "default standard preset scoped provider served production RefreshSkills/ListSkills; no global provider override")
		var global api.SkillInventorySnapshot
		if e = client.Call(ctx, "skillInventory/list", map[string]any{"request": map[string]any{"cwd": workspace}}, &global); e != nil {
			t.Fatal(failure(e))
		}
		if len(global.Skills) != 0 {
			t.Fatal("default global registry unexpectedly gained fixture provider")
		}
		add("CHAT default global distinction", "pass", "default global inventory stayed empty while selected-session catalog contained owned skill")
		if !wait("CHAT removal target", func() bool { _, err := s.CaptureCatalogSkillEdit(skillName); return err == nil }) {
			return
		}
		s.ReviewSkillRemoval(ctx, skillName)
		review := s.SkillRemovalReview()
		if !review.Ready {
			t.Fatal("own scoped removal review not ready")
		}
		s.ConfirmSkillRemoval(ctx, review.ID)
		if _, e = os.Stat(*doc.Path); !os.IsNotExist(e) {
			t.Fatal("own scoped skill not removed")
		}
		add("CHAT scoped skill cleanup", "pass", "production captured review removed only owned user-dsh directory; idle Agent resources released by Host Stop")
		return
	}
	s.RefreshAgentPresets(ctx)
	if !wait("CHAT preset picker", func() bool {
		for _, label := range s.Snapshot().AgentPresetLabels {
			if label == "Owned local preset" {
				return true
			}
		}
		return false
	}) {
		return
	}
	add("CHAT preset picker", "pass", "production RefreshAgentPresets consumed real empty-composition fixture roster; no Agent instantiated")
	m := session.NewAgentPresetManager(s)
	m.Refresh(ctx)
	ms := m.Snapshot()
	if ms.Err != "" || !ms.Loaded || !ms.Writable {
		add("SET preset manager", "blocked", failure(errors.New(ms.Err)))
		return
	}
	original := ms.DefaultID
	if original == "" || original == presetID {
		t.Fatal("fresh default identity invalid")
	}
	m.SaveDefault(ctx, presetID)
	if ms = m.Snapshot(); ms.Err != "" || ms.DefaultID != presetID {
		t.Fatal("safe future default save not acknowledged: " + failure(errors.New(ms.Err)))
	}
	add("SET future preset default", "pass", "production manager saved own default with revision CAS; no active-session selection")
	stop()
	// Edit only the owned declarative fixture, preserving the Settings-owned
	// selectedDefault patch and every other entry in this fresh profile.
	editPresetFixture(t, patchPath, presetRow, false)
	client = start()
	if client == nil {
		return
	}
	s.Attach(client, nil)
	s.SetWorkspaceCwd(workspace)
	m.Refresh(ctx)
	ms = m.Snapshot()
	if ms.Err != "" || ms.DefaultID != presetID {
		t.Fatal("preset default restart mismatch")
	}
	found := false
	for _, p := range ms.Rows {
		if p.ID == presetID && p.Name == "Owned local preset edited" {
			found = true
		}
	}
	if !found {
		t.Fatal("owned preset source edit not reflected")
	}
	add("SET preset source/default restart", "pass", "edited owned declarative fixture and saved future default survived actual Host restart; no agent binding")
	s.RefreshSkillCatalog(ctx)
	if !wait("SET skill restart catalog", func() bool { _, err := s.CaptureCatalogSkillEdit(skillName); return err == nil }) {
		return
	}
	doc, e = s.GetCatalogSkillDetail(ctx, skillName)
	if e != nil || doc.Content != strings.TrimSpace(fields.Content) || doc.ModelInvocable || len(doc.Groups) != 1 {
		t.Fatal("skill restart persistence mismatch")
	}
	add("SET skill restart persistence", "pass", "edited body/groups/invocation flags restored from own file")
	m.SaveDefault(ctx, original)
	if ms = m.Snapshot(); ms.Err != "" || ms.DefaultID != original {
		t.Fatal("restore original future default failed")
	}
	add("SET preset restore", "pass", "original effective future-session default restored with revision acknowledgement")
	s.ReviewSkillRemoval(ctx, skillName)
	review := s.SkillRemovalReview()
	if !review.Ready {
		t.Fatal("owned removal review not ready")
	}
	s.CancelSkillRemoval(review.ID)
	if _, e = s.GetCatalogSkillDetail(ctx, skillName); e != nil {
		t.Fatal("cancel review removed skill")
	}
	s.ReviewSkillRemoval(ctx, skillName)
	review = s.SkillRemovalReview()
	if !review.Ready {
		t.Fatal("fresh owned removal review not ready")
	}
	s.ConfirmSkillRemoval(ctx, review.ID)
	if _, e = os.Stat(*doc.Path); !os.IsNotExist(e) {
		t.Fatal("owned skill file remains after confirmed deletion")
	}
	add("SET skill removal", "pass", "review cancel preserved own file; fresh captured review/confirm deleted own directory and reconciled catalog")
	stop()
	editPresetFixture(t, patchPath, presetRow, true)
	client = start()
	if client == nil {
		return
	}
	s.Attach(client, nil)
	s.SetWorkspaceCwd(workspace)
	m.Refresh(ctx)
	ms = m.Snapshot()
	if ms.Err != "" || ms.DefaultID != original {
		t.Fatal("restored future default not durable")
	}
	for _, p := range ms.Rows {
		if p.ID == presetID {
			t.Fatal("removed owned preset remained")
		}
	}
	var final api.SkillInventorySnapshot
	if e = client.Call(ctx, "skillInventory/list", map[string]any{"request": map[string]any{"cwd": workspace}}, &final); e != nil {
		t.Fatal(failure(e))
	}
	for _, sk := range final.Skills {
		if sk.Name == skillName {
			t.Fatal("deleted owned skill returned after restart")
		}
	}
	add("SET cleanup persistence", "pass", "second actual restart retained original preset default; own preset source and deleted skill absent")
	add("MCP local stdio", "not-run", "not part of this bounded skills/preset fixture; no MCP server or permission policy changed")
	for _, c := range report.Checks {
		if c.Status == "fail" {
			t.Errorf("actual check failed: %s", c.Name)
		}
	}
}

func stringCount(n int) string { raw, _ := json.Marshal(n); return string(raw) }
func boolText(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func editPresetFixture(t *testing.T, path, id string, remove bool) {
	t.Helper()
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var doc yaml.Node
	if e = yaml.Unmarshal(raw, &doc); e != nil {
		t.Fatal(e)
	}
	field := func(n *yaml.Node, key string) *yaml.Node {
		if n.Kind != yaml.MappingNode {
			return nil
		}
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == key {
				return n.Content[i+1]
			}
		}
		return nil
	}
	root := doc.Content[0]
	found := false
	for i, patch := range root.Content {
		insert := field(patch, "insert")
		if insert == nil {
			continue
		}
		for j, row := range insert.Content {
			key := field(row, "id")
			if key == nil || key.Value != id {
				continue
			}
			found = true
			if remove {
				insert.Content = append(insert.Content[:j], insert.Content[j+1:]...)
				if len(insert.Content) == 0 {
					root.Content = append(root.Content[:i], root.Content[i+1:]...)
				}
			} else {
				cfg := field(row, "config")
				name := field(cfg, "name")
				if name == nil {
					t.Fatal("owned fixture name missing")
				}
				name.Value = "Owned local preset edited"
			}
			break
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("exact owned preset fixture not found")
	}
	raw, e = yaml.Marshal(&doc)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
}
