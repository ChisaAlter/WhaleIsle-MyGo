package harness

import (
	"context"
	"path/filepath"
	"strings"
	"whaleisle/internal/petwatch"
)

// PetEventSource is derived from the actual running process environment, never
// the currently edited settings or a conventional ~/.dsh fallback. It carries
// no Host cookie, token, model setting or other credential to the tail worker.
type PetEventSource struct {
	Generation         int
	Home, Node, Origin string
}

func (s *Supervisor) PetEventSource() (PetEventSource, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.State != Ready || s.cmd == nil || s.status.Generation != s.gen {
		return PetEventSource{}, false
	}
	var home string
	for _, entry := range s.cmd.Env {
		key, value, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, "DSH_HOME") {
			home = value
		}
	}
	if home == "" || !filepath.IsAbs(home) {
		return PetEventSource{}, false
	}
	return PetEventSource{Generation: s.gen, Home: filepath.Clean(home), Node: s.opts.Launch.NodeBin, Origin: s.status.Endpoint.Origin}, true
}

func WatchPetEvents(ctx context.Context, source PetEventSource, ack <-chan string, report func(petwatch.Snapshot)) error {
	bin, err := resolveNode(source.Node)
	if err != nil {
		return err
	}
	return petwatch.Run(ctx, bin, source.Home, ack, report)
}
