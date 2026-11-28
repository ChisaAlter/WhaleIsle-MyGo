//go:build windows && composerflowacceptance

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/processutil"
	"whaleisle/internal/recovery"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

// This optional scenario uses the existing production main twice, with one
// local model outside both desktop processes. A loopback transport drops real
// Host responses; it never fabricates session events or admission results.
func runSubmissionRecoveryAcceptance() (err error) {
	if phase := os.Getenv("WHALEISLE_RECOVERY_PHASE"); phase != "" {
		return runSubmissionRecoveryDesktop(phase)
	}
	root, output := os.Getenv("WHALEISLE_COMPOSER_QA_WORKSPACE"), os.Getenv("WHALEISLE_COMPOSER_QA_OUT")
	if !isolatedDesktopHidden() || !filepath.IsAbs(root) || !filepath.IsAbs(output) {
		return errors.New("hidden owned recovery environment required")
	}
	model, err := newQAModel()
	if err != nil {
		return err
	}
	defer model.server.Close()
	record := map[string]any{"scope": "two real hidden desktop/Host processes; real transport loss, durable resume and Host journal", "model": "deterministic loopback", "passed": false}
	defer func() {
		record["passed"], record["error"], record["modelRequests"] = err == nil, fmt.Sprint(err), model.requests.Load()
		body, _ := json.MarshalIndent(record, "", "  ")
		if e := os.WriteFile(output, append(body, '\n'), 0600); err == nil {
			err = e
		}
	}()
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	var firstCount int64
	for _, phase := range []string{"submit", "restore"} {
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
		cmd := exec.CommandContext(ctx, exe, "-test.run=^TestComposerFlowAcceptance$", "-test.v", "-test.timeout=140s")
		cmd.Env = append(os.Environ(), "WHALEISLE_RECOVERY_PHASE="+phase, "WHALEISLE_RECOVERY_MODEL=http://"+model.addr+"/anthropic")
		processutil.BackgroundTree(cmd)
		log, openErr := os.Create(output + "." + phase + ".log")
		if openErr != nil {
			cancel()
			return openErr
		}
		cmd.Stdout, cmd.Stderr = log, log
		runErr := cmd.Run()
		cancel()
		_ = log.Close()
		var child map[string]any
		body, readErr := os.ReadFile(output + "." + phase + ".json")
		if readErr == nil {
			readErr = json.Unmarshal(body, &child)
		}
		record[phase] = child
		if runErr != nil {
			return fmt.Errorf("desktop %s: %w", phase, runErr)
		}
		if readErr != nil {
			return readErr
		}
		if child["passed"] != true {
			return fmt.Errorf("desktop %s failed", phase)
		}
		if phase == "submit" {
			firstCount = model.requests.Load()
			record["modelRequestsAfterSubmit"] = firstCount
		} else if model.requests.Load() != firstCount {
			return fmt.Errorf("restart triggered another model request: %d -> %d", firstCount, model.requests.Load())
		}
	}
	record["restartModelRequests"] = model.requests.Load() - firstCount
	return nil
}

type recoveryWireReceipt struct {
	RequestID, SessionID string
	Accepted             bool
}
type recoveryWire struct {
	mu              sync.Mutex
	target          *url.URL
	server          *http.Server
	origin          string
	armed, offline  bool
	connections     map[*websocket.Conn]bool
	receipts        chan recoveryWireReceipt
	promptCalls     int
	transportErrors []string
}

func newRecoveryWire(origin string) (*recoveryWire, error) {
	u, err := url.Parse(origin)
	if err != nil {
		return nil, err
	}
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	w := &recoveryWire{target: u, origin: "http://" + l.Addr().String(), connections: map[*websocket.Conn]bool{}, receipts: make(chan recoveryWireReceipt, 4)}
	proxy := httputil.NewSingleHostReverseProxy(u)
	director := proxy.Director
	proxy.Director = func(r *http.Request) {
		director(r)
		r.Host = u.Host
		if r.Header.Get("Origin") != "" {
			r.Header.Set("Origin", u.Scheme+"://"+u.Host)
		}
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, r *http.Request, err error) {
		http.Error(rw, "owned transport unavailable", http.StatusBadGateway)
	}
	w.server = &http.Server{Handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path == dsh.MuxPath {
			w.socket(rw, r)
			return
		}
		if r.URL.Path == "/api/session/prompt" {
			w.prompt(rw, r)
			return
		}
		proxy.ServeHTTP(rw, r)
	})}
	go w.server.Serve(l)
	return w, nil
}
func (w *recoveryWire) closeSockets() {
	w.mu.Lock()
	var conns []*websocket.Conn
	for c := range w.connections {
		conns = append(conns, c)
	}
	w.mu.Unlock()
	for _, c := range conns {
		_ = c.CloseNow()
	}
}
func (w *recoveryWire) close()   { w.closeSockets(); _ = w.server.Close() }
func (w *recoveryWire) arm()     { w.mu.Lock(); w.armed = true; w.mu.Unlock() }
func (w *recoveryWire) restore() { w.mu.Lock(); w.armed, w.offline = false, false; w.mu.Unlock() }
func (w *recoveryWire) socket(rw http.ResponseWriter, r *http.Request) {
	w.mu.Lock()
	offline := w.offline
	w.mu.Unlock()
	if offline {
		http.Error(rw, "owned link disconnected", 503)
		return
	}
	upURL := *w.target
	upURL.Scheme = "ws"
	upURL.Path = dsh.MuxPath
	up, _, err := websocket.Dial(r.Context(), upURL.String(), &websocket.DialOptions{HTTPHeader: r.Header.Clone()})
	if err != nil {
		w.mu.Lock()
		w.transportErrors = append(w.transportErrors, "upstream websocket: "+err.Error())
		w.mu.Unlock()
		http.Error(rw, "owned upstream socket failed", 502)
		return
	}
	defer up.CloseNow()
	down, err := websocket.Accept(rw, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer down.CloseNow()
	up.SetReadLimit(128 << 20)
	down.SetReadLimit(128 << 20)
	w.mu.Lock()
	w.connections[up], w.connections[down] = true, true
	w.mu.Unlock()
	defer func() { w.mu.Lock(); delete(w.connections, up); delete(w.connections, down); w.mu.Unlock() }()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	done := make(chan struct{}, 2)
	pump := func(from, to *websocket.Conn, downlink bool) {
		defer func() { done <- struct{}{} }()
		for {
			kind, raw, err := from.Read(ctx)
			if err != nil {
				return
			}
			w.mu.Lock()
			drop := downlink && (w.armed || w.offline)
			w.mu.Unlock()
			if drop {
				continue
			}
			if err = to.Write(ctx, kind, raw); err != nil {
				return
			}
		}
	}
	go pump(down, up, false)
	go pump(up, down, true)
	<-done
}
func (w *recoveryWire) prompt(rw http.ResponseWriter, r *http.Request) {
	request, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(rw, "read request", 400)
		return
	}
	var envelope struct {
		Payload struct {
			Args struct {
				Request struct {
					RequestID string `json:"requestId"`
					SessionID string `json:"sessionId"`
				} `json:"request"`
			} `json:"args"`
		} `json:"payload"`
	}
	if json.Unmarshal(request, &envelope) != nil {
		http.Error(rw, "invalid request", 400)
		return
	}
	w.mu.Lock()
	w.promptCalls++
	armed := w.armed
	w.mu.Unlock()
	u := *w.target
	u.Path = r.URL.Path
	req, _ := http.NewRequestWithContext(r.Context(), r.Method, u.String(), bytes.NewReader(request))
	req.Header = r.Header.Clone()
	// The downstream client advertises gzip. Let Go's upstream transport own
	// compression so the receipt observer decodes the same body Client.Call
	// would see, rather than attempting JSON on compressed network bytes.
	req.Header.Del("Accept-Encoding")
	req.Host = w.target.Host
	response, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(rw, "forward failed", 502)
		return
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		http.Error(rw, "response failed", 502)
		return
	}
	var answer struct {
		Result struct {
			OK    bool `json:"ok"`
			Value struct {
				Accepted bool `json:"accepted"`
			} `json:"value"`
		} `json:"result"`
	}
	_ = json.Unmarshal(body, &answer)
	if mediaType, params, parseErr := mime.ParseMediaType(response.Header.Get("Content-Type")); parseErr == nil && strings.HasPrefix(mediaType, "multipart/") {
		reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
		for {
			part, e := reader.NextPart()
			if e != nil {
				break
			}
			raw, e := io.ReadAll(part)
			if e == nil && json.Unmarshal(raw, &answer) == nil && answer.Result.OK {
				break
			}
		}
	}
	if armed {
		w.mu.Lock()
		w.offline = true
		w.mu.Unlock()
		w.closeSockets()
		w.receipts <- recoveryWireReceipt{envelope.Payload.Args.Request.RequestID, envelope.Payload.Args.Request.SessionID, answer.Result.OK && answer.Result.Value.Accepted}
		if h, ok := rw.(http.Hijacker); ok {
			conn, _, e := h.Hijack()
			if e == nil {
				_ = conn.Close()
				return
			}
		}
		panic(http.ErrAbortHandler)
	}
	for key, values := range response.Header {
		for _, v := range values {
			rw.Header().Add(key, v)
		}
	}
	rw.WriteHeader(response.StatusCode)
	_, _ = rw.Write(body)
}

type recoveryExpected struct {
	A, B, FirstRequest, RestartRequest string
}
type recoveryObservation struct {
	Selected, A, B, ErrorA string
	PendingA               int
	Phases                 map[string]string
}

func recoveryObserve(sessions *session.Store, expected recoveryExpected) recoveryObservation {
	v := recoveryObservation{Selected: sessions.SelectedSessionID(), Phases: map[string]string{}}
	mygo.RunOnMain(func() {
		if state.win != nil {
			ui.DriveWindow(state.win).Frame()
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		if d := state.composerDrafts[expected.A]; d != nil {
			v.A, v.ErrorA, v.PendingA = d.Text, d.Error, len(d.PendingSubmissionFlights())
		}
		if d := state.composerDrafts[expected.B]; d != nil {
			v.B = d.Text
		}
		if state.resumeStore != nil {
			var saved session.SubmissionRecovery
			if json.Unmarshal(state.resumeStore.Snapshot().Submissions, &saved) == nil {
				for id, r := range saved.Records {
					v.Phases[id] = r.Phase
				}
			}
		}
	})
	return v
}
func recoveryWait(ctx context.Context, what string, check func() (bool, error)) error {
	for {
		ok, err := check()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s: %w", what, ctx.Err())
		case <-time.After(40 * time.Millisecond):
		}
	}
}

// recoveryJournalCountWait retries the follow-snapshot count because every new
// subscription starts from a fresh durable baseline: admission journaling can
// still be in flight when the desktop first observes ownership, so a single
// snapshot races the write it is supposed to verify.
func recoveryJournalCountWait(ctx context.Context, endpoint harness.Endpoint, sid, id string) (int, error) {
	for {
		count, err := recoveryJournalCount(ctx, endpoint, sid, id)
		if count >= 1 {
			return count, nil
		}
		select {
		case <-ctx.Done():
			return count, err
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func recoveryJournalCount(ctx context.Context, endpoint harness.Endpoint, sid, id string) (int, error) {
	mux := &dsh.Mux{Endpoint: dsh.Endpoint{Origin: endpoint.Origin, Cookie: endpoint.Cookie}}
	defer mux.Close()
	if err := mux.Dial(ctx); err != nil {
		return 0, err
	}
	stream, err := mux.Open(ctx, "session/follow", map[string]any{"args": map[string]any{"request": map[string]any{"address": map[string]any{"kind": "session", "sessionId": sid}, "maxMessages": 200}}})
	if err != nil {
		return 0, err
	}
	defer stream.Cancel()
	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			return 0, err
		}
		var snap struct {
			Type    string `json:"type"`
			Records []struct {
				Type  string `json:"type"`
				Event struct {
					Type string `json:"type"`
					Data struct {
						Source struct {
							RPCID string `json:"rpcId"`
						} `json:"source"`
					} `json:"data"`
				} `json:"event"`
			} `json:"records"`
		}
		if json.Unmarshal(raw, &snap) != nil || snap.Type != "snapshot" {
			continue
		}
		count := 0
		finished := false
		seen := []string{}
		for _, r := range snap.Records {
			if r.Type == "event" && r.Event.Type == "user/message" {
				seen = append(seen, r.Event.Data.Source.RPCID)
			}
			if r.Type == "event" && r.Event.Type == "user/message" && r.Event.Data.Source.RPCID == id {
				count++
				finished = false
			}
			if count > 0 && r.Type == "event" && r.Event.Type == "turn/end" {
				finished = true
			}
		}
		if finished {
			return count, nil
		}
		// Admission can precede the model request. Wait for the real durable
		// turn end before stopping the Host so a restored interrupted turn is
		// not confused with a replayed user submission.
		for {
			raw, err = stream.Recv(ctx)
			if err != nil {
				return count, err
			}
			if bytes.Contains(raw, []byte(`"turn/end"`)) {
				if count == 0 && len(seen) > 0 {
					return count, fmt.Errorf("no user/message with rpcId %s; transcript holds %v", id, seen)
				}
				return count, nil
			}
		}
	}
}

func runSubmissionRecoveryDesktop(phase string) (failure error) {
	data, workspace, output := os.Getenv("WHALEISLE_USER_DATA"), os.Getenv("WHALEISLE_COMPOSER_QA_WORKSPACE"), os.Getenv("WHALEISLE_COMPOSER_QA_OUT")
	if !isolatedDesktopHidden() || !filepath.IsAbs(data) || !filepath.IsAbs(workspace) || !filepath.IsAbs(output) {
		return errors.New("owned hidden recovery paths required")
	}
	modelURL := os.Getenv("WHALEISLE_RECOVERY_MODEL")
	if !strings.HasPrefix(modelURL, "http://127.0.0.1:") {
		return errors.New("owned loopback model required")
	}
	_ = os.Setenv("DEEPSEEK_BASE_URL", modelURL)
	_ = os.Setenv("DEEPSEEK_API_KEY", "qa-loopback-only")
	for _, key := range []string{"WHALEISLE_API_KEY", "DSH_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY"} {
		_ = os.Unsetenv(key)
	}
	if err := os.MkdirAll(workspace, 0700); err != nil {
		return err
	}
	cfg, err := config.Open(data)
	if err != nil {
		return err
	}
	if err = cfg.Update(func(c *config.Config) {
		c.Workspace = workspace
		c.Host = "127.0.0.1"
		c.NodeBin = os.Getenv("WHALEISLE_NODE")
		c.AutoDownloadUpdates = false
		c.AutoStartDesktop = true
		c.RemoteEnabled = false
		c.HarnessAutoRestart = false
		c.CloseToTray = false
	}); err != nil {
		return err
	}
	_ = os.WriteFile(filepath.Join(data, "welcome-shown"), nil, 0600)
	expectedFile := filepath.Join(workspace, "recovery-expected.json")
	var expected recoveryExpected
	if phase == "restore" {
		body, e := os.ReadFile(expectedFile)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(body, &expected); e != nil {
			return e
		}
	}
	report := map[string]any{"phase": phase, "pid": os.Getpid(), "hidden": true, "passed": false}
	result := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 110*time.Second)
		defer cancel()
		var wire *recoveryWire
		finish := func(err error) {
			state.mu.Lock()
			report["connection"] = map[string]any{"ready": state.dshMuxReady, "lost": state.dshMuxLost, "bootstrapError": state.dshBootstrapError, "resumeError": state.resumeError}
			if state.sessions != nil {
				v := state.sessions.Snapshot()
				report["lastSession"] = map[string]any{"clientReady": v.ClientReady, "workspaceReady": v.WorkspaceReady, "listLoading": v.ListLoading, "listError": v.ListErr, "lastAction": v.LastAction}
			}
			state.mu.Unlock()
			if wire != nil {
				wire.mu.Lock()
				report["wireErrors"] = append([]string(nil), wire.transportErrors...)
				wire.mu.Unlock()
			}
			state.stopDSH()
			state.mu.Lock()
			sup := state.sup
			state.mu.Unlock()
			if sup != nil {
				sup.Stop()
			}
			if wire != nil {
				wire.close()
			}
			report["passed"], report["error"] = err == nil, fmt.Sprint(err)
			body, _ := json.MarshalIndent(report, "", "  ")
			_ = os.WriteFile(output+"."+phase+".json", append(body, '\n'), 0600)
			state.mu.Lock()
			state.quitting = true
			state.quitCleanupFinished = true
			state.mu.Unlock()
			result <- err
			mygo.App.Quit()
		}
		var sessions *session.Store
		var win *mygo.Window
		var status harness.Status
		ready := func() (bool, error) {
			state.mu.Lock()
			sessions, win, status = state.sessions, state.win, state.bootStatus
			ready := state.dshMuxReady
			state.mu.Unlock()
			if status.State == harness.Failed {
				return false, fmt.Errorf("Host failed: %v", status.Err)
			}
			return win != nil && sessions != nil && ready && sessions.Snapshot().WorkspaceReady && !sessions.Snapshot().ListLoading, nil
		}
		if err := recoveryWait(ctx, "desktop workspace ready", ready); err != nil {
			finish(err)
			return
		}
		if phase == "restore" {
			if err := recoveryWait(ctx, "restored receipt and drafts", func() (bool, error) {
				v := recoveryObserve(sessions, expected)
				report["afterRestart"] = v
				return v.Selected == expected.B && v.A == "new draft after restart send" && v.B == "keep B unsent" && v.PendingA == 0 && v.Phases[expected.RestartRequest] == "confirmed", nil
			}); err != nil {
				finish(err)
				return
			}
			count, err := recoveryJournalCountWait(ctx, status.Endpoint, expected.A, expected.RestartRequest)
			report["restartJournalCount"] = count
			if err != nil || count != 1 {
				finish(fmt.Errorf("restart journal count=%d: %v", count, err))
				return
			}
			if err = state.saveResume(); err != nil {
				finish(err)
				return
			}
			finish(nil)
			return
		}
		wire, err = newRecoveryWire(status.Endpoint.Origin)
		if err != nil {
			finish(err)
			return
		}
		proxied := status
		proxied.Endpoint.Origin = wire.origin
		state.syncDialogue(proxied)
		proxyCtx, proxyCancel := context.WithTimeout(ctx, 20*time.Second)
		err = recoveryWait(proxyCtx, "proxied production connection", ready)
		proxyCancel()
		if err != nil {
			finish(err)
			return
		}
		create := func() (string, error) {
			before := sessions.SelectedSessionID()
			mygo.RunOnMain(func() { sessions.CreateSession(ctx) })
			e := recoveryWait(ctx, "create recovery session", func() (bool, error) {
				snap := sessions.Snapshot()
				sel := sessions.SelectedSessionID()
				// Blank-reuse is a correct create outcome: the gesture over an
				// untouched blank session adopts it instead of proliferating
				// empty identities, so the selected id legitimately stays put.
				reused := sel == before && snap.LastAction == "blank session reused"
				return sel != "" && (sel != before || reused) && snap.HistoryReady, nil
			})
			return sessions.SelectedSessionID(), e
		}
		if expected.A, err = create(); err != nil {
			finish(err)
			return
		}
		// A second create gesture while A is still blank would correctly reuse
		// A again and collapse the two fixture identities into one. Send one
		// real loopback message first so the Host no longer considers A blank.
		// The composer drops hero-state submits until the workspace evidence
		// settles, and registration lands asynchronously after boot. Gate on
		// the exact predicate the submit path evaluates: RequiresWorkspace
		// also exempts a session whose cwd sits under a registered workspace
		// root, so roster row existence alone is sufficient evidence.
		if err = recoveryWait(ctx, "workspace evidence admits A", func() (bool, error) {
			snap := sessions.Snapshot()
			return snap.WorkspaceReady && !composer.RequiresWorkspace(snap, true), nil
		}); err != nil {
			finish(err)
			return
		}
		mygo.RunOnMain(func() {
			state.mu.Lock()
			draft := state.composerDraftLocked(expected.A)
			draft.Text = "QA_SEED_A"
			state.mu.Unlock()
			state.submitComposer(win, sessions, draft, expected.A, workspace, false, sessions.Snapshot().ComposerHostGeneration)
		})
		seedOK := func() (bool, string) {
			v := recoveryObserve(sessions, expected)
			snap := sessions.Snapshot()
			// The persisted transcript is the direct evidence the Host owns a
			// user message in A; once it does, blankReuse cannot fire for the
			// next create gesture regardless of list-summary refresh timing.
			userCount, replyCount := 0, 0
			if snap.SelectedID == expected.A {
				for _, m := range snap.Messages {
					if m.Role == "user" && strings.Contains(m.Body, "QA_SEED_A") {
						userCount++
					}
					if m.Role == "assistant" && strings.Contains(m.Body, "QA_REPLY_OK") {
						replyCount++
					}
				}
			}
			detail := fmt.Sprintf("pending=%d err=%q user=%d reply=%d busy=%v sel=%q wantA=%q", v.PendingA, v.ErrorA, userCount, replyCount, snap.PromptBusy, snap.SelectedID, expected.A)
			return v.PendingA == 0 && v.ErrorA == "" && userCount >= 1 && replyCount >= 1 && !snap.PromptBusy, detail
		}
		var seedDetail string
		if err = recoveryWait(ctx, "seed message confirmed in A", func() (bool, error) {
			ok, detail := seedOK()
			seedDetail = detail
			return ok, nil
		}); err != nil {
			finish(fmt.Errorf("%w (%s)", err, seedDetail))
			return
		}
		if expected.B, err = create(); err != nil {
			finish(err)
			return
		}
		mygo.RunOnMain(func() {
			state.mu.Lock()
			state.composerDraftLocked(expected.B).Text = "keep B unsent"
			state.mu.Unlock()
		})
		sendLost := func(text, newDraft string) (string, error) {
			sessions.Select(expected.A)
			if e := recoveryWait(ctx, "select recovery target", func() (bool, error) {
				return sessions.Snapshot().SelectedID == expected.A && sessions.Snapshot().HistoryReady, nil
			}); e != nil {
				return "", e
			}
			wire.arm()
			mygo.RunOnMain(func() {
				state.mu.Lock()
				draft := state.composerDraftLocked(expected.A)
				draft.Text = text
				state.mu.Unlock()
				state.submitComposer(win, sessions, draft, expected.A, workspace, false, sessions.Snapshot().ComposerHostGeneration)
				sessions.Select(expected.B)
				state.mu.Lock()
				draft.Text = newDraft
				state.mu.Unlock()
			})
			var receipt recoveryWireReceipt
			select {
			case receipt = <-wire.receipts:
			case <-ctx.Done():
				return "", ctx.Err()
			}
			if !receipt.Accepted || receipt.RequestID == "" || receipt.SessionID != expected.A {
				return "", fmt.Errorf("Host did not accept original request: %+v", receipt)
			}
			e := recoveryWait(ctx, "unconfirmed request remains owned", func() (bool, error) {
				v := recoveryObserve(sessions, expected)
				return v.Selected == expected.B && v.PendingA == 1 && v.Phases[receipt.RequestID] == "dispatching" && strings.Contains(v.ErrorA, "待确认"), nil
			})
			return receipt.RequestID, e
		}
		if expected.FirstRequest, err = sendLost("QA_RECOVERY_RECONNECT", "new draft during reconnect"); err != nil {
			finish(err)
			return
		}
		report["beforeReconnect"] = recoveryObserve(sessions, expected)
		count, err := recoveryJournalCountWait(ctx, status.Endpoint, expected.A, expected.FirstRequest)
		report["acceptedBeforeReconnectJournalCount"] = count
		if err != nil || count != 1 {
			finish(fmt.Errorf("accepted journal count=%d: %v", count, err))
			return
		}
		wire.restore()
		state.requestConnectionReconnect()
		if err = recoveryWait(ctx, "same request confirmed after reconnect", func() (bool, error) {
			v := recoveryObserve(sessions, expected)
			report["afterReconnect"] = v
			return v.PendingA == 0 && v.A == "new draft during reconnect" && v.B == "keep B unsent" && v.Selected == expected.B && v.Phases[expected.FirstRequest] == "confirmed", nil
		}); err != nil {
			finish(err)
			return
		}
		if expected.RestartRequest, err = sendLost("QA_RECOVERY_RESTART", "new draft after restart send"); err != nil {
			finish(err)
			return
		}
		report["beforeRestart"] = recoveryObserve(sessions, expected)
		count, err = recoveryJournalCount(ctx, status.Endpoint, expected.A, expected.RestartRequest)
		report["acceptedBeforeRestartJournalCount"] = count
		if err != nil || count != 1 {
			finish(fmt.Errorf("restart accepted journal count=%d: %v", count, err))
			return
		}
		if err = state.saveResume(); err != nil {
			finish(err)
			return
		}
		body, _ := json.Marshal(expected)
		if err = os.WriteFile(expectedFile, body, 0600); err != nil {
			finish(err)
			return
		}
		// Read the actual durable store before quitting; the restored process
		// must independently reconcile it against the Host's persisted journal.
		saved, err := recovery.Open(data)
		if err != nil {
			finish(err)
			return
		}
		report["savedSelected"] = saved.Snapshot().SelectedSession
		wire.mu.Lock()
		calls := wire.promptCalls
		wire.mu.Unlock()
		report["promptDispatches"] = calls
		if calls != 3 {
			finish(fmt.Errorf("prompt replay before restart: %d", calls))
			return
		}
		finish(nil)
	}()
	main()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		return errors.New("desktop exited without recovery evidence")
	}
}
