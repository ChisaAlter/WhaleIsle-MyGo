package pluginmanager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
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
			c := a.c
			c.mu.Lock()
			if c.state.Owner != a.owner || c.client == nil || c.state.ActionBusy(p.Name) {
				c.mu.Unlock()
				return
			}
			client, ctx := Client{c.client}, c.life
			if c.state.BusyKeys == nil {
				c.state.BusyKeys = map[string]bool{}
			}
			c.state.BusyKeys[p.Name] = true
			c.mu.Unlock()
			c.fire()
			go func() {
				result, err := client.Remove(ctx, p.Name)
				accepted := err == nil && (result.Application != "failed" || result.Error != nil && result.Error.Code == "stop-profile")
				if !c.patch(a.owner, func(s *State) {
					delete(s.BusyKeys, p.Name)
					if err != nil {
						s.Notice = err.Error()
					} else {
						s.Notice = changeMessage(result)
					}
				}) {
					return
				}
				if accepted {
					c.openCatalogInstall(a.owner, p, p.Enabled)
				}
				_ = c.Load(ctx)
				c.BundlesChanged(a.owner)
			}()
			return
		}
	}
}
func (c *Controller) openInstall(owner uint64, spec, registry string, enabled *bool, exact bool) bool {
	c.mu.Lock()
	if c.state.Owner != owner || c.client == nil {
		c.mu.Unlock()
		return false
	}
	if c.state.Install.RequestID != "" && c.state.Install.Phase != "reviewed" {
		c.state.Install.Open = true
		c.mu.Unlock()
		c.fire()
		return false
	}
	if c.inspectCancel != nil {
		c.inspectCancel()
		c.inspectCancel = nil
	}
	choice := RegistryChoice{Kind: "offered"}
	if c.registryMemory != nil {
		choice = *cloneRegistryChoice(c.registryMemory)
	}
	transient := registry != ""
	if transient {
		value := registry
		choice = RegistryChoice{Kind: "custom", URL: value}
	}
	c.state.Install = InstallState{Open: true, Spec: spec, Enabled: enabled, SaveExact: exact, Phase: "idle", TransientRegistry: transient}
	setRegistrySelection(&c.state.Install, choice)
	c.installRead++
	c.registryRead++
	c.choiceRead++
	read, selection := c.registryRead, c.choiceRead
	client, ctx := Client{c.client}, c.life
	c.mu.Unlock()
	c.fire()
	go c.readInstallRegistries(ctx, client, owner, read, selection)
	return true
}
func (c *Controller) openCatalogInstall(owner uint64, p Package, enabled bool) {
	if p.InstallTarget == nil || !c.openInstall(owner, p.InstallTarget.Spec, "", &enabled, !strings.HasPrefix(p.InstallTarget.Spec, "link:")) {
		return
	}
	c.patch(owner, func(s *State) {
		bundle := true
		s.Install.Inspection = &Inspection{Status: "accepted", Name: p.Name, Version: p.InstallTarget.Version, Bundle: &bundle}
		s.Install.Confirmed = true
		s.Install.Title = p.Name
		s.Install.Version = p.InstallTarget.Version
		if p.Meta != nil {
			s.Install.ConfirmedTitle = p.Meta.Title
		}
	})
	c.Actions(owner).startConfirmedInstall()
}
func (a Actions) startConfirmedInstall() {
	c := a.c
	c.mu.Lock()
	if c.state.Owner != a.owner {
		c.mu.Unlock()
		return
	}
	if !c.state.Install.RegistryReadDone {
		c.state.Install.StartAfterRegistry = true
		c.state.Install.WaitingForRegistry = true
		c.state.Install.Phase = "checking"
		c.mu.Unlock()
		c.fire()
		return
	}
	c.state.Install.StartAfterRegistry = false
	if c.state.Install.WaitingForRegistry {
		c.state.Install.WaitingForRegistry = false
		c.state.Install.Phase = "idle"
	} else if c.state.Install.Phase == "checking" {
		c.mu.Unlock()
		return
	}
	prepared := c.state.Install.Inspection != nil
	c.mu.Unlock()
	if prepared {
		a.RequestInstall()
	} else {
		a.Inspect()
	}
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
	if !pending {
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
		if s.Busy || s.Install.Phase == "checking" || installPending(s.Install.Phase) || s.Install.Phase == "unknown" {
			return
		}
		a.c.installRead++
		if s.Install.Phase != "idle" {
			reopenInstallInput(&s.Install)
		}
		if s.Install.Spec != spec || s.Install.Registry != registry {
			s.Install.ApprovedBuilds = nil
		}
		if s.Install.Registry != registry {
			a.c.choiceRead++
			setRegistrySelection(&s.Install, RegistryChoice{Kind: "custom", URL: registry})
			s.Install.TransientRegistry = false
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
	if c.state.Owner != a.owner || c.client == nil || c.state.Busy || c.state.Install.Phase == "checking" || installPending(c.state.Install.Phase) || c.state.Install.Phase == "unknown" {
		c.mu.Unlock()
		return
	}
	spec := strings.TrimSpace(c.state.Install.Spec)
	registry := c.state.Install.Registry
	if spec == "" {
		c.mu.Unlock()
		return
	}
	if !c.state.Install.RegistryReadDone {
		c.state.Install.StartAfterRegistry = true
		c.state.Install.WaitingForRegistry = true
		c.state.Install.Phase = "checking"
		c.mu.Unlock()
		c.fire()
		return
	}
	choice := c.state.Install.RegistrySelection
	if choice.Kind == "custom" {
		raw := strings.TrimSpace(choice.URL)
		if !registryInputValid(raw) {
			c.state.Install.RegistryError = true
			c.state.Install.RegistryOpen = true
			c.mu.Unlock()
			c.fire()
			return
		}
		choice.URL = raw
		setRegistrySelection(&c.state.Install, choice)
		registry = raw
	}
	for _, pkg := range c.state.Packages {
		if pkg.Name == spec && pkg.Availability != "missing" && c.state.Install.Enabled == nil {
			c.state.Install.Error = "Plugin is already installed: " + spec
			if pkg.Availability == "installation" {
				c.state.Install.Error = "This plugin is supplied by the runtime: " + spec
			}
			c.mu.Unlock()
			c.fire()
			return
		}
	}
	remember := !c.state.Install.TransientRegistry
	c.state.Install.Spec = spec
	client := Client{c.client}
	ctx, cancel := context.WithCancel(c.life)
	c.inspectCancel = cancel
	c.installRead++
	read := c.installRead
	c.state.Install.Phase = "checking"
	c.state.Install.Error = ""
	c.state.Install.Result = nil
	c.state.Install.RequestID = ""
	c.state.Install.ApprovedBuilds = nil
	c.state.Install.Attempts = nil
	c.state.Install.Runs = nil
	c.state.Install.Log = ""
	c.state.Install.DetailsOpen = false
	c.state.Install.RegistryOpen = false
	c.mu.Unlock()
	c.fire()
	if remember {
		c.rememberRegistry(a.owner, choice)
	}
	go func() {
		defer cancel()
		var selected *string
		if registry != "" {
			selected = &registry
		}
		inspection, err := client.Inspect(ctx, spec, selected)
		automatic := false
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
			automatic = s.Install.Error == ""
		})
		if automatic {
			a.RequestInstall()
		}
	}()
}
func (a Actions) RequestInstall() {
	c := a.c
	c.mu.Lock()
	s := &c.state
	i := &s.Install
	if s.Owner != a.owner || !s.Connected || s.Busy || i.Inspection == nil || i.Inspection.Status != "accepted" || (i.Inspection.Bundle != nil && !*i.Inspection.Bundle) || i.Phase == "checking" || installPending(i.Phase) || i.Phase == "unknown" {
		c.mu.Unlock()
		return
	}
	id := requestID()
	enabled := i.Enabled
	if enabled == nil {
		disabled := false
		enabled = &disabled
	}
	options := InstallOptions{Enabled: enabled, SaveExact: i.SaveExact, RequestID: id, ApprovedBuilds: append([]string{}, i.ApprovedBuilds...), Registry: i.Inspection.Registry}
	if i.Enabled != nil && i.Registry != "" {
		registry := i.Registry
		options.Registry = &registry
	}
	remember := i.Enabled != nil && !i.TransientRegistry
	choice := i.RegistrySelection
	op := action{kind: "install", spec: i.Spec, target: i.Inspection.Name, owner: a.owner, revision: c.revision, options: options}
	i.RequestID = id
	i.Phase = "checking"
	i.DetailsOpen = false
	i.RegistryOpen = false
	i.ApprovedBuilds = nil
	c.mu.Unlock()
	if remember {
		c.rememberRegistry(a.owner, choice)
	}
	c.run(op)
}
func (a Actions) RetryInstall() {
	s := a.c.Snapshot()
	if s.Owner != a.owner {
		return
	}
	if s.Install.Enabled != nil && s.Install.Inspection != nil {
		a.RequestInstall()
	} else {
		a.Inspect()
	}
}
func (a Actions) ApproveBuilds() {
	approved := false
	a.edit(func(s *State) {
		if s.Install.Phase != "failed" || s.Install.Result == nil || len(s.Install.Result.PendingBuilds) == 0 {
			return
		}
		s.Install.ApprovedBuilds = append([]string{}, s.Install.Result.PendingBuilds...)
		s.Install.Phase = "idle"
		s.Install.Result = nil
		approved = true
	})
	if approved {
		a.RequestInstall()
	}
}
func (a Actions) EnableInstalled() {
	c := a.c
	c.mu.Lock()
	i := &c.state.Install
	if c.state.Owner != a.owner || c.client == nil || i.Phase != "done" || i.Enabling || i.Result == nil {
		c.mu.Unlock()
		return
	}
	name := i.Result.Bundle
	if name == "" {
		c.mu.Unlock()
		return
	}
	i.Enabling = true
	client, ctx := Client{c.client}, c.life
	c.mu.Unlock()
	c.fire()
	go func() {
		var result Change
		var err error
		if name != "" {
			result, err = client.SetBundleEnabled(ctx, name, true)
		}
		if !c.patch(a.owner, func(s *State) {
			if err != nil {
				s.Notice = err.Error()
			} else if name != "" {
				s.Notice = changeMessage(result)
			}
			s.Install = InstallState{Phase: "idle"}
			s.Tab = "installed"
			s.Highlight = name
			s.HighlightUntil = time.Now().Add(2400 * time.Millisecond).UnixNano()
		}) {
			return
		}
		_ = c.Load(ctx)
		c.BundlesChanged(a.owner)
	}()
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
	if c.state.Owner == a.owner && (i.Phase == "checking" || i.Phase == "failed" || i.Phase == "unknown") {
		if c.inspectCancel != nil {
			c.inspectCancel()
			c.inspectCancel = nil
		}
		c.installRead++
		reopenInstallInput(i)
		c.state.Busy = false
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
		cancelled := false
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
				reopenInstallInput(&s.Install)
				s.Busy = false
				s.Notice = "Installation cancelled."
				cancelled = true
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
		if cancelled {
			_ = c.Load(ctx)
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
	if c.state.Owner != owner || c.client == nil || id == "" || c.state.Install.RequestID != id || c.state.Install.Recovering || !installPending(c.state.Install.Phase) {
		c.mu.Unlock()
		return
	}
	client, ctx := Client{c.client}, c.life
	c.state.Install.Recovering = true
	c.mu.Unlock()
	c.fire()
	go func() {
		result, err := client.WaitInstall(ctx, id)
		settled := false
		c.patch(owner, func(s *State) {
			if s.Install.RequestID != id {
				return
			}
			s.Install.Recovering = false
			if !installPending(s.Install.Phase) {
				return
			}
			if err != nil {
				s.Install.Error = err.Error()
				return
			}
			if result == nil {
				s.Install.Phase = "unknown"
				s.Install.Error = "The Host has no retained installation result."
				s.Busy = false
				settleInstallRuns(&s.Install, nil)
				return
			}
			settleInstallState(s, *result)
			s.Notice = changeMessage(*result)
			settled = true
		})
		if settled {
			_ = c.Load(ctx)
		}
	}()
}

func settleInstallRuns(i *InstallState, exitCode *int) {
	for n := range i.Runs {
		if !i.Runs[n].Settled {
			i.Runs[n].Settled = true
			i.Runs[n].ExitCode = exitCode
		}
	}
}
func settleInstallState(s *State, result Change) {
	i := &s.Install
	s.Busy = false
	if result.Application == "cancelled" {
		reopenInstallInput(i)
		s.Notice = "Installation cancelled."
		return
	}
	i.Recovering = false
	i.Result = &result
	i.Phase = "done"
	i.Error = ""
	exitCode := 0
	if result.Application == "failed" {
		i.Phase = "failed"
		var code *int
		if result.PackageResult != nil {
			code = result.PackageResult.ExitCode
		}
		settleInstallRuns(i, code)
	} else {
		settleInstallRuns(i, &exitCode)
	}
	if result.Registries != nil {
		i.Attempts = nil
		for n, registry := range result.Registries {
			i.Attempts = append(i.Attempts, InstallAttempt{Registry: registry, Index: n + 1, Total: len(result.Registries)})
		}
	}
	if i.Phase == "done" && i.Enabled != nil && result.Bundle != "" {
		s.Highlight = result.Bundle
		s.HighlightUntil = time.Now().Add(2400 * time.Millisecond).UnixNano()
	}
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
