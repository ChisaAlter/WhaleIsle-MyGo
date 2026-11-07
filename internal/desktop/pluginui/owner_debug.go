package pluginui

import (
	"encoding/json"
	"os"

	"github.com/egoist/mygo"
)

// Opt-in snapshot dump for live debugging: WHALEISLE_PLUGINUI_DUMP=<path>
// appends one JSON line per accepted page snapshot.
func debugDumpSnapshot(owner string, s Snapshot) {
	path := os.Getenv("WHALEISLE_PLUGINUI_DUMP")
	if path == "" {
		return
	}
	type rect struct {
		X, Y, W, H, VW, VH float64
	}
	type docSum struct {
		Slot       string `json:"slot"`
		Registrant string `json:"registrant"`
		Nodes      int    `json:"nodes"`
		Ambient    int    `json:"ambient"`
		Rects      []rect `json:"rects,omitempty"`
	}
	sum := func(docs []NativeDocument) []docSum {
		out := make([]docSum, 0, len(docs))
		for _, d := range docs {
			ds := docSum{Slot: d.Slot, Registrant: d.Registrant, Nodes: len(d.Nodes)}
			for _, n := range d.Nodes {
				if n.Ambient {
					ds.Ambient++
					if len(ds.Rects) < 8 {
						ds.Rects = append(ds.Rects, rect{n.X, n.Y, n.Width, n.Height, n.VW, n.VH})
					}
				}
			}
			out = append(out, ds)
		}
		return out
	}
	raw, err := json.Marshal(map[string]any{
		"owner": owner, "revision": s.Revision, "surface": s.Surface,
		"documents": sum(s.Documents), "global": sum(s.GlobalDocuments),
	})
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(raw, '\n'))
}

// debugDumpRich records the native-side layout actually pushed to the page.
func debugDumpRich(owner string, o *Owner, bounds mygo.Rectangle, seats []richSeat, holes []mygo.Rectangle) {
	path := os.Getenv("WHALEISLE_PLUGINUI_DUMP")
	if path == "" {
		return
	}
	raw, err := json.Marshal(map[string]any{
		"owner": owner, "rich": map[string]any{
			"bounds": bounds, "seats": seats, "holes": len(holes),
			"err": o.rich.err, "applied": o.rich.applied != nil, "pending": o.rich.pending != nil,
		},
	})
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(raw, '\n'))
}

// debugDumpDiag records a page-side diagnostic payload under the same env gate.
func debugDumpDiag(owner string, data json.RawMessage) {
	path := os.Getenv("WHALEISLE_PLUGINUI_DUMP")
	if path == "" || len(data) == 0 {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	line := make([]byte, 0, len(data)+64)
	line = append(line, `{"owner":`...)
	enc, _ := json.Marshal(owner)
	line = append(line, enc...)
	line = append(line, `,"diag":`...)
	line = append(line, data...)
	line = append(line, '\n')
	_, _ = f.Write(line)
}
