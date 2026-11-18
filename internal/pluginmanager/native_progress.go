package pluginmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// One ordered import keeps its submitted prefix separate from entries never
// sent to the Host. Cancellation cannot turn the latter into unknown writes.
type nativeProgress struct {
	mu              sync.Mutex
	labels          []string
	sent, completed int
	running         bool
}

func newNativeProgress(op action) *nativeProgress {
	p := &nativeProgress{}
	for _, raw := range op.mcpSpecs {
		var spec struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(raw, &spec)
		p.labels = append(p.labels, "MCP "+spec.ID)
	}
	for _, skill := range op.localSkills {
		p.labels = append(p.labels, "Skill "+skill.Name)
	}
	return p
}
func (p *nativeProgress) setRunning(running bool) { p.mu.Lock(); p.running = running; p.mu.Unlock() }
func (p *nativeProgress) startNext()              { p.mu.Lock(); p.sent++; p.mu.Unlock() }
func (p *nativeProgress) completeNext()           { p.mu.Lock(); p.completed++; p.mu.Unlock() }
func (p *nativeProgress) snapshot() (sent, completed int, running bool, warnings []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, label := range p.labels {
		state := "not submitted"
		if i < p.completed {
			state = "saved"
		} else if i < p.sent {
			state = "result unconfirmed"
		}
		warnings = append(warnings, label+": "+state)
	}
	return p.sent, p.completed, p.running, warnings
}

func (c Client) reconcileNative(ctx context.Context, op action) (Change, error) {
	if op.progress == nil {
		return c.confirmNative(ctx, op)
	}
	sent, _, running, warnings := op.progress.snapshot()
	if running {
		return Change{}, fmt.Errorf("native installation is still stopping")
	}
	confirmed := op
	if sent < len(op.mcpSpecs) {
		confirmed.mcpSpecs = op.mcpSpecs[:sent]
		confirmed.localSkills = nil
	} else {
		confirmed.localSkills = op.localSkills[:sent-len(op.mcpSpecs)]
	}
	var result Change
	if sent > 0 {
		var err error
		if result, err = c.confirmNative(ctx, confirmed); err != nil {
			return Change{}, fmt.Errorf("%s; %w", strings.Join(warnings, "; "), err)
		}
	}
	if sent < len(op.mcpSpecs)+len(op.localSkills) {
		for i := 0; i < sent; i++ {
			warnings[i] = op.progress.labels[i] + ": saved"
		}
		return Change{Changed: sent > 0, Application: "failed", Stage: "partial", Target: op.target, Warnings: warnings, Error: &ManagementError{Code: "native-install/partial", Diagnostic: "Installation stopped. Saved entries were retained; remaining entries were not submitted. Import only the remaining entries."}}, nil
	}
	return result, nil
}

// FinishNativeReview leaves uncertain writes visible without holding every
// later install hostage. It never retries, removes, or replaces any entry.
func (a Actions) FinishNativeReview() {
	c := a.c
	c.mu.Lock()
	op := c.nativeInstall
	if c.state.Owner != a.owner || op == nil || op.progress == nil || c.state.Install.RequestID != op.options.RequestID || c.state.Install.Phase != "unconfirmed" || c.state.Install.Recovering {
		c.mu.Unlock()
		return
	}
	_, completed, running, warnings := op.progress.snapshot()
	if running {
		c.mu.Unlock()
		return
	}
	result := Change{Changed: completed > 0, Application: "failed", Stage: "unconfirmed", Target: op.target, Warnings: warnings, Error: &ManagementError{Code: "native-install/unconfirmed", Diagnostic: "Review ended with unconfirmed entries. Inspect MCP Settings and Skills before importing those entries again."}}
	settleInstallState(&c.state, result)
	c.state.Notice = changeMessage(result)
	c.mu.Unlock()
	c.fire()
}
