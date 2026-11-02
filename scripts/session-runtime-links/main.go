// The installer runs its own embedded helper from $PLUGINSDIR, never an old
// installation executable. The graph stores only vendor-relative targets.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type linkSpec struct {
	Target string `json:"target"`
	Kind   string `json:"kind"`
}
type graph struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Files         map[string]string   `json:"files"`
	Links         map[string]linkSpec `json:"links"`
}

func within(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && !filepath.IsAbs(relative) && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
func child(root, relative string) (string, error) {
	if relative == "" || strings.ContainsAny(relative, "\\:<>\"|?*\r\n\x00") || filepath.IsAbs(relative) {
		return "", errors.New("unsafe runtime graph path")
	}
	for _, part := range strings.Split(relative, "/") {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return "", errors.New("unsafe runtime graph component")
		}
		name := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if name == "CON" || name == "PRN" || name == "AUX" || name == "NUL" || len(name) == 4 && (strings.HasPrefix(name, "COM") || strings.HasPrefix(name, "LPT")) && name[3] >= '1' && name[3] <= '9' {
			return "", errors.New("reserved runtime graph component")
		}
	}
	result := filepath.Join(root, filepath.FromSlash(relative))
	if result == root || !within(root, result) {
		return "", errors.New("runtime graph path escapes root")
	}
	return result, nil
}
func isLink(info os.FileInfo) bool { return info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 }
func ordinaryParents(root, filename string, include bool) error {
	return checkParents(root, filename, include, false)
}
func checkParents(root, filename string, include, allowMissing bool) error {
	current := filepath.Dir(filename)
	if include {
		current = filename
	}
	for {
		if !within(root, current) {
			return errors.New("runtime ancestor escapes root")
		}
		info, err := os.Lstat(current)
		if err != nil && !(allowMissing && os.IsNotExist(err)) {
			return err
		}
		if err == nil {
			_, reparse, tagErr := runtimeReparseTag(current)
			if tagErr != nil {
				return tagErr
			}
			if reparse || !info.IsDir() {
				return fmt.Errorf("redirected runtime ancestor: %s", current)
			}
		}
		if strings.EqualFold(current, root) {
			return nil
		}
		current = filepath.Dir(current)
	}
}
func load(root string) (graph, error) {
	var g graph
	absolute, err := filepath.Abs(root)
	if err != nil {
		return g, err
	}
	root = absolute
	if err = ordinaryParents(filepath.VolumeName(root)+string(filepath.Separator), root, true); err != nil {
		return g, err
	}
	filename := filepath.Join(root, "runtime-graph.json")
	info, err := os.Lstat(filename)
	if err != nil {
		return g, err
	}
	_, reparse, tagErr := runtimeReparseTag(filename)
	if tagErr != nil {
		return g, tagErr
	}
	if reparse || !info.Mode().IsRegular() || info.Size() > 64<<20 {
		return g, errors.New("invalid runtime graph file")
	}
	file, err := os.Open(filename)
	if err != nil {
		return g, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&g); err != nil {
		return g, err
	}
	if g.SchemaVersion != 1 || g.Files == nil || g.Links == nil || len(g.Files) > 500000 || len(g.Links) > 50000 {
		return g, errors.New("unsupported runtime graph")
	}
	seen := map[string]bool{}
	for name, digest := range g.Files {
		if _, err = child(root, name); err != nil {
			return g, err
		}
		if seen[strings.ToLower(name)] {
			return g, errors.New("duplicate runtime graph path")
		}
		seen[strings.ToLower(name)] = true
		bytes, e := hex.DecodeString(digest)
		if e != nil || len(bytes) != 32 {
			return g, errors.New("invalid runtime graph hash")
		}
	}
	for name, spec := range g.Links {
		origin, e := child(root, name)
		if e != nil {
			return g, e
		}
		target, e := child(root, spec.Target)
		if e != nil {
			return g, e
		}
		if spec.Kind != "directory" && spec.Kind != "file" || within(target, origin) || within(origin, target) {
			return g, errors.New("invalid self/ancestor runtime link")
		}
		if seen[strings.ToLower(name)] {
			return g, errors.New("duplicate runtime graph path")
		}
		seen[strings.ToLower(name)] = true
	}
	for name := range g.Links {
		for parent := filepath.ToSlash(filepath.Dir(filepath.FromSlash(name))); parent != "."; parent = filepath.ToSlash(filepath.Dir(filepath.FromSlash(parent))) {
			if _, ok := g.Links[parent]; ok {
				return g, errors.New("link has redirected declared ancestor")
			}
		}
	}
	return g, nil
}
func digest(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func preflight(root string, g graph, full, requireLinks bool) error {
	for name, spec := range g.Links {
		origin, _ := child(root, name)
		target, _ := child(root, spec.Target)
		if err := checkParents(root, origin, false, !full); err != nil {
			return err
		}
		existing, originErr := os.Lstat(origin)
		if originErr != nil && !os.IsNotExist(originErr) {
			return originErr
		}
		if !full && os.IsNotExist(originErr) {
			if err := checkParents(root, target, false, true); err != nil {
				return err
			}
			continue
		}
		if err := ordinaryParents(root, target, false); err != nil {
			return err
		}
		info, err := os.Lstat(target)
		if err != nil {
			return err
		}
		_, reparse, tagErr := runtimeReparseTag(target)
		if tagErr != nil {
			return tagErr
		}
		if reparse || (spec.Kind == "directory") != info.IsDir() {
			return fmt.Errorf("runtime link target is not physical: %s", name)
		}
		err = originErr
		if os.IsNotExist(err) {
			if requireLinks {
				return fmt.Errorf("runtime link missing: %s", name)
			}
			continue
		}
		if err != nil {
			return err
		}
		tag, reparse, tagErr := runtimeReparseTag(origin)
		if tagErr != nil {
			return tagErr
		}
		if spec.Kind == "file" {
			if reparse || !existing.Mode().IsRegular() {
				return errors.New("invalid runtime file alias")
			}
			same, err := os.Stat(target)
			if err != nil || !os.SameFile(existing, same) {
				return errors.New("runtime file alias changed")
			}
			continue
		}
		if !reparse || tag != 0xA0000003 {
			return fmt.Errorf("refuse non-junction runtime alias: %s", name)
		}
		resolved, err := resolveDirectoryLink(origin)
		if err != nil || !strings.EqualFold(resolved, target) {
			return fmt.Errorf("runtime link target changed: %s resolved=%q expected=%q: %v", name, resolved, target, err)
		}
	}
	// Scan physical entries only. An unlisted link blocks the entire operation
	// before even one declared junction is removed or created.
	err := filepath.WalkDir(root, func(filename string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filename == root {
			return nil
		}
		name, _ := filepath.Rel(root, filename)
		name = filepath.ToSlash(name)
		info, err := os.Lstat(filename)
		if err != nil {
			return err
		}
		_, reparse, tagErr := runtimeReparseTag(filename)
		if tagErr != nil {
			return tagErr
		}
		if reparse {
			if _, ok := g.Links[name]; !ok {
				return fmt.Errorf("unknown runtime reparse point: %s", name)
			}
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported runtime entry: %s", name)
		}
		if info.Mode().IsRegular() && full && name != "runtime-graph.json" && name != "dshd-remote/debug.log" {
			if alias, ok := g.Links[name]; ok && alias.Kind == "file" {
				return nil
			}
			expected, ok := g.Files[name]
			if !ok {
				return fmt.Errorf("untracked runtime file: %s", name)
			}
			actual, err := digest(filename)
			if err != nil || actual != expected {
				return fmt.Errorf("runtime file changed: %s", name)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if full {
		for name, expected := range g.Files {
			filename, _ := child(root, name)
			if err := ordinaryParents(root, filename, false); err != nil {
				return err
			}
			actual, err := digest(filename)
			if err != nil || actual != expected {
				return fmt.Errorf("runtime file missing or changed: %s", name)
			}
		}
	}
	return nil
}
func operate(command, root string) error {
	if strings.HasPrefix(root, `\\?\UNC\`) {
		root = `\\` + strings.TrimPrefix(root, `\\?\UNC\`)
	} else {
		root = strings.TrimPrefix(root, `\\?\`)
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	g, err := load(root)
	if err != nil {
		return err
	}
	if err = preflight(root, g, command == "verify" || command == "restore" || command == "check", command == "verify"); err != nil {
		return err
	}
	if command == "verify" || command == "check" {
		return nil
	}
	if command != "restore" && command != "detach" {
		return errors.New("expected verify, check, restore or detach")
	}
	names := make([]string, 0, len(g.Links))
	for name := range g.Links {
		names = append(names, name)
	}
	sort.Strings(names)
	created := []string{}
	detached := []string{}
	restoreDetached := func() error {
		for _, previous := range detached {
			spec := g.Links[previous]
			origin, _ := child(root, previous)
			target, _ := child(root, spec.Target)
			var restoreErr error
			if spec.Kind == "directory" {
				restoreErr = createDirectoryLink(target, origin)
			} else {
				restoreErr = os.Link(target, origin)
			}
			if restoreErr != nil {
				return fmt.Errorf("cannot restore detached runtime link %s: %w", previous, restoreErr)
			}
		}
		return nil
	}
	for _, name := range names {
		spec := g.Links[name]
		origin, _ := child(root, name)
		target, _ := child(root, spec.Target)
		_, err := os.Lstat(origin)
		if command == "detach" {
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return errors.Join(err, restoreDetached())
			}
			if spec.Kind == "directory" {
				err = removeDirectoryLink(origin)
			} else {
				err = os.Remove(origin)
			}
			if err != nil {
				return errors.Join(err, restoreDetached())
			}
			detached = append(detached, name)
			continue
		}
		if err == nil {
			continue
		}
		if !os.IsNotExist(err) {
			return err
		}
		if spec.Kind == "directory" {
			err = createDirectoryLink(target, origin)
		} else {
			err = os.Link(target, origin)
		}
		if err != nil {
			for i := len(created) - 1; i >= 0; i-- {
				previous := created[i]
				if g.Links[previous].Kind == "directory" {
					_ = removeDirectoryLink(filepath.Join(root, filepath.FromSlash(previous)))
				} else {
					_ = os.Remove(filepath.Join(root, filepath.FromSlash(previous)))
				}
			}
			return err
		}
		created = append(created, name)
	}
	return nil
}
func main() {
	if len(os.Args) != 3 {
		os.Exit(2)
	}
	if err := operate(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
