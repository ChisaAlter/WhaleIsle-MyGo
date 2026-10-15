// Package jobflow implements the native view of the Host's existing job protocol.
// It never starts a job or synthesizes completion from a kill receipt.
package jobflow

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

const RenderTailLimit = 128 * 1024

type Job struct {
	ID         string  `json:"id"`
	Kind       string  `json:"kind"`
	Label      string  `json:"label"`
	Owner      string  `json:"owner,omitempty"`
	Status     string  `json:"status"`
	Progress   string  `json:"progress,omitempty"`
	Detail     string  `json:"detail,omitempty"`
	StartedAt  float64 `json:"startedAt"`
	FinishedAt float64 `json:"finishedAt,omitempty"`
	Output     struct {
		Total      int64    `json:"total"`
		Earliest   int64    `json:"earliest"`
		SpillPaths []string `json:"spillPaths,omitempty"`
	} `json:"output"`
}

func (j Job) Live() bool       { return j.Status == "running" || j.Status == "stopping" }
func (j Job) Observable() bool { return j.Live() || j.Output.Total > 0 }

type Chunk struct {
	At        int64  `json:"at"`
	Text      string `json:"text"`
	Channel   string `json:"channel,omitempty"`
	GapBefore bool   `json:"gapBefore,omitempty"`
}
type Frame struct {
	Type   string  `json:"type"`
	Job    Job     `json:"job"`
	From   int64   `json:"from"`
	Chunks []Chunk `json:"chunks"`
	Next   int64   `json:"next"`
	Lossy  bool    `json:"lossy,omitempty"`
}
type Observation struct {
	JobID, Text, Error              string
	GapBefore, Streaming, HasCursor bool
	Cursor                          int64
}

func (o *Observation) Apply(f Frame) error {
	switch f.Type {
	case "opened":
		if f.Job.ID == "" || f.From < 0 {
			return fmt.Errorf("invalid job output anchor")
		}
		if o.JobID != "" && o.JobID != f.Job.ID {
			return fmt.Errorf("job output identity changed")
		}
		o.JobID = f.Job.ID
		o.GapBefore = o.GapBefore || (o.Text == "" && f.From > 0) || f.From < f.Job.Output.Earliest
		o.Cursor = f.From
		o.HasCursor = true
		o.Streaming = true
		o.Error = ""
	case "output":
		if !o.HasCursor {
			return fmt.Errorf("job output arrived before its anchor")
		}
		if f.Next < o.Cursor {
			return fmt.Errorf("job output cursor moved backwards")
		}
		var b strings.Builder
		b.WriteString(o.Text)
		for _, chunk := range f.Chunks {
			b.WriteString(chunk.Text)
			o.GapBefore = o.GapBefore || chunk.GapBefore
		}
		o.Text = b.String()
		o.GapBefore = o.GapBefore || f.Lossy
		o.Cursor = f.Next
		if len(o.Text) > RenderTailLimit {
			cut := len(o.Text) - RenderTailLimit
			for cut < len(o.Text) && !utf8.RuneStart(o.Text[cut]) {
				cut++
			}
			o.Text = o.Text[cut:]
			o.GapBefore = true
		}
	case "status":
		if !o.HasCursor || f.Job.ID != o.JobID || f.Job.Live() {
			return fmt.Errorf("invalid terminal job status")
		}
		o.Streaming = false
	default:
		return fmt.Errorf("unknown job output frame %q", f.Type)
	}
	return nil
}
func order(rows []Job) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Live() != b.Live() {
			return a.Live()
		}
		if a.Live() {
			return a.StartedAt < b.StartedAt
		}
		af, bf := a.FinishedAt, b.FinishedAt
		if af == 0 {
			af = a.StartedAt
		}
		if bf == 0 {
			bf = b.StartedAt
		}
		if af == bf {
			return a.StartedAt < b.StartedAt
		}
		return af > bf
	})
}
