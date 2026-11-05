package settings

import (
	"context"
	"encoding/json"

	"whaleisle/internal/dsh/api"
	store "whaleisle/internal/store/settings"
)

type modelProviderRow struct {
	entry                                               api.ListConfigurableProvidersResultItem
	namespace                                           store.Namespace
	active, configured, keyConfigured, accountAvailable bool
}

// modelProviderDirectory uses the same join as the published Models store:
// registered routes, configurable positions, account catalog and key metadata.
func modelProviderDirectory(m Model) (rows []modelProviderRow, anyUsable, readinessKnown bool) {
	directory, _, _ := m.Store.ConfigurableProviders()
	registered, readinessKnown := m.Store.RegisteredProviders()
	if credentials := m.ReferencedCredentials; credentials != nil && !credentials.Snapshot.Ready && credentials.Snapshot.Err == "" {
		readinessKnown = false
	}
	active, declared := map[string]bool{}, map[string]bool{}
	for _, provider := range registered {
		active[provider.Id] = true
	}
	for _, entry := range directory {
		declared[entry.Provider] = true
	}
	for _, provider := range registered {
		if !declared[provider.Id] {
			anyUsable = true
		}
	}
	for _, entry := range directory {
		accountAvailable := false
		if entry.Provider == "deepseek-account" {
			m.Store.EnsureModelConfigurationCatalog(context.Background())
			available, known := m.Store.AccountModelsAvailable()
			_, _, catalogError := m.Store.ModelConfigurationCatalog()
			if !known && catalogError == "" {
				readinessKnown = false
			}
			accountAvailable = available
			anyUsable = anyUsable || active[entry.Provider] && available
		}
		binding := m.Store.BindNamespace(entry.SettingsNs)
		if binding == nil {
			// An active route without an editable namespace owes this page no
			// named key, just like a registered route absent from the directory.
			anyUsable = anyUsable || active[entry.Provider] && entry.Provider != "deepseek-account"
			continue
		}
		state := binding.Snapshot()
		if len(state.Namespaces) == 0 {
			continue
		}
		ns := state.Namespaces[0]
		var effective any
		_ = json.Unmarshal(ns.Value, &effective)
		profile, configured := providerPathValue(effective, entry.SettingsPath)
		if len(entry.SettingsPath) == 0 {
			configured = true
		}
		row := modelProviderRow{entry: entry, namespace: ns, active: active[entry.Provider], configured: configured}
		if entry.Provider == "deepseek-account" {
			row.accountAvailable = accountAvailable
			managed := false
			for _, raw := range []json.RawMessage{ns.User, ns.Base} {
				var layer any
				_ = json.Unmarshal(raw, &layer)
				for _, field := range []string{"models", "modelOverrides"} {
					_, exists := providerPathValue(layer, append(append([]string(nil), entry.SettingsPath...), field))
					managed = managed || exists
				}
			}
			if !accountAvailable && !managed {
				continue
			}
		} else {
			ref := configurationText(configurationObjectValue(profile)["apiKeyEnv"])
			if m.ReferencedCredentials != nil {
				for _, credential := range m.ReferencedCredentials.Snapshot.Rows {
					if credential.Namespace == entry.SettingsNs && credential.Provider == entry.Provider {
						row.keyConfigured = credential.Known && credential.Configured
						break
					}
				}
			}
			anyUsable = anyUsable || row.active && (ref == "" || row.keyConfigured)
		}
		// An adopted directory provider is visible immediately in this editor's
		// draft, before its first commit; it never contributes usable=true yet.
		if !row.configured && len(entry.SettingsPath) > 0 {
			var draft any
			if json.Unmarshal([]byte(state.Drafts[entry.SettingsPath[0]]), &draft) == nil {
				_, row.configured = providerPathValue(draft, entry.SettingsPath[1:])
			}
		}
		if row.configured {
			rows = append(rows, row)
		}
	}
	return rows, anyUsable, readinessKnown
}
