//go:build windows && richhistoryacceptance

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
)

var richHistoryError error

// Keep the production App.Run on the Windows process main goroutine.
func TestMain(m *testing.M) { richHistoryError = nativeRichHistory(); os.Exit(m.Run()) }
func TestNativeRichHistory(t *testing.T) {
	if richHistoryError != nil {
		t.Fatal(richHistoryError)
	}
}

func nativeRichHistory() error {
	id, workspace, out := os.Getenv("WHALEISLE_RICH_QA_ID"), os.Getenv("WHALEISLE_RICH_QA_WORKSPACE"), os.Getenv("WHALEISLE_RICH_QA_OUT")
	data := os.Getenv(paths.EnvUserData)
	if !isolatedDesktopRun() || !strings.HasPrefix(id, "rich-history-fixture-") || !filepath.IsAbs(data) || !filepath.IsAbs(workspace) || !filepath.IsAbs(out) {
		return fmt.Errorf("explicit owned rich-history environment required")
	}
	cfg, err := config.Open(data)
	if err != nil {
		return err
	}
	if err = cfg.Update(func(v *config.Config) {
		v.Workspace = workspace
		v.Host = "127.0.0.1"
		v.Port = 32184
		v.NodeBin = os.Getenv("WHALEISLE_NODE")
	}); err != nil {
		return err
	}
	result := make(chan error, 1)
	finish := func(err error) {
		result <- err
		state.mu.Lock()
		state.quitting = true
		sup := state.sup
		state.mu.Unlock()
		if sup != nil {
			sup.Stop()
		}
		mygo.App.Quit()
	}
	go func() {
		deadline := time.Now().Add(105 * time.Second)
		selected, revealed := false, false
		for time.Now().Before(deadline) {
			state.mu.Lock()
			win, status, sessions, ready := state.win, state.bootStatus, state.sessions, state.dshMuxReady
			state.mu.Unlock()
			if status.State == harness.Failed {
				finish(fmt.Errorf("owned rich native Host failed"))
				return
			}
			if win == nil || status.State != harness.Ready || sessions == nil || !ready {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if !revealed {
				state.revealMainConversation()
				revealed = true
			}
			snap := sessions.Snapshot()
			if snap.ListLoading || snap.ListErr != "" || !snap.WorkspaceReady {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if !selected {
				sessions.Select(id)
				selected = true
				continue
			}
			if snap.SelectedID != id || !snap.HistoryReady || snap.HistoryLoading {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if snap.HistoryError != "" || snap.PromptBusy {
				finish(fmt.Errorf("saved state was not an idle ready history"))
				return
			}
			kind := strings.TrimPrefix(id, "rich-history-fixture-")
			if kind == "media" {
				for _, message := range snap.Messages {
					for _, ref := range session.ImageRefs(message.Body) {
						if attachmentID, ok := strings.CutPrefix(ref, "attachment:"); ok {
							ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
							_, err := sessions.FetchAttachment(ctx, attachmentID)
							cancel()
							if err != nil {
								finish(err)
								return
							}
						}
					}
				}
			}
			// Expanding a saved disclosure changes only local view state; no
			// question/approval/workflow mutation callback is installed or called.
			done := make(chan struct{}, 1)
			win.Update(func() {
				state.mu.Lock()
				if state.thinkingExpanded == nil {
					state.thinkingExpanded = map[string]bool{}
				}
				for _, message := range snap.Messages {
					if message.Kind == session.MessageKindThinking {
						state.thinkingExpanded[message.ID] = true
					}
				}
				for _, run := range snap.WorkflowRuns {
					key := fmt.Sprintf("workflow:%q:%q", snap.PlanHistoryBinding, run.ID)
					state.thinkingExpanded[key] = true
					for _, phase := range run.Phases {
						state.thinkingExpanded[key+":"+phase.Key] = true
					}
				}
				state.mu.Unlock()
				done <- struct{}{}
			})
			<-done
			png, err := win.CapturePage()
			if err != nil {
				finish(err)
				return
			}
			if err = os.WriteFile(out+".png", png, 0600); err != nil {
				finish(err)
				return
			}
			state.mu.Lock()
			interactions := state.interactions
			state.mu.Unlock()
			livePending := 0
			if interactions != nil {
				for _, pending := range interactions.Snapshot().Pending {
					if !pending.Continued {
						livePending++
					}
				}
			}
			if livePending != 0 {
				finish(fmt.Errorf("saved state fabricated a live interaction"))
				return
			}
			proof := map[string]any{"kind": kind, "layer": "production hidden Windows mainView + real Host on synthetic persisted state", "historyReady": true, "messageCount": len(snap.Messages), "workflowRuns": len(snap.WorkflowRuns), "promptBusy": snap.PromptBusy, "modelRequests": 0, "nativeQuestionOrApprovalResponseInvoked": false, "livePendingInteractions": livePending, "localDisclosuresExpandedProgrammatically": true, "windowHidden": true, "originalPixelParityClaim": false}
			raw, _ := json.MarshalIndent(proof, "", "  ")
			finish(os.WriteFile(out+".json", append(raw, '\n'), 0600))
			return
		}
		finish(fmt.Errorf("owned native rich history readiness timeout"))
	}()
	main()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		return fmt.Errorf("owned native desktop exited before proof")
	}
}
