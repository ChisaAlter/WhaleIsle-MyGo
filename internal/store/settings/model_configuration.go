package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"whaleisle/internal/dsh/api"
)

// ConfigurationFields resolves the Host schema at a profile or model path.
func ConfigurationFields(raw json.RawMessage, path ...string) []Field {
	root, err := resolveRoot(raw)
	if err != nil {
		return nil
	}
	var visit func(schemaNode, []string) []Field
	visit = func(node schemaNode, rest []string) []Field {
		if node.Type == "union" || node.Type == "intersect" {
			var fields []Field
			positions := map[string]int{}
			for _, ref := range node.List {
				child, err := resolveRef(node, ref)
				if err != nil {
					continue
				}
				for _, field := range visit(child, rest) {
					if index, exists := positions[field.Key]; exists {
						for _, option := range field.EnumOptions {
							found := false
							for _, current := range fields[index].EnumOptions {
								found = found || current == option
							}
							if !found {
								fields[index].EnumOptions = append(fields[index].EnumOptions, option)
							}
						}
						continue
					}
					positions[field.Key] = len(fields)
					fields = append(fields, field)
				}
			}
			return fields
		}
		if len(rest) == 0 {
			var fields []Field
			collectFields(node, "", &fields)
			for i := range fields {
				child, err := resolveRef(node, node.Dict[fields[i].Key])
				if err == nil && child.Type == "const" {
					var value string
					if json.Unmarshal(child.Value, &value) == nil {
						fields[i].Kind = FieldEnum
						fields[i].EnumOptions = []string{value}
					}
				}
			}
			return fields
		}
		var ref json.RawMessage
		switch node.Type {
		case "object":
			ref = node.Dict[rest[0]]
		case "dict", "array":
			ref = node.Inner
		default:
			return nil
		}
		child, err := resolveRef(node, ref)
		if err != nil {
			return nil
		}
		return visit(child, rest[1:])
	}
	return visit(root, path)
}

// SetConfigurationForm keeps creation form text separate from dirty settings.
func (s *Store) SetConfigurationForm(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.selectedNS == "" {
		return
	}
	s.ensureDraftsLocked(s.selectedNS)
	s.drafts[s.selectedNS][key] = value
}

func (s *Store) ModelConfigurationCatalog() ([]VisionRoute, bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]VisionRoute(nil), s.modelRoutes...), s.modelCatalogLoading, s.modelCatalogErr
}

func (s *Store) EnsureModelConfigurationCatalog(ctx context.Context) {
	s.mu.Lock()
	loaded := s.modelCatalogLoaded || s.modelCatalogLoading || s.modelCatalogErr != ""
	s.mu.Unlock()
	if !loaded {
		s.RefreshModelConfigurationCatalog(ctx)
	}
}

func (s *Store) InvalidateModelConfigurationCatalog(ctx context.Context) {
	s.mu.Lock()
	s.modelCatalogRequest++
	s.modelCatalogLoaded, s.modelCatalogLoading, s.modelCatalogErr = false, false, ""
	enabled := false
	if ns := s.findNamespaceLocked("subagent-model-selection-settings"); ns != nil {
		s.ensureDraftsLocked(ns.NS)
		enabled = s.drafts[ns.NS]["enabled"] == "true"
	}
	s.mu.Unlock()
	if enabled {
		s.RefreshModelConfigurationCatalog(ctx)
	}
}

// RefreshModelConfigurationCatalog uses the global catalog, not a current session selection.
func (s *Store) RefreshModelConfigurationCatalog(ctx context.Context) {
	s.mu.Lock()
	client, generation := s.client, s.generation
	if client == nil || s.modelCatalogLoading {
		s.mu.Unlock()
		return
	}
	s.modelCatalogLoading = true
	s.modelCatalogRequest++
	request := s.modelCatalogRequest
	s.modelCatalogErr = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		var catalog api.ModelCatalog
		err := client.Call(ctx, "session/modelCatalog", map[string]any{}, &catalog)
		var routes []VisionRoute
		for _, group := range catalog.Groups {
			for _, model := range group.Models {
				routes = append(routes, VisionRoute{Provider: group.Id, Model: model.Id, Label: fmt.Sprintf("%s / %s [%s / %s]", group.Name, model.Name, group.Id, model.Id)})
			}
		}
		s.mu.Lock()
		if generation != s.generation || request != s.modelCatalogRequest {
			s.mu.Unlock()
			return
		}
		s.modelCatalogLoading = false
		if err != nil {
			s.modelCatalogErr = err.Error()
		} else {
			s.modelCatalogLoaded = true
			s.modelCatalog = catalog
			s.modelRoutes = routes
			for _, failure := range catalog.Failures {
				s.modelCatalogErr += failure.Name + ": " + failure.Message + "\n"
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// ConfigurationModels retains the live catalog's per-model metadata for the account editor.
func (s *Store) ConfigurationModels(provider string) ([]any, bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := []any{}
	for _, group := range s.modelCatalog.Groups {
		if group.Id == provider {
			for _, model := range group.Models {
				row := map[string]any{"id": model.Id, "name": model.Name}
				if model.ContextWindow != nil {
					row["contextWindow"] = *model.ContextWindow
				}
				if model.MaxTokens != nil {
					row["maxTokens"] = *model.MaxTokens
				}
				if len(model.InputModalities) > 0 {
					modalities := make([]any, len(model.InputModalities))
					for i, modality := range model.InputModalities {
						modalities[i] = modality
					}
					row["inputModalities"] = modalities
				}
				rows = append(rows, row)
			}
		}
	}
	return rows, s.modelCatalogLoading, s.modelCatalogErr
}

// ConfigurableProviders is the current Host directory, not an invented list of APIs.
func (s *Store) ConfigurableProviders() (api.ListConfigurableProvidersResult, bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append(api.ListConfigurableProvidersResult(nil), s.providerDirectory...)
	for i := range out {
		out[i].SettingsPath = append([]string(nil), out[i].SettingsPath...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		priority := func(id string) int {
			if id == "deepseek-account" {
				return 0
			}
			if id == "deepseek-official" {
				return 1
			}
			return 2
		}
		return priority(out[i].Provider) < priority(out[j].Provider)
	})
	return out, s.providerDirectoryLoading, s.providerDirectoryErr
}

func (s *Store) RegisteredProviders() (api.ListProvidersResult, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append(api.ListProvidersResult(nil), s.providerRegistered...), s.providerDirectoryLoaded
}

func (s *Store) ProviderPageOwner() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("%p/%d/%d", s.client, s.generation, providerPage(s))
}

func (s *Store) AccountModelsAvailable() (available, known bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.modelCatalogLoaded {
		return false, false
	}
	for _, group := range s.modelCatalog.Groups {
		if group.Id == "deepseek-account" {
			return len(group.Models) > 0, true
		}
	}
	return false, true
}

func (s *Store) EnsureProviderDirectory(ctx context.Context) {
	s.mu.Lock()
	needed := !s.providerDirectoryLoaded && !s.providerDirectoryLoading && s.providerDirectoryErr == ""
	s.mu.Unlock()
	if needed {
		s.RefreshProviderDirectory(ctx)
	}
}

func (s *Store) RefreshProviderDirectory(ctx context.Context) {
	s.mu.Lock()
	client, generation := s.client, s.generation
	if client == nil || s.providerDirectoryLoading {
		s.mu.Unlock()
		return
	}
	s.providerDirectoryLoading, s.providerDirectoryErr = true, ""
	s.mu.Unlock()
	s.fire()
	go func() {
		var directory api.ListConfigurableProvidersResult
		err := client.Call(ctx, "llm/listConfigurableProviders", nil, &directory)
		var registered api.ListProvidersResult
		if err == nil {
			err = client.Call(ctx, "llm/listProviders", nil, &registered)
		}
		s.mu.Lock()
		if s.generation != generation || s.client != client {
			s.mu.Unlock()
			return
		}
		s.providerDirectoryLoading = false
		if err == nil {
			s.providerDirectory, s.providerDirectoryLoaded = directory, true
			s.providerRegistered = registered
		} else {
			s.providerDirectoryErr = err.Error()
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// AccountModelDefaults reads the migration baseline declared by the current Host schema.
func AccountModelDefaults(schema json.RawMessage) []any {
	root, err := resolveRoot(schema)
	if err != nil {
		return nil
	}
	node, err := resolveRef(root, root.Dict["models"])
	if err != nil || node.Meta == nil {
		return nil
	}
	var extra struct {
		Rows []any `json:"legacyModelDefaults"`
	}
	_ = json.Unmarshal(node.Meta.Extra, &extra)
	return extra.Rows
}
