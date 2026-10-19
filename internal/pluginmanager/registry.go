package pluginmanager

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

const OfficialNPMRegistry = "https://registry.npmjs.org/"
const NPMMirrorRegistry = "https://registry.npmmirror.com/"

func cloneRegistryChoice(value *RegistryChoice) *RegistryChoice {
	if value == nil {
		return nil
	}
	copy := *value
	if value.Registry != nil {
		registry := *value.Registry
		copy.Registry = &registry
	}
	return &copy
}
func RegistryValue(choice RegistryChoice) *string {
	if choice.Kind == "custom" {
		value := strings.TrimSpace(choice.URL)
		return &value
	}
	if choice.Registry == nil {
		return nil
	}
	value := *choice.Registry
	return &value
}
func NormalizeRegistry(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if parsed != nil {
		parsed.Scheme = strings.ToLower(parsed.Scheme)
	}
	if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return "", fmt.Errorf("a registry must be an http(s) URL: %s", raw)
	}
	parsed.Host = strings.ToLower(parsed.Host)
	if parsed.Scheme == "https" && parsed.Port() == "443" || parsed.Scheme == "http" && parsed.Port() == "80" {
		parsed.Host = parsed.Hostname()
		if strings.Contains(parsed.Host, ":") {
			parsed.Host = "[" + parsed.Host + "]"
		}
	}
	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path += "/"
	}
	return parsed.String(), nil
}
func registryInputValid(raw string) bool {
	return ((strings.HasPrefix(raw, "http://") && len(raw) > 7) || (strings.HasPrefix(raw, "https://") && len(raw) > 8)) && strings.IndexFunc(raw, unicode.IsSpace) < 0
}
func RegistryKey(registry, resolved *string) string {
	if registry == nil {
		registry = resolved
	}
	if registry == nil {
		return ""
	}
	normalized, err := NormalizeRegistry(*registry)
	if err != nil {
		return *registry
	}
	return normalized
}
func OfferedRegistries(info *RegistryInfo) []*string {
	offered := []*string{}
	seen := map[string]bool{}
	candidates := []*string{}
	var resolved *string
	if info != nil {
		candidates = append(candidates, info.Registry)
		resolved = info.Resolved
		for _, registry := range info.FallbackRegistries {
			value := registry
			candidates = append(candidates, &value)
		}
	}
	candidates = append(candidates, nil)
	for _, registry := range candidates {
		key := RegistryKey(registry, resolved)
		if !seen[key] {
			offered = append(offered, registry)
			seen[key] = true
		}
	}
	return offered
}
func reconciledRegistry(remembered *RegistryChoice, info *RegistryInfo) RegistryChoice {
	if remembered == nil {
		return RegistryChoice{Kind: "offered", Registry: info.Registry}
	}
	if remembered.Kind == "custom" || remembered.Registry == nil {
		return *cloneRegistryChoice(remembered)
	}
	key := RegistryKey(remembered.Registry, info.Resolved)
	for _, offered := range OfferedRegistries(info) {
		if RegistryKey(offered, info.Resolved) == key {
			return RegistryChoice{Kind: "offered", Registry: offered}
		}
	}
	return RegistryChoice{Kind: "custom", URL: *remembered.Registry}
}
func setRegistrySelection(install *InstallState, choice RegistryChoice) {
	install.RegistrySelection = *cloneRegistryChoice(&choice)
	install.Registry = ""
	if value := RegistryValue(choice); value != nil {
		install.Registry = *value
	}
}

// SetRegistryPersistence uses the desktop preference store; it never reads a
// profile credential or changes Host package-manager configuration.
func (c *Controller) SetRegistryPersistence(load func() (*RegistryChoice, error), save func(*RegistryChoice) error) {
	var value *RegistryChoice
	var err error
	if load != nil {
		value, err = load()
	}
	c.mu.Lock()
	c.registryMemory = cloneRegistryChoice(value)
	c.registrySave = save
	if err != nil {
		c.state.PreferenceError = "Could not read the remembered installation registry: " + err.Error()
	}
	c.mu.Unlock()
	c.fire()
}
func (a Actions) DismissPreferenceError() { a.edit(func(s *State) { s.PreferenceError = "" }) }
func (c *Controller) rememberRegistry(owner uint64, choice RegistryChoice) {
	c.mu.Lock()
	if c.state.Owner != owner {
		c.mu.Unlock()
		return
	}
	c.registryMemory = cloneRegistryChoice(&choice)
	save := c.registrySave
	c.mu.Unlock()
	var err error
	if save != nil {
		err = save(cloneRegistryChoice(&choice))
	}
	c.patch(owner, func(s *State) {
		s.PreferenceError = ""
		if err != nil {
			s.PreferenceError = "Could not remember the installation registry: " + err.Error()
		}
	})
}
func (a Actions) ChooseRegistry(choice RegistryChoice) {
	a.edit(func(s *State) {
		if s.Install.Phase != "idle" {
			return
		}
		a.c.choiceRead++
		setRegistrySelection(&s.Install, choice)
		s.Install.TransientRegistry = false
		s.Install.RegistryError = false
	})
}
func (a Actions) ToggleRegistryOptions() {
	a.edit(func(s *State) { s.Install.RegistryOpen = !s.Install.RegistryOpen })
}
func (a Actions) ToggleInstallDetails() {
	a.edit(func(s *State) { s.Install.DetailsOpen = !s.Install.DetailsOpen })
}
func (a Actions) ChangeRegistry() {
	a.edit(func(s *State) {
		if s.Install.Phase != "failed" || s.Install.Confirmed {
			return
		}
		reopenInstallInput(&s.Install)
		s.Install.RegistryOpen = true
	})
}
func reopenInstallInput(install *InstallState) {
	kept := *install
	*install = InstallState{Open: kept.Open, Spec: kept.Spec, RegistryInfo: kept.RegistryInfo, RegistryReadDone: kept.RegistryReadDone, Registry: kept.Registry, RegistrySelection: kept.RegistrySelection, Confirmed: kept.Confirmed, ConfirmedTitle: kept.ConfirmedTitle, Title: kept.Title, Version: kept.Version, TransientRegistry: kept.TransientRegistry, MirrorRecovery: kept.MirrorRecovery, Phase: "idle"}
	if kept.Enabled != nil {
		install.Inspection = kept.Inspection
		install.Enabled = kept.Enabled
		install.SaveExact = kept.SaveExact
	}
}
func GitHubRecoveryRegistry(install InstallState) (*string, bool) {
	if install.Confirmed || install.Phase != "failed" || install.Result == nil || install.Result.FailedAt != "spec-host" || install.Result.PackageResult == nil {
		return nil, false
	}
	kind := install.Result.PackageResult.Kind
	if kind != "network" && kind != "timeout" {
		return nil, false
	}
	if install.Inspection == nil {
		return nil, false
	}
	host := strings.ToLower(strings.Split(install.Inspection.Host, ":")[0])
	if host != "github.com" && !strings.HasSuffix(host, ".github.com") {
		return nil, false
	}
	var resolved *string
	if install.RegistryInfo != nil {
		resolved = install.RegistryInfo.Resolved
	}
	for _, registry := range OfferedRegistries(install.RegistryInfo) {
		if RegistryKey(registry, resolved) == NPMMirrorRegistry {
			return registry, true
		}
	}
	return nil, false
}
func (a Actions) UseGitHubMirror() {
	c := a.c
	c.mu.Lock()
	if c.state.Owner != a.owner {
		c.mu.Unlock()
		return
	}
	registry, offered := GitHubRecoveryRegistry(c.state.Install)
	if !offered {
		c.mu.Unlock()
		return
	}
	var resolved *string
	if c.state.Install.RegistryInfo != nil {
		resolved = c.state.Install.RegistryInfo.Resolved
	}
	already := RegistryKey(RegistryValue(c.state.Install.RegistrySelection), resolved) == NPMMirrorRegistry
	reopenInstallInput(&c.state.Install)
	c.state.Install.Spec = ""
	c.state.Install.MirrorRecovery = true
	choice := c.state.Install.RegistrySelection
	if !already {
		choice = RegistryChoice{Kind: "offered", Registry: registry}
		setRegistrySelection(&c.state.Install, choice)
	}
	c.mu.Unlock()
	c.fire()
	if !already {
		c.rememberRegistry(a.owner, choice)
	}
}
func eligiblePublicMirror(info RegistryInfo) bool {
	if info.Registry != nil || info.Resolved == nil || RegistryKey(info.Resolved, nil) != OfficialNPMRegistry {
		return false
	}
	for _, registry := range info.FallbackRegistries {
		if registry == NPMMirrorRegistry {
			return true
		}
	}
	return false
}
func (c *Controller) readInstallRegistries(ctx context.Context, client Client, owner, read, choiceRead uint64) {
	info, err := client.Registries(ctx)
	probe := false
	if !c.patch(owner, func(s *State) {
		i := &s.Install
		if c.registryRead != read || !i.Open {
			return
		}
		if err == nil {
			i.RegistryInfo = &info
			untouched := c.choiceRead == choiceRead && !i.TransientRegistry && (i.Phase == "idle" || i.Phase == "checking")
			if untouched {
				setRegistrySelection(i, reconciledRegistry(c.registryMemory, &info))
			}
			probe = untouched && c.registryMemory == nil && eligiblePublicMirror(info)
		}
	}) {
		return
	}
	if probe {
		fastest, probeErr := client.FastestRegistry(ctx)
		c.patch(owner, func(s *State) {
			i := &s.Install
			if c.registryRead == read && c.choiceRead == choiceRead && i.Open && (i.Phase == "idle" || i.Phase == "checking") && probeErr == nil && fastest != nil && *fastest == NPMMirrorRegistry {
				value := *fastest
				setRegistrySelection(i, RegistryChoice{Kind: "offered", Registry: &value})
			}
		})
	}
	start := false
	c.patch(owner, func(s *State) {
		if c.registryRead != read || !s.Install.Open {
			return
		}
		s.Install.RegistryReadDone = true
		start = s.Install.StartAfterRegistry
	})
	if start {
		c.Actions(owner).startConfirmedInstall()
	}
}
