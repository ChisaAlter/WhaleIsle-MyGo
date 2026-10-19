// Package projectflow follows the native Project Host control plane, not the session CRUD API.
package projectflow

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"
	"whaleisle/internal/desktop/pluginrpc"
)

type Project struct {
	Queue                     *RequestQueue `json:"queue"`
	Archiving                 bool          `json:"archiving"`
	Paused                    bool          `json:"paused"`
	ID                        string        `json:"id"`
	Title                     string        `json:"title"`
	CanonicalWorkingDirectory string        `json:"canonicalWorkingDirectory"`
	CoordinatorSessionID      string        `json:"coordinatorSessionId"`
	Lifecycle                 string        `json:"lifecycle"`
	Activity                  struct {
		State              string `json:"state"`
		CoordinatorRunning bool   `json:"coordinatorRunning"`
		Running            int    `json:"running"`
		Queued             int    `json:"queued"`
		Provisioning       int    `json:"provisioning"`
	} `json:"activity"`
}
type Listing struct {
	Projects  []Project `json:"projects"`
	Available bool      `json:"available"`
	Error     string    `json:"error"`
}
type Client struct {
	Origin, Cookie string
	HTTP           *http.Client
}

func (c Client) call(ctx context.Context, method string, payload any, out any) error {
	return (&pluginrpc.Client{BaseURL: c.Origin, Cookie: c.Cookie, HTTPClient: c.HTTP}).Call(ctx, "dsh-project", method, payload, out)
}
func (c Client) List(ctx context.Context) (Listing, error) {
	var out Listing
	err := c.call(ctx, "list", map[string]any{}, &out)
	if err == nil && !out.Available && out.Error == "" {
		out.Error = "Project Host is unavailable."
	}
	return out, err
}
func (c Client) Create(ctx context.Context, path string, authorized []string, requestID string) (Project, error) {
	allowed := false
	for _, p := range authorized {
		if p != "" && path == p {
			allowed = true
			break
		}
	}
	if !allowed || strings.TrimSpace(requestID) == "" {
		return Project{}, fmt.Errorf("select an existing authorized workspace as the picker starting directory")
	}
	var out struct {
		Project Project `json:"project"`
	}
	err := c.call(ctx, "create", map[string]any{"workingDirectory": path, "requestId": requestID}, &out)
	return out.Project, err
}
func (c Client) Rename(ctx context.Context, id, title string) (Project, error) {
	title = strings.TrimSpace(title)
	if id == "" || title == "" || utf8.RuneCountInString(title) > 240 {
		return Project{}, fmt.Errorf("project title must contain 1–240 characters")
	}
	var out struct {
		Project Project `json:"project"`
	}
	err := c.call(ctx, "rename", map[string]any{"projectId": id, "title": title}, &out)
	return out.Project, err
}

// Opening the original Lead reads detail; it never creates a replacement Session or prompts a model.
func (c Client) Open(ctx context.Context, id string) (string, error) {
	var out struct {
		Project Project `json:"project"`
	}
	if id == "" {
		return "", fmt.Errorf("select a Project")
	}
	if err := c.call(ctx, "detail", map[string]any{"projectId": id}, &out); err != nil {
		return "", err
	}
	if out.Project.ID != id || out.Project.CoordinatorSessionID == "" {
		return "", fmt.Errorf("Project Lead session is unavailable")
	}
	return out.Project.CoordinatorSessionID, nil
}
