package settings

import (
	"context"
	"encoding/json"
	"fmt"
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

// RefreshModelConfigurationCatalog uses the global catalog, not a current session selection.
func (s *Store) RefreshModelConfigurationCatalog(ctx context.Context) {
	s.mu.Lock()
	client, generation := s.client, s.generation
	if client == nil || s.modelCatalogLoading {
		s.mu.Unlock()
		return
	}
	s.modelCatalogLoading = true
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
		if generation != s.generation {
			s.mu.Unlock()
			return
		}
		s.modelCatalogLoading = false
		if err != nil {
			s.modelCatalogErr = err.Error()
		} else {
			s.modelRoutes = routes
			for _, failure := range catalog.Failures {
				s.modelCatalogErr += failure.Name + ": " + failure.Message + "\n"
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}
