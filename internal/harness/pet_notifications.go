package harness

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"whaleisle/internal/petwatch"
)

// PetEventSource is derived from the actual running process environment, never
// the currently edited settings or a conventional ~/.dsh fallback. It carries
// no Host cookie, token, model setting or other credential to the tail worker.
type PetEventSource struct {
	Generation                      int
	Home, Node, Origin, StickerRoot string
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
	return PetEventSource{Generation: s.gen, Home: filepath.Clean(home), Node: s.opts.Launch.NodeBin, Origin: s.status.Endpoint.Origin, StickerRoot: s.petStickerRootLocked()}, true
}

func WatchPetEvents(ctx context.Context, source PetEventSource, ack <-chan string, report func(petwatch.Snapshot)) error {
	bin, err := resolveNode(source.Node)
	if err != nil {
		return err
	}
	return petwatch.Run(ctx, bin, source.Home, source.StickerRoot, ack, report)
}

// Resolve the bundled sticker pack from the exact desktop-managed plugin patch
// present in this running command. Next-launch settings and conventional vendor
// paths are not authority for accessing a different package's files.
func (s *Supervisor) petStickerRootLocked() string {
	if s.attemptRecovery || s.cmd == nil {
		return ""
	}
	for i, arg := range s.cmd.Args {
		if arg != "--patch" || i+1 >= len(s.cmd.Args) {
			continue
		}
		patch := s.cmd.Args[i+1]
		if !filepath.IsAbs(patch) || filepath.Base(patch) != "desktop-dsh-whale.patch.yml" || filepath.Base(filepath.Dir(patch)) != "dsh-whale" {
			continue
		}
		raw, err := os.ReadFile(patch)
		if err != nil || string(raw) != overlayYAML("dsh-whale", "dsh-whale") {
			continue
		}
		profile := filepath.Dir(filepath.Dir(filepath.Dir(patch)))
		packageRoot, err := filepath.EvalSymlinks(filepath.Join(profile, "node_modules", "dsh-whale"))
		if err != nil {
			continue
		}
		root := filepath.Join(packageRoot, "stickers")
		info, err := os.Lstat(root)
		if err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			return root
		}
	}
	return ""
}
