package pluginmanager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"unicode/utf8"
)

func requestID() string {
	var bytes [16]byte
	_, _ = rand.Read(bytes[:])
	return hex.EncodeToString(bytes[:])
}
func (a Actions) OpenInstall() { a.c.openInstall(a.owner, "", "", nil, false) }
func (a Actions) Update(name string) {
	s := a.c.Snapshot()
	if s.Owner != a.owner {
		return
	}
	for _, p := range s.Packages {
		if p.Name == name && UpdateAvailable(p) {
			enabled := p.Enabled
			a.c.openInstall(a.owner, p.InstallTarget.Spec, "", &enabled, true)
			return
		}
	}
}
func (c *Controller) openInstall(owner uint64, spec, registry string, enabled *bool, exact bool) {
	c.mu.Lock()
	if c.state.Owner != owner || c.client == nil {
		c.mu.Unlock()
		return
	}
	if c.state.Install.RequestID != "" && (installPending(c.state.Install.Phase) || c.state.Install.Phase == "unknown") {
		c.state.Install.Open = true
		c.mu.Unlock()
		c.fire()
		return
	}
	c.state.Install = InstallState{Open: true, Spec: spec, Registry: registry, Enabled: enabled, SaveExact: exact, Phase: "idle"}
	c.installRead++
	c.registryRead++
	read := c.registryRead
	client, ctx := Client{c.client}, c.life
	c.mu.Unlock()
	c.fire()
	go func() {
		registries, err := client.Registries(ctx)
		c.patch(owner, func(s *State) {
			if c.registryRead != read {
				return
			}
			if err != nil {
				s.Install.Error = err.Error()
				return
			}
			s.Install.RegistryInfo = &registries
		})
	}()
}
func (a Actions) CloseInstall() {
	c := a.c
	c.mu.Lock()
	if c.state.Owner != a.owner {
		c.mu.Unlock()
		return
	}
	pending := installPending(c.state.Install.Phase)
	request := c.state.Install.RequestID
	c.state.Install.Open = false
	if !pending && c.state.Install.Phase != "unknown" {
		if c.inspectCancel != nil {
			c.inspectCancel()
			c.inspectCancel = nil
		}
		c.installRead++
		c.registryRead++
		c.state.Install = InstallState{Phase: "idle"}
	}
	c.mu.Unlock()
	c.fire()
	if pending {
		a.cancelInstall(request)
	}
}
func (a Actions) EditInstall(spec, registry string) {
	a.edit(func(s *State) {
		if s.Busy || s.Install.Phase == "checking" || s.Install.Phase == "unconfirmed" || s.Install.Phase == "unknown" {
			return
		}
		a.c.installRead++
		if s.Install.Spec != spec || s.Install.Registry != registry {
			s.Install.ApprovedBuilds = nil
		}
		s.Install.Spec = spec
		s.Install.Registry = registry
		s.Install.Inspection = nil
		s.Install.Error = ""
		s.Install.Result = nil
		s.Install.Phase = "idle"
	})
}
func (a Actions) Inspect() {
	c := a.c
	c.mu.Lock()
	if c.state.Owner != a.owner || c.client == nil || c.state.Busy || c.state.Install.Phase == "checking" || c.state.Install.Phase == "unconfirmed" || c.state.Install.Phase == "unknown" {
		c.mu.Unlock()
		return
	}
	spec := strings.TrimSpace(c.state.Install.Spec)
	registry := c.state.Install.Registry
	if spec == "" {
		c.mu.Unlock()
		return
	}
	c.state.Install.Spec = spec
	client := Client{c.client}
	ctx, cancel := context.WithCancel(c.life)
	c.inspectCancel = cancel
	c.installRead++
	read := c.installRead
	c.state.Install.Phase = "checking"
	c.state.Install.Error = ""
	c.mu.Unlock()
	c.fire()
	go func() {
		defer cancel()
		var selected *string
		if registry != "" {
			selected = &registry
		}
		inspection, err := client.Inspect(ctx, spec, selected)
		c.patch(a.owner, func(s *State) {
			if c.installRead != read || s.Install.Spec != spec || s.Install.Registry != registry {
				return
			}
			c.inspectCancel = nil
			s.Install.Phase = "idle"
			if err != nil {
				s.Install.Error = err.Error()
				return
			}
			s.Install.Inspection = &inspection
			if inspection.Bundle != nil && !*inspection.Bundle {
				s.Install.Error = "The package does not declare a plugin bundle."
			}
			if inspection.Status != "accepted" {
				s.Install.Error = inspection.Problem + ": " + inspection.Reason
			}
		})
	}()
}
func (a Actions) RequestInstall() {
	c := a.c
	c.mu.Lock()
	defer c.mu.Unlock()
	s := &c.state
	i := &s.Install
	if s.Owner != a.owner || !s.Connected || s.Busy || i.Inspection == nil || i.Inspection.Status != "accepted" || (i.Inspection.Bundle != nil && !*i.Inspection.Bundle) || i.Phase == "unconfirmed" || i.Phase == "unknown" {
		return
	}
	id := requestID()
	options := InstallOptions{Enabled: i.Enabled, SaveExact: i.SaveExact, RequestID: id, ApprovedBuilds: append([]string{}, i.ApprovedBuilds...)}
	if i.Registry != "" {
		registry := i.Registry
		options.Registry = &registry
	}
	c.pending = &action{kind: "install", spec: i.Spec, target: i.Inspection.Name, owner: a.owner, revision: c.revision, options: options}
	s.Confirmation = &Confirmation{Kind: "install", Target: i.Spec, Description: "Install " + i.Spec + "? Plugins run with the profile's permissions."}
	i.RequestID = id
	go c.fire()
}
func (a Actions) ApproveBuilds() {
	a.edit(func(s *State) {
		if s.Install.Result == nil || len(s.Install.Result.PendingBuilds) == 0 {
			return
		}
		s.Install.ApprovedBuilds = append([]string{}, s.Install.Result.PendingBuilds...)
		s.Install.Phase = "idle"
		s.Install.Result = nil
	})
}
func (a Actions) RequestExemption(pkg, runtime string) {
	c := a.c
	c.mu.Lock()
	if c.state.Owner != a.owner || c.state.Busy {
		c.mu.Unlock()
		return
	}
	valid := false
	if result := c.state.Install.Result; result != nil && result.Error != nil {
		for _, item := range result.Error.Incompatible {
			if item.Name+"@"+item.Version == pkg && item.RuntimeVersion == runtime {
				valid = true
			}
		}
	}
	if valid {
		c.pending = &action{kind: "exemption", target: pkg, runtime: runtime, owner: a.owner, revision: c.revision}
		c.state.Confirmation = &Confirmation{Kind: "exemption", Target: pkg, Description: "Accept the compatibility risk and exempt " + pkg + " for runtime " + runtime + "?"}
	}
	c.mu.Unlock()
	c.fire()
}
func (a Actions) CancelInstall() { a.cancelInstall("") }
func (a Actions) cancelInstall(expectedID string) {
	c := a.c
	c.mu.Lock()
	i := &c.state.Install
	if expectedID != "" && i.RequestID != expectedID {
		c.mu.Unlock()
		return
	}
	if c.state.Owner == a.owner && i.Phase == "checking" {
		if c.inspectCancel != nil {
			c.inspectCancel()
			c.inspectCancel = nil
		}
		c.installRead++
		i.Phase, i.Error = "idle", ""
		i.Inspection, i.Result = nil, nil
		c.mu.Unlock()
		c.fire()
		return
	}
	if c.state.Owner != a.owner || c.client == nil || i.RequestID == "" || (i.Phase != "starting" && i.Phase != "running" && i.Phase != "unconfirmed") || (i.CancellationRequested && !i.WaitingForStart) {
		c.mu.Unlock()
		return
	}
	id, client, ctx := c.state.Install.RequestID, Client{c.client}, c.life
	acknowledged := i.Acknowledged
	i.CancellationRequested = true
	i.WaitingForStart = false
	i.Phase = "cancelling"
	c.mu.Unlock()
	c.fire()
	go func() {
		status, err := client.CancelInstall(ctx, id)
		retry := false
		reconcile := false
		c.patch(a.owner, func(s *State) {
			if s.Install.RequestID != id || !installPending(s.Install.Phase) {
				return
			}
			if err != nil {
				if s.Install.Phase != "applying" {
					s.Install.Phase = "unconfirmed"
				}
				s.Install.WaitingForStart = !s.Install.Acknowledged
				if s.Install.Acknowledged {
					s.Install.CancellationRequested = false
				}
				s.Install.Error = err.Error()
				return
			}
			switch status {
			case "cancelled":
				s.Install.Phase = "cancelled"
			case "too-late":
				s.Install.Phase = "applying"
				s.Install.Acknowledged = true
				s.Install.CancellationRequested = false
				reconcile = true
			default:
				if s.Install.Phase == "applying" {
					return
				}
				s.Install.Phase = "unconfirmed"
				s.Install.WaitingForStart = !acknowledged
				retry = !acknowledged && s.Install.Acknowledged
				if acknowledged {
					s.Install.CancellationRequested = false
				}
				s.Install.Error = "No active request was confirmed; reconcile the result before retrying."
			}
		})
		if retry {
			a.cancelInstall(id)
		}
		if reconcile {
			c.reconcileInstall(a.owner, id)
		}
	}()
}
func (a Actions) ReconcileInstall() {
	c := a.c
	c.mu.Lock()
	id := c.state.Install.RequestID
	c.mu.Unlock()
	c.reconcileInstall(a.owner, id)
}
func (c *Controller) reconcileInstall(owner uint64, id string) {
	c.mu.Lock()
	if c.state.Owner != owner || c.client == nil || id == "" || c.state.Install.RequestID != id {
		c.mu.Unlock()
		return
	}
	client, ctx := Client{c.client}, c.life
	c.mu.Unlock()
	go func() {
		result, err := client.WaitInstall(ctx, id)
		c.patch(owner, func(s *State) {
			if s.Install.RequestID != id {
				return
			}
			if err != nil {
				s.Install.Error = err.Error()
				return
			}
			if result == nil {
				s.Install.Phase = "unknown"
				s.Install.Error = "The Host has no retained installation result. Refresh inventory before any new install."
				return
			}
			s.Install.Result = result
			s.Install.Phase = "done"
			if result.Application == "failed" || result.Application == "cancelled" {
				s.Install.Phase = result.Application
			}
			s.Install.Error = ""
		})
	}()
}

// Events are accepted only for the captured Host owner and installation id.
func (c *Controller) AppendLog(owner uint64, chunk InstallLog) {
	retryCancellation := false
	c.patch(owner, func(s *State) {
		if chunk.RequestID == "" || chunk.RequestID != s.Install.RequestID {
			return
		}
		i := &s.Install
		if i.Phase == "starting" || i.Phase == "cancelling" || i.Phase == "unconfirmed" {
			retryCancellation = progressInstall(i, InstallProgress{RequestID: chunk.RequestID, Phase: "installing"})
		}
		index := -1
		for n := range i.Runs {
			if i.Runs[n].JobID == chunk.JobID {
				index = n
				break
			}
		}
		if index == -1 {
			if !installPending(i.Phase) {
				return
			}
			i.Runs = append(i.Runs, InstallRun{JobID: chunk.JobID, Argv: append([]string{}, chunk.Argv...), Command: strings.Join(chunk.Argv, " "), CWD: chunk.CWD})
			index = len(i.Runs) - 1
		}
		run := &i.Runs[index]
		run.Output += chunk.Text
		if chunk.Text != "" {
			if n := len(run.Chunks); n > 0 && run.Chunks[n-1].Stream == chunk.Stream {
				run.Chunks[n-1].Text += chunk.Text
			} else {
				run.Chunks = append(run.Chunks, InstallOutput{Stream: chunk.Stream, Text: chunk.Text})
			}
		}
		if chunk.ExitCodeSet || chunk.ExitCode != nil {
			run.Settled = true
			run.ExitCode = chunk.ExitCode
		}
		boundRuns(i)
		s.Install.Log += chunk.Text
		s.Install.Log = outputTail(s.Install.Log, 64000)
	})
	if retryCancellation {
		c.Actions(owner).CancelInstall()
	}
}
func (c *Controller) Progress(owner uint64, progress InstallProgress) {
	retryCancellation := false
	c.patch(owner, func(s *State) {
		retryCancellation = progressInstall(&s.Install, progress)
	})
	if retryCancellation {
		c.Actions(owner).CancelInstall()
	}
}

func installPending(phase string) bool {
	return phase == "starting" || phase == "running" || phase == "cancelling" || phase == "applying" || phase == "unconfirmed"
}

// Returns true only when acknowledgement releases a cancellation waiting for start.
func progressInstall(i *InstallState, p InstallProgress) bool {
	if p.RequestID == "" || p.RequestID != i.RequestID || !installPending(i.Phase) {
		return false
	}
	if p.Phase != "installing" && p.Phase != "cancelling" && p.Phase != "applying" {
		return false
	}
	i.Acknowledged = true
	if i.Phase == "applying" {
		return false
	}
	if p.Phase == "installing" && (i.CancellationRequested || i.Phase == "cancelling") {
		return i.WaitingForStart
	}
	if p.Phase == "installing" {
		i.Phase = "running"
	} else {
		i.Phase = p.Phase
	}
	if p.Attempt != nil {
		i.Attempts = append(i.Attempts, *p.Attempt)
	}
	return false
}

func outputTail(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	text = text[len(text)-limit:]
	for len(text) > 0 && !utf8.ValidString(text) {
		text = text[1:]
	}
	return text
}

// Keep command identities and final statuses while bounding retained output.
func boundRuns(i *InstallState) {
	if len(i.Runs) > 32 {
		i.Runs = i.Runs[len(i.Runs)-32:]
	}
	remaining := 64000
	for n := len(i.Runs) - 1; n >= 0; n-- {
		run := &i.Runs[n]
		before := len(run.Output)
		run.Output = outputTail(run.Output, remaining)
		run.Truncated = run.Truncated || len(run.Output) < before
		remaining -= len(run.Output)
		budget := len(run.Output)
		for k := len(run.Chunks) - 1; k >= 0; k-- {
			run.Chunks[k].Text = outputTail(run.Chunks[k].Text, budget)
			budget -= len(run.Chunks[k].Text)
			if budget == 0 {
				run.Chunks = run.Chunks[k:]
				break
			}
		}
		if len(run.Chunks) > 256 {
			run.Chunks = run.Chunks[len(run.Chunks)-256:]
		}
	}
}
