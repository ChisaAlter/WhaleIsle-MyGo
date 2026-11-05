package projectflow

import (
	"context"
	"fmt"
	"strings"
)

const NotesMarker = "<!-- whale-project:user-notes -->"

type Material struct {
	Path           string  `json:"path"`
	Text           string  `json:"text"`
	GeneratedText  string  `json:"generatedText"`
	UserText       *string `json:"userText"`
	Decisions      string  `json:"-"`
	DecisionError  string  `json:"-"`
	DecisionLoaded bool    `json:"-"`
}

func (m Material) Editable() string {
	if m.Path == "notes.md" && m.UserText != nil {
		return *m.UserText
	}
	return m.Text
}
func materialPath(path string) bool {
	return path == "notes.md" || path == "preferences.md" || path == "charter.md"
}
func (c Client) ReadMaterial(ctx context.Context, id, path string) (Material, error) {
	out, err := c.material(ctx, "store/read", id, path, nil)
	if err == nil && path == "notes.md" {
		var log struct {
			Path string `json:"path"`
			Text string `json:"text"`
		}
		logErr := c.call(ctx, "store/read", map[string]any{"projectId": id, "path": "decisions.md"}, &log)
		out.DecisionLoaded = true
		if logErr != nil {
			out.DecisionError = logErr.Error()
		} else if log.Path != "decisions.md" {
			out.DecisionError = "Host did not return the requested decision log"
		} else {
			out.Decisions = log.Text
		}
	}
	return out, err
}
func (c Client) WriteMaterial(ctx context.Context, id, path, text string) (Material, error) {
	if path == "charter.md" && len(text) > 16000 {
		return Material{}, fmt.Errorf("Project rules are limited to 16000 UTF-8 bytes; shorten them before saving")
	}
	if len(text) > 256000 || (path == "notes.md" && strings.Contains(text, NotesMarker)) {
		return Material{}, fmt.Errorf("Only the user notes body or preferences up to 256000 UTF-8 bytes may be saved")
	}
	return c.material(ctx, "store/write", id, path, &text)
}
func (c Client) material(ctx context.Context, method, id, path string, text *string) (Material, error) {
	var out Material
	if id == "" || !materialPath(path) {
		return out, fmt.Errorf("Select a Project and notes.md, preferences.md or charter.md")
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
	Decisions, DecisionError                                string
	DecisionLoaded                                          bool
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
	if m.DecisionLoaded {
		d.DecisionLoaded = true
		d.DecisionError = m.DecisionError
		if m.DecisionError == "" {
			d.Decisions = m.Decisions
		}
	}
	d.Loaded = true
	d.Error = ""
}
