package assistant

// Opt-in real Host integration. Optional modules are enabled only in this new
// test profile. Objectives, routines, watches, notifications and messaging are
// never enabled or submitted; fresh inputs cannot contain prior queued work.
import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

func TestFunctionalAssistantRuntime(t *testing.T) {
	stage, base, output := os.Getenv("WHALEISLE_ASSISTANT_TEST_STAGE"), os.Getenv("WHALEISLE_ASSISTANT_TEST_ROOT"), os.Getenv("WHALEISLE_ASSISTANT_TEST_REPORT")
	if stage == "" || base == "" || output == "" {
		t.Skip("opt-in: set WHALEISLE_ASSISTANT_TEST_STAGE, ROOT and REPORT")
	}
	type check struct {
		Name  string `json:"name"`
		Pass  bool   `json:"pass"`
		Count int    `json:"count,omitempty"`
	}
	report := struct {
		SchemaVersion                         int      `json:"schemaVersion"`
		Scope                                 string   `json:"scope"`
		OwnedRoot                             string   `json:"ownedRoot"`
		Passed                                bool     `json:"passed"`
		Checks                                []check  `json:"checks"`
		Errors                                []string `json:"errors,omitempty"`
		ModuleDiagnostics                     []string `json:"moduleDiagnostics,omitempty"`
		HostStarts                            int      `json:"hostStarts"`
		OwnedHostStopped                      bool     `json:"ownedHostStopped"`
		OwnedListenerClosed                   bool     `json:"ownedListenerClosed"`
		PromptRoutineNotificationMessageCalls int      `json:"promptRoutineNotificationMessageCalls"`
		NotCovered                            []string `json:"notCovered"`
	}{SchemaVersion: 1, Scope: "real isolated rc.2 Host + production plugin RPC clients and native Whale editor; Dshbot/Whale enabled only in this owned test profile",
		NotCovered: []string{"native UI clicks/rendering", "accounts, devices, external routes and message/notification delivery", "objective/routine/trigger execution", "custom panels and sender-policy writes"}}
	defer func() {
		report.Passed = !t.Failed()
		raw, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			err = os.WriteFile(output, append(raw, '\n'), 0600)
		}
		if err != nil {
			t.Error(err)
		}
	}()
	require := func(err error) {
		t.Helper()
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
			t.Fatal(err)
		}
	}
	assert := func(name string, pass bool, count int) {
		t.Helper()
		report.Checks = append(report.Checks, check{name, pass, count})
		if !pass {
			t.Fatalf("%s failed", name)
		}
	}
	root := os.Getenv("WHALEISLE_ASSISTANT_TEST_READ_OWNED")
	readOwned := root != ""
	var err error
	if readOwned {
		rel, e := filepath.Rel(base, root)
		require(e)
		if filepath.IsAbs(rel) || strings.Contains(rel, string(filepath.Separator)) || !strings.HasPrefix(rel, "assistant-functional-check-") {
			t.Fatal("readback must name this check's owned isolated run")
		}
	} else {
		root, err = os.MkdirTemp(base, "assistant-functional-check-")
		require(err)
	}
	report.OwnedRoot = root
	data := filepath.Join(root, "user-data")
	home := filepath.Join(data, "dsh-home")
	workspace := filepath.Join(root, "workspace")
	require(os.MkdirAll(workspace, 0700))
	vendor := filepath.Join(stage, "vendor")
	harnessRoot := filepath.Join(vendor, "deepseek-harness")
	node := filepath.Join(stage, "node", "node.exe")
	t.Setenv(paths.EnvUserData, data)
	t.Setenv(paths.EnvVendor, vendor)
	t.Setenv("DSH_HOME", home)
	supervisor := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}),
		Launch:    harness.LaunchOptions{NodeBin: node, HarnessRoot: harnessRoot, VendorDir: vendor, Host: "127.0.0.1", Port: 32184, Home: home},
		Prepare:   harness.PrepareOptions{VendorDir: vendor, HarnessRoot: harnessRoot, DshHome: home, ProfileDir: filepath.Join(home, "profiles", "web"), RemoteSet: true, Remote: false, Dshbot: true, Whale: true},
		Workspace: workspace, ReadyTimeout: 80 * time.Second})
	var endpoint harness.Endpoint
	stop := func() {
		supervisor.Stop()
		report.OwnedHostStopped = supervisor.Status().State == harness.Stopped
		if endpoint.Origin != "" {
			u, e := url.Parse(endpoint.Origin)
			if e == nil {
				conn, e := net.DialTimeout("tcp", u.Host, time.Second)
				report.OwnedListenerClosed = e != nil
				if conn != nil {
					conn.Close()
				}
			}
		}
	}
	defer stop()
	ctx, cancel := context.WithTimeout(context.Background(), 110*time.Second)
	defer cancel()
	start := func() {
		t.Helper()
		supervisor.Start()
		report.HostStarts++
		for {
			status := supervisor.Status()
			if status.State == harness.Ready {
				endpoint = status.Endpoint
				return
			}
			if status.State == harness.Failed {
				require(status.Err)
				t.Fatal("Host failed")
			}
			select {
			case <-ctx.Done():
				require(ctx.Err())
			case <-time.After(25 * time.Millisecond):
			}
		}
	}
	client := func() *pluginrpc.Client {
		return &pluginrpc.Client{BaseURL: endpoint.Origin, Cookie: endpoint.Cookie, HTTPClient: &http.Client{Timeout: 15 * time.Second}}
	}
	start()
	c := client()
	settings, whaleErr := c.WhaleSettings(ctx)
	catalog, botErr := c.DshbotBotCatalog(ctx)
	if readOwned {
		require(botErr)
		assert("BOT-02-readback-created-before-receipt-fix", len(catalog.Items) == 1 && catalog.Items[0].ID == "owned-bot-1" && catalog.Items[0].SessionID == "session-owned-bot-1", len(catalog.Items))
		cmd := pluginrpc.DshbotBotCreateCommand{ID: "owned-bot-1", SessionID: "session-owned-bot-1", Expected: pluginrpc.DshbotBotCatalog{Revision: 0}, Draft: pluginrpc.DshbotBotDraft{Name: "Owned Bot 1", Title: "Owned local bot", Description: "Empty objective, no routines or notifications.", ScratchCwd: workspace, Provider: "deepseek-official", Model: "deepseek-v4-pro", MemoryReview: false, Notify: false}}
		_, e := pluginrpc.DshbotCreatedBot(cmd, catalog)
		require(e)
		assert("BOT-02-existing-create-reviewed-values-match-without-retry", true, 1)
		return
	}
	for _, probe := range []struct {
		name string
		err  error
	}{{"BOT-01-whale-actually-served", whaleErr}, {"BOT-02-dshbot-actually-served", botErr}} {
		report.Checks = append(report.Checks, check{probe.name, probe.err == nil, 0})
		if probe.err != nil {
			report.Errors = append(report.Errors, probe.name+": "+probe.err.Error())
			t.Errorf("%s: %v", probe.name, probe.err)
		}
	}
	if whaleErr != nil || botErr != nil {
		for _, line := range supervisor.Logs() {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "volatile") || strings.Contains(lower, "dshbot") || strings.Contains(lower, "dsh-whale") {
				if len(line) > 2048 {
					line = line[:2048]
				}
				report.ModuleDiagnostics = append(report.ModuleDiagnostics, line)
			}
		}
	}
	var savedWhale pluginrpc.WhaleSettings
	if whaleErr == nil {
		var editor WhaleSettingsEditor
		editor.Bind("owned-host-1")
		ticket, ok := editor.BeginLoad("owned-host-1")
		assert("BOT-01-native-editor-load-ticket", ok, 0)
		assert("BOT-01-native-editor-real-load", editor.FinishLoad(ticket, settings, nil) && editor.Snapshot().Loaded, 0)
		editor.Edit("owned-host-1", "name", "Owned Whale")
		editor.Edit("owned-host-1", "title", "Owned user")
		editor.Edit("owned-host-1", "persona", "Owned local persona; no prompt.")
		name, title, persona := "Owned Whale", "Owned user", "Owned local persona; no prompt."
		_, err = c.WhaleUpdateSettings(ctx, pluginrpc.WhaleSettingsPatch{Name: &name, UserTitle: &title, PersonaText: &persona})
		require(err)
		require(c.WhaleReplaceMemory(ctx, "Owned local memory. No delivery."))
		savedWhale, err = c.WhaleSettings(ctx)
		require(err)
		assert("BOT-01-settings-memory-real-roundtrip", savedWhale.Name == name && savedWhale.UserTitle == title && savedWhale.PersonaText == persona && savedWhale.Memory == "Owned local memory. No delivery.", 0)
		// Host change revokes the prior editor ticket without emitting a write.
		ticket, ok = editor.BeginLoad("owned-host-1")
		assert("BOT-01-native-editor-refresh-ticket", ok, 0)
		editor.Bind("owned-host-2")
		assert("BOT-01-native-editor-late-read-revoked", !editor.FinishLoad(ticket, savedWhale, nil) && !editor.Snapshot().Loaded, 0)
	}
	var botIDs []string
	var botSessionIDs []string
	var roomID, duplicateID string
	if botErr == nil {
		assert("BOT-02-fresh-empty-bot-catalog", len(catalog.Items) == 0, len(catalog.Items))
		for i := 1; i <= 2; i++ {
			cmd := pluginrpc.DshbotBotCreateCommand{ID: fmt.Sprintf("%s-bot-%d", filepath.Base(root), i), SessionID: fmt.Sprintf("session-%s-bot-%d", filepath.Base(root), i), Expected: catalog,
				// Actual rc.2 llm-deepseek-api-key PROVIDER and declared DEFAULT_MODELS.
				// resolveCallConfig validates metadata; authentication/transport are
				// deferred to generate(), which this test never invokes.
				Draft: pluginrpc.DshbotBotDraft{Name: fmt.Sprintf("Owned Bot %d", i), Title: "Owned local bot", Description: "Empty objective, no routines or notifications.", ScratchCwd: workspace, Provider: "deepseek-official", Model: "deepseek-v4-pro", MemoryReview: false, Notify: false}}
			created, err := c.DshbotCreateBot(ctx, cmd)
			require(err)
			catalog = created.Catalog
			botIDs = append(botIDs, created.Item.ID)
			botSessionIDs = append(botSessionIDs, created.Item.SessionID)
			assert("BOT-02-real-create-idle-bot", created.Item.ID == cmd.ID && created.Item.SessionID == cmd.SessionID && created.Item.Notify != nil && !*created.Item.Notify, 1)
		}
		contacts, err := c.DshbotContacts(ctx)
		require(err)
		section := pluginrpc.DshbotContactCommand{Expected: contacts, Action: "create", ID: "owned-section", Name: "Owned section"}
		changed, err := c.DshbotChangeContacts(ctx, section)
		require(err)
		contacts = changed.Catalog
		assert("BOT-02-section-create", pluginrpc.DshbotContactApplied(section, contacts), len(contacts.Sections))
		section = pluginrpc.DshbotContactCommand{Expected: contacts, Action: "rename", ID: "owned-section", Name: "Owned renamed section"}
		changed, err = c.DshbotChangeContacts(ctx, section)
		require(err)
		contacts = changed.Catalog
		assert("BOT-02-section-rename", pluginrpc.DshbotContactApplied(section, contacts), len(contacts.Sections))
		section = pluginrpc.DshbotContactCommand{Expected: contacts, Action: "assign", ID: botIDs[0], SessionID: botSessionIDs[0], SectionID: "owned-section"}
		changed, err = c.DshbotChangeContacts(ctx, section)
		require(err)
		contacts = changed.Catalog
		assert("BOT-02-contact-section-assignment", pluginrpc.DshbotContactApplied(section, contacts), 0)
		catalog, err = c.DshbotBotCatalog(ctx)
		require(err)
		visibility, err := c.DshbotSetVisibility(ctx, pluginrpc.DshbotVisibilityCommand{Expected: catalog, ID: botIDs[0], SessionID: botSessionIDs[0], Hidden: true, UpdatedAt: time.Now().UnixMilli()})
		require(err)
		assert("BOT-02-contact-hide", visibility.Item.Hidden, 0)
		visibility, err = c.DshbotSetVisibility(ctx, pluginrpc.DshbotVisibilityCommand{Expected: visibility.Catalog, ID: botIDs[0], SessionID: botSessionIDs[0], Hidden: false, UpdatedAt: time.Now().UnixMilli() + 1})
		require(err)
		catalog = visibility.Catalog
		assert("BOT-02-contact-unhide", !visibility.Item.Hidden, 0)
		duplicate, err := c.DshbotDuplicateBot(ctx, pluginrpc.DshbotBotDuplicateCommand{SourceID: botIDs[0], Name: "Owned duplicate", ScratchCwd: workspace, Expected: catalog})
		require(err)
		duplicateID = duplicate.Item.ID
		assert("BOT-02-real-duplicate-empty-objective", duplicateID != "" && duplicateID != botIDs[0], 1)
		rooms, err := c.DshbotRoomCatalog(ctx)
		require(err)
		createdRoom, err := c.DshbotCreateRoom(ctx, pluginrpc.DshbotRoomCreateCommand{ID: "owned-room", SessionID: "session-owned-room", Expected: rooms,
			Draft: pluginrpc.DshbotRoomDraft{Name: "Owned room", Description: "No messages submitted.", ScratchCwd: workspace, MemberBotIDs: botIDs, MaxRounds: 1, MaxSpeaks: 1}})
		require(err)
		roomID = createdRoom.Item.ID
		assert("BOT-03-real-room-catalog-create", roomID == "owned-room" && len(createdRoom.Item.MemberBotIDs) == 2, 2)
		runtime, err := c.DshbotRoomRuntime(ctx, roomID)
		require(err)
		assert("BOT-03-room-runtime-read-without-dispatch", runtime.Room.ID == roomID && len(runtime.Members) == 0, len(runtime.Members))
		routines, err := c.DshbotRoutines(ctx, botIDs[0])
		require(err)
		tasks, err := c.DshbotTasks(ctx, botIDs[0])
		require(err)
		assert("BOT-01-safe-empty-routine-task-reads", len(routines.Value.Routines) == 0 && len(tasks.Value.Tasks) == 0, 0)
	}
	if whaleErr == nil && botErr == nil {
		stop()
		assert("first-owned-assistant-host-stopped", report.OwnedHostStopped && report.OwnedListenerClosed, 0)
		start()
		c = client()
		read, err := c.WhaleSettings(ctx)
		require(err)
		assert("BOT-01-whale-restart-persistence", read.Name == savedWhale.Name && read.UserTitle == savedWhale.UserTitle && read.PersonaText == savedWhale.PersonaText && read.Memory == savedWhale.Memory && read.SessionID == savedWhale.SessionID, 0)
		catalog, err = c.DshbotBotCatalog(ctx)
		require(err)
		assert("BOT-02-BOT-03-restart-bots-room-persistence", len(catalog.Items) == 4, len(catalog.Items))
		contacts, err := c.DshbotContacts(ctx)
		require(err)
		assert("BOT-02-restart-section-persistence", len(contacts.Sections) == 1 && contacts.Sections[0].Name == "Owned renamed section", len(contacts.Sections))
		runtime, err := c.DshbotRoomRuntime(ctx, roomID)
		require(err)
		assert("BOT-03-restart-room-idle", runtime.Room.ID == roomID && len(runtime.Room.MemberBotIDs) == 2 && len(runtime.Members) == 0, 0)
		plan, err := c.DshbotBotDeletePlan(ctx, duplicateID)
		require(err)
		deleted, err := c.DshbotDeleteBot(ctx, pluginrpc.DshbotBotDeleteCommand{Plan: plan, ConfirmedID: duplicateID, ConfirmedName: plan.Target.Name, Acknowledged: true})
		require(err)
		assert("BOT-02-delete-only-owned-duplicate", deleted.Removed == duplicateID && deleted.PreservedHistory && deleted.PreservedMemory, 1)
		catalog, err = c.DshbotBotCatalog(ctx)
		require(err)
		assert("BOT-02-surviving-bots-room-intact", len(catalog.Items) == 3, len(catalog.Items))
	}
}
