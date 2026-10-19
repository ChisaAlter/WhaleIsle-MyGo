//go:build windows && nativeacceptance

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image/png"
	"math"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/taskcontrol"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/store/session"
)

// This optional mode uses the existing nativeacceptance TestMain. It drives
// only the actual production callback, never OS input or an alternate Mux.
func TestNativeReconnectOwnedDesktop(t *testing.T) {
	if os.Getenv("WHALEISLE_ACCEPTANCE_RECONNECT_OUTPUT") == "" {
		t.Skip("explicit owned reconnect acceptance required")
	}
	if nativeAcceptanceError != nil {
		t.Fatal(nativeAcceptanceError)
	}
}

type nativeReconnectObservation struct {
	win                          *mygo.Window
	client                       *dsh.Client
	mux                          *dsh.Mux
	sessions                     *session.Store
	interactions                 *interaction.Store
	supervisor                   *harness.Supervisor
	ready, hostReady, hostFailed bool
	hostGeneration               int
	origin                       string
	snapshot                     session.ViewState
}

func observeNativeReconnect() nativeReconnectObservation {
	state.dshMu.Lock()
	state.mu.Lock()
	v := nativeReconnectObservation{win: state.win, client: state.dshClient, mux: state.dshMux, sessions: state.sessions, interactions: state.interactions, supervisor: state.sup, ready: state.dshMuxReady}
	state.mu.Unlock()
	state.dshMu.Unlock()
	if v.supervisor != nil {
		status := v.supervisor.Status()
		v.hostReady = status.State == harness.Ready
		v.hostFailed = status.State == harness.Failed
		v.hostGeneration = status.Generation
		v.origin = status.Endpoint.Origin
	}
	if v.sessions != nil {
		v.snapshot = v.sessions.Snapshot()
	}
	return v
}
func (v nativeReconnectObservation) usable() bool {
	g := v.snapshot.NativeContributions
	return v.win != nil && v.client != nil && v.mux != nil && v.sessions != nil && v.interactions != nil && v.supervisor != nil && v.hostReady && v.ready && v.snapshot.ClientReady && !v.snapshot.ListLoading && v.snapshot.ListErr == "" && v.snapshot.WorkspaceReady && g.Known && g.Error == "" && g.FileCommand && g.ModelCommand && g.PermissionCommand && g.FeedbackCommand && g.ModelLabel1017
}
func waitNativeReconnect(ctx context.Context, predicate func(nativeReconnectObservation) bool) (nativeReconnectObservation, error) {
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		v := observeNativeReconnect()
		if v.hostFailed {
			return v, errors.New("owned desktop Host failed before reconnect acceptance")
		}
		if predicate(v) {
			return v, nil
		}
		select {
		case <-ctx.Done():
			return v, errors.New("owned desktop reconnect readiness timeout")
		case <-tick.C:
		}
	}
}
func nativeReconnectFacts(v nativeReconnectObservation) map[string]any {
	g := v.snapshot.NativeContributions
	return map[string]any{"muxReady": v.ready, "clientReady": v.snapshot.ClientReady, "sessionListReady": !v.snapshot.ListLoading && v.snapshot.ListErr == "", "workspaceReady": v.snapshot.WorkspaceReady, "hostReady": v.hostReady, "supervisorGeneration": v.hostGeneration, "origin": v.origin, "composerHostGeneration": v.snapshot.ComposerHostGeneration, "sessionGeneration": v.snapshot.ComposerSessionGeneration, "sessionID": v.snapshot.SelectedID, "registrationGeneration": v.snapshot.NativeContributionsGeneration, "permissionCatalogRevision": v.snapshot.PermissionCatalogRevision, "nativeGraphKnown": g.Known, "nativeFlags": map[string]bool{"file": g.FileCommand, "model": g.ModelCommand, "permission": g.PermissionCommand, "feedback": g.FeedbackCommand, "modelLabel1017": g.ModelLabel1017}}
}

func runNativeReconnectDesktop(output string) error {
	if os.Getenv("WHALEISLE_ISOLATED") != "1" || !filepath.IsAbs(os.Getenv(paths.EnvUserData)) || !filepath.IsAbs(os.Getenv(paths.EnvVendor)) || !filepath.IsAbs(output) {
		return errors.New("explicit hidden isolation, absolute owned data/vendor/output required")
	}
	if os.Getenv("WHALEISLE_ACCEPTANCE_OUTPUT") != "" {
		return errors.New("reconnect acceptance and existing core acceptance are mutually exclusive")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0700); err != nil {
		return err
	}
	receipt := map[string]any{"startedUTC": time.Now().UTC().Format(time.RFC3339Nano), "appPID": os.Getpid(), "scope": "Real production main -> authenticated bootstrap -> EventsReady -> driver invokes existing production action callback -> new EventsReady; no physical reconnect/menu gesture; owned hidden window software Content scene", "physicalInput": false, "originalSameStateVisualComparison": false, "startupModelAndBillingMetadataReads": "ordinary production startup; not suppressed or claimed absent", "promptOrCompletionSubmittedByDriver": false, "accountOrDeviceActionByDriver": false, "hostPIDDirectlyObserved": false}
	mainReturned := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		ownedOrigin := ""
		finish := func(err error) {
			state.mu.Lock()
			state.quitting = true
			supervisor := state.sup
			state.mu.Unlock()
			stopped, listenerClosed := false, false
			if supervisor != nil {
				origin := ownedOrigin
				if origin == "" {
					origin = supervisor.Status().Endpoint.Origin
				}
				if supervisor.Status().State != harness.Stopped {
					supervisor.Stop()
				}
				stopped = supervisor.Status().State == harness.Stopped
				if endpoint, e := url.Parse(origin); e == nil && endpoint.User == nil && endpoint.Port() != "" && (strings.EqualFold(endpoint.Hostname(), "localhost") || net.ParseIP(endpoint.Hostname()).IsLoopback()) {
					connection, dialErr := net.DialTimeout("tcp", endpoint.Host, time.Second)
					listenerClosed = dialErr != nil
					if connection != nil {
						connection.Close()
					}
				}
			}
			receipt["ownedHostStopped"] = stopped
			receipt["ownedListenerClosed"] = listenerClosed
			if !stopped || !listenerClosed {
				receipt["protectedGracefulQuitCertified"] = false
			}
			if _, ok := receipt["protectedGracefulQuitCertified"]; !ok {
				receipt["protectedGracefulQuitCertified"] = false
			}
			if err == nil && (!stopped || !listenerClosed) {
				err = errors.New("owned desktop Host cleanup did not complete")
			}
			receipt["finishedUTC"] = time.Now().UTC().Format(time.RFC3339Nano)
			receipt["passed"] = err == nil
			if err != nil {
				receipt["error"] = err.Error()
			}
			raw, writeErr := json.MarshalIndent(receipt, "", "  ")
			if writeErr == nil {
				writeErr = os.WriteFile(output+".json", raw, 0600)
			}
			if err == nil {
				err = writeErr
			}
			result <- err
			select {
			case <-mainReturned:
			default:
				mygo.App.Quit()
			}
		}
		before, err := waitNativeReconnect(ctx, func(v nativeReconnectObservation) bool { return v.usable() })
		if err != nil {
			finish(err)
			return
		}
		ownedOrigin = before.origin
		baselineImage, err := before.win.CapturePage()
		if err != nil {
			finish(errors.New("owned initial Content capture failed"))
			return
		}
		baseline, err := png.DecodeConfig(bytes.NewReader(baselineImage))
		width, height := before.win.ContentSize()
		if err != nil || baseline.Width == 0 || baseline.Height == 0 || width <= 0 || height <= 0 {
			finish(errors.New("owned initial Content scene dimensions unavailable"))
			return
		}
		receipt["baselineScenePixels"] = []int{baseline.Width, baseline.Height}
		receipt["initialContentSizeDIP"] = []int{width, height}
		receipt["before"] = nativeReconnectFacts(before)
		receipt["firstReadyUTC"] = time.Now().UTC().Format(time.RFC3339Nano)
		requested := make(chan nativeReconnectObservation, 1)
		before.win.Update(func() {
			state.revealMainConversation()
			state.requestConnectionReconnect()
			requested <- observeNativeReconnect()
		})
		select {
		case observation := <-requested:
			receipt["readyFalseObservedAfterCallback"] = !observation.ready
			receipt["requestedUTC"] = time.Now().UTC().Format(time.RFC3339Nano)
		case <-ctx.Done():
			finish(errors.New("owned UI thread did not run reconnect callback"))
			return
		}
		after, err := waitNativeReconnect(ctx, func(v nativeReconnectObservation) bool {
			return v.usable() && v.snapshot.NativeContributionsGeneration > before.snapshot.NativeContributionsGeneration
		})
		if err != nil {
			finish(err)
			return
		}
		receipt["after"] = nativeReconnectFacts(after)
		receipt["secondReadyUTC"] = time.Now().UTC().Format(time.RFC3339Nano)
		same := after.client == before.client && after.mux == before.mux && after.sessions == before.sessions && after.interactions == before.interactions && after.supervisor == before.supervisor && after.win == before.win && after.hostGeneration == before.hostGeneration && after.origin == before.origin && after.snapshot.ComposerHostGeneration == before.snapshot.ComposerHostGeneration && after.snapshot.SelectedID == before.snapshot.SelectedID
		receipt["sameClientMuxStoresWindowAndSupervisedHostGeneration"] = same
		if !same {
			finish(errors.New("manual reconnect replaced a production owner or supervised Host"))
			return
		}
		if after.snapshot.PermissionCatalogRevision <= before.snapshot.PermissionCatalogRevision {
			finish(errors.New("new events delivery did not re-establish permission catalog authority"))
			return
		}
		frame := make(chan struct{}, 1)
		after.win.Update(func() { frame <- struct{}{} })
		select {
		case <-frame:
			receipt["uiUpdateCallbackObserved"] = true
		case <-ctx.Done():
			finish(errors.New("owned UI thread did not process post-reconnect frame update"))
			return
		}
		// CapturePage rasterizes MyGo's last Content scene. A changed window size
		// and matching PNG geometry prove a new scene, unlike an Update callback.
		after.win.SetContentSize(width+1, height+1)
		var image []byte
		dimensions := baseline
		sceneTick := time.NewTicker(50 * time.Millisecond)
		defer sceneTick.Stop()
		for {
			image, err = after.win.CapturePage()
			if err != nil {
				finish(errors.New("owned mainView capture failed after reconnect"))
				return
			}
			dimensions, err = png.DecodeConfig(bytes.NewReader(image))
			if err != nil {
				finish(errors.New("owned mainView capture was not a valid PNG"))
				return
			}
			actualWidth, actualHeight := after.win.ContentSize()
			changed := dimensions.Width != baseline.Width || dimensions.Height != baseline.Height
			matches := actualWidth == width+1 && actualHeight == height+1 && math.Abs(float64(dimensions.Width)-float64(baseline.Width)*float64(actualWidth)/float64(width)) <= 1.5 && math.Abs(float64(dimensions.Height)-float64(baseline.Height)*float64(actualHeight)/float64(height)) <= 1.5
			if changed && matches {
				receipt["resizedContentSizeDIP"] = []int{actualWidth, actualHeight}
				break
			}
			select {
			case <-ctx.Done():
				finish(errors.New("new main Content scene was not observed at resized dimensions"))
				return
			case <-sceneTick.C:
			}
		}
		receipt["newContentSceneObservedByDimensions"] = true
		receipt["captureLayer"] = "Latest main Content scene rasterized to PNG in software by MyGo CapturePage; not native framebuffer, screen capture or visible GPU acceptance"
		if err = os.WriteFile(output, image, 0600); err != nil {
			finish(err)
			return
		}
		digest := sha256.Sum256(image)
		receipt["mainViewImageSHA256"] = hex.EncodeToString(digest[:])
		receipt["imagePixels"] = []int{dimensions.Width, dimensions.Height}
		receipt["windowHidden"] = true
		receipt["reconnectPassed"] = true
		quitErr := tryNativeReconnectProtectedQuit(ctx, after, mainReturned, receipt)
		finish(quitErr)
	}()
	main()
	close(mainReturned)
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		return errors.New("owned reconnect desktop exited before its receipt")
	}
}

// Component preflight checks the same complete task inspection but releases its
// admission lock on return. Successful quit preflight would retain that lock.
func tryNativeReconnectProtectedQuit(ctx context.Context, v nativeReconnectObservation, mainReturned <-chan struct{}, receipt map[string]any) error {
	preflight, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	asked := false
	proceeded, err := v.supervisor.ProtectTaskAction(preflight, "component", state.localTaskRunningCount, func(i taskcontrol.Inspection) (bool, error) {
		asked = true
		receipt["quitPreflightInspection"] = map[string]any{"ok": i.OK, "hostGenerationPresent": i.HostGeneration != "", "coverage": i.Coverage, "activeWorkCount": len(i.ActiveWork), "scheduledWorkCount": len(i.ScheduledWork), "localRunning": i.LocalRunning, "unknown": i.Unknown != "", "needsConsent": i.NeedsConsent()}
		return false, nil
	}, func() error { return nil })
	receipt["quitPreflightOperation"] = "component guard admission with noop and release; not certification of quit"
	receipt["quitPreflightCompleteIdle"] = proceeded && err == nil && !asked
	if !proceeded || err != nil || asked {
		receipt["protectedQuitStatus"] = "blocked at preflight; no native confirmation or production quit requested"
		return nil
	}
	called := make(chan struct{}, 1)
	v.win.Update(func() { state.requestQuitProtected(v.win, v.supervisor); called <- struct{}{} })
	select {
	case <-called:
		receipt["productionQuitCallbackObserved"] = true
	case <-ctx.Done():
		return errors.New("owned UI did not invoke production quit")
	}
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		state.mu.Lock()
		finished, pending := state.quitCleanupFinished, state.quitRequestPending
		state.mu.Unlock()
		if finished && v.supervisor.Status().State == harness.Stopped {
			select {
			case <-mainReturned:
				receipt["protectedGracefulQuitCertified"] = true
				receipt["protectedQuitStatus"] = "production idle quit cleanup finished and main returned"
				return nil
			case <-ctx.Done():
				return errors.New("production quit cleanup finished but main did not return")
			}
		}
		if !pending && !finished {
			return errors.New("production quit declined after complete idle preflight")
		}
		select {
		case <-ctx.Done():
			return errors.New("production protected quit did not finish; no native confirmation automated")
		case <-tick.C:
		}
	}
}
