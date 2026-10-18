package projectflow

import (
	"context"
	"fmt"
	"strings"
)

const NotesMarker = "<!-- whale-project:user-notes -->"

type Material struct {
	Path          string  `json:"path"`
	Text          string  `json:"text"`
	GeneratedText string  `json:"generatedText"`
	UserText      *string `json:"userText"`
}

func (m Material) Editable() string {
	if m.Path == "notes.md" && m.UserText != nil {
		return *m.UserText
	}
	return m.Text
}
func materialPath(path string) bool { return path == "notes.md" || path == "preferences.md" }
func (c Client) ReadMaterial(ctx context.Context, id, path string) (Material, error) {
	return c.material(ctx, "store/read", id, path, nil)
}
func (c Client) WriteMaterial(ctx context.Context, id, path, text string) (Material, error) {
	if len(text) > 256000 || (path == "notes.md" && strings.Contains(text, NotesMarker)) {
		return Material{}, fmt.Errorf("Only the user notes body or preferences up to 256000 UTF-8 bytes may be saved")
	}
	return c.material(ctx, "store/write", id, path, &text)
}
func (c Client) material(ctx context.Context, method, id, path string, text *string) (Material, error) {
	var out Material
	if id == "" || !materialPath(path) {
		return out, fmt.Errorf("Select a Project and notes.md or preferences.md")
	}
	payload := map[string]any{"projectId": id, "path": path}
	if text != nil {
		payload["text"] = *text
	}
	err := c.call(ctx, method, payload, &out)
	if err == nil && (out.Path != path || (path == "notes.md" && out.UserText == nil)) {
		err = fmt.Errorf("Host did not return the requested editable Project material")
	}
	return out, err
}

type MaterialDraft struct {
	ProjectID, Path, Draft, Saved, Generated, Error, Notice string
	Loaded                                                  bool
}

func (d *MaterialDraft) Dirty() bool { return d.Loaded && d.Draft != d.Saved }
func (d *MaterialDraft) Apply(m Material, err error) {
	if err != nil {
		d.Error = err.Error()
		d.Notice = ""
		return
	}
	d.Draft = m.Editable()
	d.Saved = d.Draft
	d.Generated = m.GeneratedText
	d.Loaded = true
	d.Error = ""
}
