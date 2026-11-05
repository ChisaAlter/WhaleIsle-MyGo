package main

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

//go:embed real_guard.cjs
var realGuardSource []byte

type realModelReport struct {
	Mode             string `json:"mode"`
	Model            string `json:"model"`
	NetworkCalls     int    `json:"networkCalls"`
	SettledCalls     int    `json:"settledCalls"`
	PreflightValid   int    `json:"preflightValidated"`
	ChargedMicroCNY  int64  `json:"chargedMicroCNY"`
	ReservedMicroCNY int64  `json:"reservedMicroCNY"`
	GuardAudit       string `json:"guardAudit"`
	Ledger           string `json:"ledger,omitempty"`
	TurnsCompleted   int    `json:"turnsCompleted"`
	Boundary         string `json:"boundary"`
}

type realFlowFailure string

func (e realFlowFailure) Error() string { return string(e) }
func safeRealFailure(err error) string {
	var controlled realFlowFailure
	if errors.As(err, &controlled) {
		return string(controlled)
	}
	return "operation-stopped-without-recording-provider-or-user-content"
}

func validateRealMode(secret, ledger string, real, preflight bool) error {
	if real == preflight {
		return realFlowFailure("choose-exactly-one-real-model-mode")
	}
	if preflight {
		if secret != "" || ledger != "" {
			return realFlowFailure("preflight-forbids-secret-and-ledger")
		}
	} else if !filepath.IsAbs(secret) || !filepath.IsAbs(ledger) || secret == ledger {
		return realFlowFailure("explicit-distinct-absolute-secret-file-and-ledger-required")
	}
	return nil
}

// Only OS bootstrap values are carried into this one-shot check. In particular
// no existing API key, proxy, NODE_OPTIONS, user npm config or Host setting is read.
func isolateRealEnvironment(root, stage, home string) error {
	env := map[string]string{}
	for _, key := range []string{"SystemRoot", "WINDIR", "PATH", "PATHEXT", "ComSpec", "NUMBER_OF_PROCESSORS", "PROCESSOR_ARCHITECTURE"} {
		if value := os.Getenv(key); value != "" {
			env[key] = value
		}
	}
	os.Clearenv()
	for key, value := range env {
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	owned := filepath.Join(root, "owned-home")
	for key, value := range map[string]string{
		paths.EnvUserData: filepath.Join(root, "user-data"), paths.EnvVendor: filepath.Join(stage, "vendor"),
		"DSH_HOME": home, "HOME": owned, "USERPROFILE": owned,
		"APPDATA": filepath.Join(owned, "AppData", "Roaming"), "LOCALAPPDATA": filepath.Join(owned, "AppData", "Local"),
		"TEMP": filepath.Join(root, "temp"), "TMP": filepath.Join(root, "temp"),
		"DEEPSEEK_API_KEY": "qa-real-flow-no-secret", "DEEPSEEK_BASE_URL": "https://api.deepseek.com/anthropic",
		"GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": os.DevNull,
	} {
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return nil
}

func runRealModelFlow(stage, base, secret, ledger string, real, preflight bool, r *report) error {
	if err := validateRealMode(secret, ledger, real, preflight); err != nil {
		return err
	}
	stage, err := filepath.Abs(stage)
	if err != nil {
		return err
	}
	base, err = filepath.Abs(base)
	if err != nil {
		return err
	}
	// A completed/partial previous batch is never silently treated as a new two-
	// request run. The Node ledger independently validates and locks the total.
	if real {
		if _, err := os.Lstat(ledger + ".lock"); err == nil || !os.IsNotExist(err) {
			return realFlowFailure("ledger-locked-or-unreadable-no-automatic-recovery")
		}
		if raw, err := os.ReadFile(ledger); err == nil {
			var v struct {
				Attempts []json.RawMessage `json:"attempts"`
			}
			if json.Unmarshal(raw, &v) != nil || len(v.Attempts) != 0 {
				return realFlowFailure("existing-ledger-has-attempts-or-is-invalid-no-batch-replay")
			}
		} else if !os.IsNotExist(err) {
			return realFlowFailure("ledger-unreadable")
		}
	}
	root, err := os.MkdirTemp(base, "core-host-real-")
	if err != nil {
		return err
	}
	r.OwnedRunRoot = root
	mode := "official-two-turns"
	if preflight {
		mode = "final-payload-preflight-no-network"
	}
	r.RealModel = &realModelReport{Mode: mode, Model: "deepseek-official/deepseek-flash", GuardAudit: filepath.Join(root, "guard-audit.jsonl"), Ledger: ledger,
		Boundary: "Node final-payload allowlist and single HTTPS POST; other external sockets, DNS, subprocesses and workers rejected"}
	home := filepath.Join(root, "user-data", "dsh-home")
	workspace := filepath.Join(root, "workspace")
	for _, dir := range []string{home, workspace, filepath.Join(root, "owned-home"), filepath.Join(root, "temp")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	if err := isolateRealEnvironment(root, stage, home); err != nil {
		return err
	}
	gitCtx, gitCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer gitCancel()
	if err := exec.CommandContext(gitCtx, "git", "init", "--quiet", "--template=", workspace).Run(); err != nil {
		return realFlowFailure("owned-git-init-failed")
	}
	actual, err := exec.CommandContext(gitCtx, "git", "-C", workspace, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return realFlowFailure("owned-git-verification-failed")
	}
	if rel, err := filepath.Rel(workspace, strings.TrimSpace(string(actual))); err != nil || rel != "." {
		return realFlowFailure("owned-git-root-mismatch")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	guardFile := filepath.Join(root, "final-wire-guard.cjs")
	configFile := filepath.Join(root, "guard-config.json")
	config, _ := json.Marshal(map[string]any{"preflight": preflight, "secretFile": secret, "ledgerFile": ledger, "auditFile": r.RealModel.GuardAudit, "hostPort": port})
	if err := os.WriteFile(guardFile, realGuardSource, 0600); err != nil {
		return err
	}
	if err := os.WriteFile(configFile, config, 0600); err != nil {
		return err
	}
	patchFile := filepath.Join(root, "controlled-real-flow.patch.yml")
	if err := os.WriteFile(patchFile, []byte(realFlowPatch), 0600); err != nil {
		return err
	}
	quoted, _ := json.Marshal(guardFile)
	// Applied to prepare-time Node processes as well: a preparation cannot make
	// an unguarded auxiliary request. The real key is not in this environment.
	if err := os.Setenv("NODE_OPTIONS", "--require="+string(quoted)); err != nil {
		return err
	}
	if err := os.Setenv("WHALE_REAL_GUARD_CONFIG", configFile); err != nil {
		return err
	}
	vendor := filepath.Join(stage, "vendor")
	harnessRoot := filepath.Join(vendor, "deepseek-harness")
	node := filepath.Join(stage, "node", "node.exe")
	sup := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}),
		Launch:  harness.LaunchOptions{NodeBin: node, HarnessRoot: harnessRoot, VendorDir: vendor, Host: "127.0.0.1", Port: port, Home: home, PatchFiles: []string{patchFile}},
		Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: harnessRoot, DshHome: home, ProfileDir: filepath.Join(home, "profiles", "web"), RemoteSet: true, Remote: false}, Workspace: workspace, ReadyTimeout: 3 * time.Minute})
	var endpoint harness.Endpoint
	defer func() {
		sup.Stop()
		r.OwnedHostStopped = sup.Status().State == harness.Stopped
		if endpoint.Origin != "" {
			r.OwnedListenerClosed = closedListener(endpoint.Origin)
		}
		readRealAudit(r)
		_ = os.WriteFile(filepath.Join(root, "host.log"), []byte(strings.Join(sup.Logs(), "\n")), 0600)
	}()
	sup.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 210*time.Second)
	defer cancel()
	if err := await(ctx, func() (bool, error) {
		status := sup.Status()
		if status.State == harness.Failed {
			return false, realFlowFailure("guarded-host-start-failed-no-log-content-recorded")
		}
		if status.State == harness.Ready {
			endpoint = status.Endpoint
			return true, nil
		}
		return false, nil
	}); err != nil {
		return err
	}
	st, mux, client := attach(endpoint)
	defer mux.Close()
	defer st.Detach()
	eventCtx, eventCancel := context.WithCancel(ctx)
	defer eventCancel()
	owner := st.BeginEvents(eventCtx, client)
	ready := make(chan struct{}, 1)
	done := make(chan error, 1)
	go func() {
		done <- client.FollowEvents(eventCtx, mux, dsh.EventsFuncs{Ready: func(dsh.EventsReady) {
			st.OnEventsReady(owner)
			select {
			case ready <- struct{}{}:
			default:
			}
		}, Emit: func(e dsh.EventsEmit) { st.OnEmit(owner, e) }, Waterfall: func(dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
			return nil, realFlowFailure("unexpected-approval-never-auto-approved")
		}})
	}()
	select {
	case <-ready:
	case <-done:
		return realFlowFailure("events-ended-before-ready")
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() {
		eventCancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
		}
	}()
	id, err := st.CreateComposerSession(ctx, workspace)
	if err != nil {
		return realFlowFailure("synthetic-session-create-failed")
	}
	st.Select(id)
	if err := await(ctx, func() (bool, error) {
		v := st.Snapshot()
		if v.HistoryError != "" {
			return false, realFlowFailure("history-follow-failed")
		}
		return v.HistoryReady && !v.HistoryLoading, nil
	}); err != nil {
		return err
	}
	prompts := []string{"Reply with the plain text QA_ONE_OK only. Do not use tools or search.", "Reply with the plain text QA_TWO_OK only. Do not use tools or search."}
	if preflight {
		prompts = prompts[:1]
	}
	for index, prompt := range prompts {
		before := st.Snapshot()
		lastSeq := 0.0
		for _, m := range before.Messages {
			if m.Seq > lastSeq {
				lastSeq = m.Seq
			}
		}
		if err := st.SubmitComposer(ctx, id, prompt, nil, 0); err != nil {
			return realFlowFailure("composer-submit-unconfirmed-no-replay")
		}
		turnCtx, turnCancel := context.WithTimeout(ctx, 80*time.Second)
		err := await(turnCtx, func() (bool, error) {
			v := st.Snapshot()
			if v.HistoryError != "" {
				return false, realFlowFailure("turn-history-error-no-replay")
			}
			if preflight {
				return hasRealAudit(r.RealModel.GuardAudit, "preflight-validated", 1) && !v.PromptBusy && v.StopTurnSeq == 0, nil
			}
			if !hasRealAudit(r.RealModel.GuardAudit, "settled", index+1) {
				return false, nil
			}
			for _, m := range v.Messages {
				if m.Seq > lastSeq && m.Role == "assistant" && !m.Streaming && m.Body != "" && !v.PromptBusy && v.StopTurnSeq == 0 {
					return true, nil
				}
			}
			return false, nil
		})
		turnCancel()
		if err != nil {
			st.CancelRunning(context.Background())
			return realFlowFailure("turn-not-confirmed-no-retry-or-replay")
		}
		if !preflight {
			r.RealModel.TurnsCompleted++
		}
	}
	// Expected: the Host probes one helper node.exe at startup and the boundary
	// refuses it; no child ever runs. Any other rejection code or target fails.
	expectedRejected := 0
	for _, row := range guardAuditRows(r.RealModel.GuardAudit) {
		if row["kind"] != "rejected" {
			continue
		}
		if row["code"] == "BOUNDARY_EGRESS_SUBPROCESS" && row["target"] == "node.exe" {
			expectedRejected++
			continue
		}
		return realFlowFailure("unexpected-guard-rejection-inspect-fixed-audit-code")
	}
	if expectedRejected > 3 {
		return realFlowFailure("unexpected-guard-rejection-inspect-fixed-audit-code")
	}
	if preflight {
		r.Checks = append(r.Checks, check{Name: "real-final-payload-preflight", Pass: true, Detail: "Actual Host payload validated then rejected before secret access, reservation and external network; not a real-model pass"})
	} else {
		r.Checks = append(r.Checks, check{Name: "official-synthetic-two-turns", Pass: true, Count: 2, Detail: "Production composer, Host SSE and durable Store terminal state; no UI acceptance"})
	}
	return nil
}

func guardAuditRows(file string) []map[string]any {
	f, err := os.Open(file)
	if err != nil {
		return nil
	}
	defer f.Close()
	var rows []map[string]any
	s := bufio.NewScanner(f)
	for s.Scan() {
		var row map[string]any
		if json.Unmarshal(s.Bytes(), &row) == nil {
			rows = append(rows, row)
		}
	}
	return rows
}
func hasRealAudit(file, kind string, number int) bool {
	for _, row := range guardAuditRows(file) {
		if row["kind"] == kind && (kind == "preflight-validated" || row["number"] == float64(number)) {
			return true
		}
	}
	return false
}
func readRealAudit(r *report) {
	if r.RealModel == nil {
		return
	}
	for _, row := range guardAuditRows(r.RealModel.GuardAudit) {
		switch row["kind"] {
		case "preflight-validated":
			r.RealModel.PreflightValid++
		case "reserved":
			r.RealModel.NetworkCalls++
			r.RealModel.ReservedMicroCNY += 69632
		case "settled":
			r.RealModel.SettledCalls++
			r.RealModel.ReservedMicroCNY -= 69632
			if n, ok := row["chargedMicroCNY"].(float64); ok {
				r.RealModel.ChargedMicroCNY += int64(n)
			}
		}
	}
	r.ModelRequests = r.RealModel.NetworkCalls
}

const realFlowPatch = `- id: llm-deepseek
  config:
    baseURL: https://api.deepseek.com/anthropic
    apiKeyEnv: DEEPSEEK_API_KEY
    thinking: disabled
    reasoningEffort: off
    maxTokens: 512
    models:
      - id: deepseek-flash
        maxTokens: 512
        inputModalities: [text]
    retryPolicy:
      mode: normal
      maxRetries: 0
- id: agent-default-model
  config:
    provider: deepseek-official
    model: deepseek-flash
    reasoningEffort: off
- id: session-title-llm
  disabled: true
- id: session-log-deepseek
  disabled: true
- id: plugin-package-inventory-deepseek
  disabled: true
- id: web-search-deepseek
  disabled: true
- id: web-deepseek-api
  disabled: true
- id: tool-web
  disabled: true
- id: preset-standard
  config:
    id: standard
    order: 1
    plugins: []
`
