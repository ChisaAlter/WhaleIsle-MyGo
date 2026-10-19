package settings

import "encoding/json"

// DeveloperToolsEnabled reads the accepted Host preference, never a pending form draft.
func DeveloperToolsEnabled(v ViewState) bool {
	p := struct {
		Enabled bool `json:"enabled"`
	}{}
	for _, ns := range v.Namespaces {
		if ns.NS == "ui-settings" {
			_ = json.Unmarshal(ns.Value, &p)
			break
		}
	}
	return p.Enabled
}
