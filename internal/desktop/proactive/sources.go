package proactive

import (
	"crypto/rand"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Source holds a native user's read-only subscription. It never authorizes
// executing a discovered issue or passing its body to a model automatically.
type Source struct {
	ID          string `json:"id"`
	Repository  string `json:"repository"`
	Enabled     bool   `json:"enabled"`
	Revision    uint64 `json:"revision"`
	LastChecked string `json:"lastChecked,omitempty"`
	LastSuccess string `json:"lastSuccess,omitempty"`
	NextCheck   string `json:"nextCheck,omitempty"`
	Error       string `json:"error,omitempty"`
}

type SourceItem struct {
	ID          string   `json:"id"`
	Repository  string   `json:"repository"`
	Number      int64    `json:"number"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Kind        string   `json:"kind"`
	State       string   `json:"state"`
	Labels      []string `json:"labels,omitempty"`
	UpdatedAt   string   `json:"updatedAt"`
}

const SourceInterval = 15 * time.Minute

var repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9_.-]+$`)

func NormalizeRepository(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "https://github.com/")
	value = strings.TrimSuffix(value, "/")
	if !repositoryPattern.MatchString(value) || strings.Contains(value, "..") {
		return "", fmt.Errorf("enter a GitHub repository as owner/name")
	}
	return strings.ToLower(value), nil
}

func (s *Service) ConfigureSource(repository string, enabled bool) (Source, error) {
	repo, err := NormalizeRepository(repository)
	if err != nil {
		return Source{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := "github:" + repo
	source := s.sources[id]
	source.ID, source.Repository, source.Enabled = id, repo, enabled
	source.Revision++
	source.Error = ""
	source.NextCheck = timestamp(s.now())
	if err := s.commit(Event{Type: "source.configured", Source: &source}); err != nil {
		return Source{}, err
	}
	return clone(source), nil
}

func (s *Service) RefreshSource(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	source, ok := s.sources[id]
	if !ok {
		return ErrNotFound
	}
	if !source.Enabled {
		return fmt.Errorf("resume this source before checking it")
	}
	source.Revision++
	source.NextCheck = timestamp(s.now())
	source.Error = ""
	return s.commit(Event{Type: "source.configured", Source: &source})
}

func (s *Service) Sources() []Source {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Source, 0, len(s.sourceOrder))
	for _, id := range s.sourceOrder {
		result = append(result, s.sources[id])
	}
	return clone(result)
}

// ApplyDiscovery commits a complete paginated scan and its cursor together.
// A source disabled/reconfigured during network I/O cannot import late results.
func (s *Service) ApplyDiscovery(source Source, started time.Time, items []SourceItem, readErr error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.sources[source.ID]
	if !ok || !current.Enabled || current.Revision != source.Revision {
		return ErrConflict
	}
	current.LastChecked = timestamp(s.now())
	current.NextCheck = timestamp(s.now().Add(SourceInterval))
	current.Error = ""
	if readErr != nil {
		current.Error = readErr.Error()
		return s.commit(Event{Type: "source.checked", Source: &current})
	}
	latest := map[string]SourceItem{}
	for _, item := range items {
		if prior, ok := latest[item.ID]; !ok || prior.UpdatedAt <= item.UpdatedAt {
			latest[item.ID] = item
		}
	}
	items = make([]SourceItem, 0, len(latest))
	for _, item := range latest {
		items = append(items, item)
	}
	var events []Event
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt == items[j].UpdatedAt {
			return items[i].ID < items[j].ID
		}
		return items[i].UpdatedAt < items[j].UpdatedAt
	})
	added := 0
	for _, item := range items {
		if item.Repository != source.Repository || item.ID == "" || item.Number <= 0 {
			return fmt.Errorf("source response contains an item outside the selected repository")
		}
		if _, err := time.Parse(time.RFC3339Nano, item.UpdatedAt); err != nil {
			return fmt.Errorf("source response has an invalid update time")
		}
		if item.State != "open" && item.State != "closed" {
			return fmt.Errorf("source response has an invalid item state")
		}
		// GitHub issue IDs are stable across repository renames and shared by
		// the issues/pulls APIs, so aliases never create a second task.
		id := "github:" + item.ID
		if prior, exists := s.tasks[id]; exists {
			if prior.SourceItem != nil {
				before, _ := time.Parse(time.RFC3339Nano, prior.SourceItem.UpdatedAt)
				after, _ := time.Parse(time.RFC3339Nano, item.UpdatedAt)
				if before.After(after) || reflect.DeepEqual(*prior.SourceItem, item) {
					continue
				}
			}
			events = append(events, Event{Type: "task.source", TaskID: id, SourceItem: &item})
			continue
		}
		// Closed historical items do not create work; a closed update for an
		// existing proposal is retained without reopening or cancelling its run.
		if item.State == "closed" {
			continue
		}
		task := Task{ID: id, Title: item.Title, Description: item.Description, Autonomy: 1, State: "discovered", Source: "github", SourceURL: item.URL, Kind: item.Kind, Labels: item.Labels, CreatedAt: timestamp(s.now()), UpdatedAt: timestamp(s.now()), SourceItem: &item}
		events = append(events, Event{Type: "task.created", Task: &task})
		added++
	}
	if added > 0 {
		n := Notice{ID: "discovery:" + rand.Text(), SourceID: source.ID, Kind: "discovery", Title: source.Repository, Body: fmt.Sprintf("%d", added), CreatedAt: timestamp(s.now()), DesktopStatus: "pending"}
		events = append(events, Event{Type: "notice.created", Notice: &n})
	}
	current.LastSuccess = timestamp(started)
	events = append(events, Event{Type: "source.checked", Source: &current})
	return s.commit(events...)
}
