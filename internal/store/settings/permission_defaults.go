package settings

import (
	"context"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// PermissionDefaults is the Host catalog for future-session defaults. It is
// deliberately independent of the current-session catalog's Options collection.
type PermissionDefaults struct {
	Options       []PermissionDefaultOption
	DefaultPreset string
	Loaded        bool
	Err           string
}

type PermissionDefaultOption struct {
	Value       string
	Name        string
	Description string
}

func (v PermissionDefaults) clone() PermissionDefaults {
	v.Options = append([]PermissionDefaultOption(nil), v.Options...)
	return v
}

func loadPermissionDefaults(ctx context.Context, client *dsh.Client, describe api.SettingsDescribeValue) PermissionDefaults {
	for _, namespace := range describe.Namespaces {
		if namespace.Ns != "permission" {
			continue
		}
		var catalog api.PermissionCatalog
		if err := client.Call(ctx, "permissionPresets/catalog", map[string]any{}, &catalog); err != nil {
			return PermissionDefaults{Err: err.Error()}
		}
		result := PermissionDefaults{Loaded: true, DefaultPreset: catalog.DefaultPreset}
		for _, option := range catalog.DefaultOptions {
			row := PermissionDefaultOption{Value: option.Value, Name: option.Name}
			if option.Description != nil {
				row.Description = *option.Description
			}
			result.Options = append(result.Options, row)
		}
		return result
	}
	return PermissionDefaults{}
}
