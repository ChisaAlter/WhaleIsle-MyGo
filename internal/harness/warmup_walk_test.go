package harness

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"testing"
)

// collectWarmPathsBaseline preserves the pre-task-37 collector, with only a
// Readlink call counter added. The exact production file is also saved in tmp.
func collectWarmPathsBaseline(root string, cancel chan struct{}) ([]string, int) {
	var paths []string
	readlinks := 0
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		if depth > 24 {
			return
		}
		select {
		case <-cancel:
			return
		default:
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			name := entry.Name()
			path := filepath.Join(dir, name)
			if entry.IsDir() {
				readlinks++
				if _, err := os.Readlink(path); err == nil {
					continue
				}
				if strings.EqualFold(name, ".git") || strings.EqualFold(name, ".cache") {
					continue
				}
				walk(path, depth+1)
				continue
			}
			if !warmExtensions[strings.ToLower(filepath.Ext(name))] {
				continue
			}
			paths = append(paths, path)
		}
	}
	walk(root, 0)
	return paths, readlinks
}

func warmWalkWrite(tb testing.TB, root, relative string) string {
	tb.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fixture module\n"), 0o644); err != nil {
		tb.Fatal(err)
	}
	return path
}

func warmWalkWant(tb testing.TB, got, want []string) {
	tb.Helper()
	got = append([]string(nil), got...)
	want = append([]string(nil), want...)
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		tb.Fatalf("collected paths:\n got %q\nwant %q", got, want)
	}
}

func warmWalkEntry(tb testing.TB, dir, name string) os.DirEntry {
	tb.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		tb.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == name {
			return entry
		}
	}
	tb.Fatalf("missing directory entry %q", name)
	return nil
}

// The caller supplies only its own t.TempDir paths. Windows permits junctions
// without symbolic-link privilege; other systems use a directory symlink.
func warmWalkDirectoryLink(t *testing.T, target, link string) {
	t.Helper()
	if err := symlinkDir(target, link); err != nil {
		t.Skipf("temporary directory-link fixture unavailable: %v", err)
	}
}

func TestCollectWarmPathsPlainBaselineAndBounds(t *testing.T) {
	root := t.TempDir()
	var want []string
	for _, relative := range []string{
		"entry.Js", "config.JSON", "native.node", "image.wasm", "shared.mjs",
		"nested/a.cjs", "nested/b.yml", "nested/deeper/c.yaml", "directory.js/inside.js",
		".gitlike/keep.js", ".cacheable/keep.json",
	} {
		want = append(want, warmWalkWrite(t, root, relative))
	}
	for _, relative := range []string{
		"README", "types/index.d.ts", "nested/app.js.map", ".GIT/ignored.js", ".CaChE/ignored.json",
	} {
		warmWalkWrite(t, root, relative)
	}
	deep := ""
	for depth := 1; depth <= 25; depth++ {
		deep = filepath.Join(deep, "d")
		path := warmWalkWrite(t, root, filepath.Join(deep, "depth.js"))
		if depth <= 24 {
			want = append(want, path)
		}
	}
	old, calls := collectWarmPathsBaseline(root, nil)
	got := collectWarmPaths(root, nil)
	warmWalkWant(t, got, want)
	warmWalkWant(t, got, old)
	if !reflect.DeepEqual(got, old) {
		t.Fatal("plain-file traversal order changed")
	}
	if calls == 0 {
		t.Fatal("baseline fixture did not exercise directory Readlink probes")
	}
	for _, path := range got {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("admitted non-regular file %s: %v %v", path, info, err)
		}
	}
	t.Logf("plain-file paths and order identical: %d; baseline directory Readlink calls: %d; new: 0", len(got), calls)
}

func TestCollectWarmPathsCancelledAndMissing(t *testing.T) {
	root := t.TempDir()
	warmWalkWrite(t, root, "nested/entry.js")
	cancel := make(chan struct{})
	close(cancel)
	if got := collectWarmPaths(root, cancel); len(got) != 0 {
		t.Fatalf("cancelled walk collected %q", got)
	}
	old, calls := collectWarmPathsBaseline(root, cancel)
	if len(old) != 0 || calls != 0 {
		t.Fatalf("cancelled baseline collected %q with %d probes", old, calls)
	}
	for _, missing := range []string{filepath.Join(root, "missing"), filepath.Join(root, "nested", "entry.js")} {
		if got := collectWarmPaths(missing, nil); len(got) != 0 {
			t.Fatalf("unreadable directory %s collected %q", missing, got)
		}
	}
}

func TestCollectWarmPathsDirectoryLinkBaseline(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "tree")
	plain := warmWalkWrite(t, root, "nested/inside.js")
	outside := warmWalkWrite(t, filepath.Join(base, "outside"), "outside.js")
	warmWalkDirectoryLink(t, filepath.Dir(outside), filepath.Join(root, "alias"))
	warmWalkDirectoryLink(t, root, filepath.Join(root, "cycle"))
	old, _ := collectWarmPathsBaseline(root, nil)
	got := collectWarmPaths(root, nil)
	warmWalkWant(t, got, []string{plain})
	warmWalkWant(t, got, old)
	entry := warmWalkEntry(t, root, "alias")
	if entry.IsDir() || entry.Type()&(os.ModeSymlink|os.ModeIrregular) == 0 {
		t.Fatalf("unexpected directory-link type: Type=%v IsDir=%v", entry.Type(), entry.IsDir())
	}
	t.Logf("actual directory link and cycle excluded; Type=%v; valid-file set unchanged", entry.Type())
}

func TestCollectWarmPathsMatchingSuffixDirectoryLinkDelta(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "tree")
	plain := warmWalkWrite(t, root, "inside.js")
	outside := warmWalkWrite(t, filepath.Join(base, "outside"), "outside.js")
	var excluded []string
	for _, name := range []string{"alias.js", "cycle.json"} {
		target := filepath.Dir(outside)
		if name == "cycle.json" {
			target = root
		}
		link := filepath.Join(root, name)
		warmWalkDirectoryLink(t, target, link)
		excluded = append(excluded, link)
	}
	old, _ := collectWarmPathsBaseline(root, nil)
	got := collectWarmPaths(root, nil)
	warmWalkWant(t, got, []string{plain})
	warmWalkWant(t, old, append([]string{plain}, excluded...))
	for _, path := range excluded {
		entry := warmWalkEntry(t, root, filepath.Base(path))
		if entry.Type() == 0 || entry.Type() == os.ModeDir {
			t.Fatalf("matching-suffix directory link was reported as regular: %v", entry.Type())
		}
		t.Logf("intentional old-new exclusion: %s Type=%v", filepath.Base(path), entry.Type())
	}
}

func TestCollectWarmPathsFileSymlinkDelta(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "tree")
	plain := warmWalkWrite(t, root, "inside.js")
	outside := warmWalkWrite(t, filepath.Join(base, "outside"), "outside.js")
	var excluded []string
	for _, fixture := range []struct{ name, target string }{
		{"linked.js", outside},
		{"relative.cjs", filepath.Join("..", "outside", "outside.js")},
		{"broken.json", filepath.Join(base, "missing.js")},
		{"ignored.txt", outside},
	} {
		link := filepath.Join(root, fixture.name)
		if err := os.Symlink(fixture.target, link); err != nil {
			if errors.Is(err, os.ErrPermission) || (runtime.GOOS == "windows" && errors.Is(err, syscall.Errno(1314))) {
				t.Skipf("temporary symbolic-link fixture unavailable: %v", err)
			}
			t.Fatal(err)
		}
		if warmExtensions[filepath.Ext(fixture.name)] {
			excluded = append(excluded, link)
		}
		entry := warmWalkEntry(t, root, fixture.name)
		if entry.IsDir() || entry.Type()&os.ModeSymlink == 0 {
			t.Fatalf("file symlink type: Type=%v IsDir=%v", entry.Type(), entry.IsDir())
		}
	}
	old, _ := collectWarmPathsBaseline(root, nil)
	got := collectWarmPaths(root, nil)
	warmWalkWant(t, got, []string{plain})
	warmWalkWant(t, old, append([]string{plain}, excluded...))
	// Demonstrate the old plan's matching-suffix link would read the external
	// target, using only a harmless file inside this test's own temporary tree.
	if n, err := readThrough(filepath.Join(root, "linked.js"), make([]byte, 64)); err != nil || n == 0 {
		t.Fatalf("baseline symlink-follow demonstration: bytes=%d err=%v", n, err)
	}
	t.Logf("intentional old-new exclusions (only matching-suffix symlinks): %q", excluded)
}

// warmSamplePathsBaseline is the original sampler with no type guard.
func warmSamplePathsBaseline(vendorDir string) []string {
	var paths []string
	add := func(dir string, max int) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		taken := 0
		for _, entry := range entries {
			if taken >= max || len(paths) >= 12 {
				return
			}
			if entry.IsDir() || !warmExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
				continue
			}
			paths = append(paths, filepath.Join(dir, entry.Name()))
			taken++
		}
	}
	harness := filepath.Join(vendorDir, "deepseek-harness")
	add(filepath.Join(harness, "apps", "cli", "lib"), 3)
	add(filepath.Join(harness, "apps", "web", "dist"), 2)
	add(filepath.Join(harness, "packages", "boot"), 2)
	instances, err := os.ReadDir(filepath.Join(harness, "node_modules", ".dsh-instances"))
	if err == nil && len(instances) > 0 {
		step := len(instances) / 4
		if step < 1 {
			step = 1
		}
		for i := 0; i < len(instances) && len(paths) < 12; i += step {
			if !instances[i].IsDir() {
				continue
			}
			dir := filepath.Join(harness, "node_modules", ".dsh-instances", instances[i].Name())
			add(dir, 1)
			add(filepath.Join(dir, "lib"), 1)
		}
	}
	return paths
}

func TestWarmSamplePathsPlainBaselineAndLimits(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		"apps/cli/lib", "apps/web/dist", "packages/boot",
		"node_modules/.dsh-instances/a", "node_modules/.dsh-instances/a/lib",
		"node_modules/.dsh-instances/b", "node_modules/.dsh-instances/b/lib",
		"node_modules/.dsh-instances/c", "node_modules/.dsh-instances/c/lib",
	} {
		for i := 0; i < 4; i++ {
			warmWalkWrite(t, root, filepath.Join("deepseek-harness", dir, fmt.Sprintf("%d.js", i)))
		}
		warmWalkWrite(t, root, filepath.Join("deepseek-harness", dir, "no.txt"))
	}
	want := warmSamplePathsBaseline(root)
	got := warmSamplePaths(root)
	if !reflect.DeepEqual(got, want) || len(got) != 12 {
		t.Fatalf("plain sampler order/set/limit changed:\n got %q\nwant %q", got, want)
	}
}

func TestWarmSamplePathsDirectoryLinkDelta(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "vendor")
	dir := filepath.Join(root, "deepseek-harness", "apps", "cli", "lib")
	plain := warmWalkWrite(t, dir, "plain.js")
	outside := warmWalkWrite(t, filepath.Join(base, "outside"), "outside.js")
	link := filepath.Join(dir, "alias.json")
	warmWalkDirectoryLink(t, filepath.Dir(outside), link)
	warmWalkWant(t, warmSamplePaths(root), []string{plain})
	warmWalkWant(t, warmSamplePathsBaseline(root), []string{link, plain})
}

func TestWarmSamplePathsFileSymlinkDelta(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "vendor")
	dir := filepath.Join(root, "deepseek-harness", "apps", "cli", "lib")
	plain := warmWalkWrite(t, dir, "plain.js")
	outside := warmWalkWrite(t, filepath.Join(base, "outside"), "outside.js")
	var excluded []string
	for _, name := range []string{"alias.js", "broken.json"} {
		link := filepath.Join(dir, name)
		target := outside
		if name == "broken.json" {
			target = filepath.Join(base, "missing.js")
		}
		if err := os.Symlink(target, link); err != nil {
			if errors.Is(err, os.ErrPermission) || (runtime.GOOS == "windows" && errors.Is(err, syscall.Errno(1314))) {
				t.Skipf("temporary symbolic-link fixture unavailable: %v", err)
			}
			t.Fatal(err)
		}
		excluded = append(excluded, link)
	}
	warmWalkWant(t, warmSamplePaths(root), []string{plain})
	warmWalkWant(t, warmSamplePathsBaseline(root), append([]string{plain}, excluded...))
}

func TestWarmReadThroughSizeLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.js")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(warmMaxFile + 1); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if n, err := readThrough(path, make([]byte, 64)); n != 0 || !errors.Is(err, errWarmSkip) {
		t.Fatalf("oversized file read: bytes=%d err=%v", n, err)
	}
}

// This benchmark measures traversal of one cached synthetic fixture, not Node
// startup, cold-cache behavior, or performance on the running runtime tree.
func BenchmarkCollectWarmPathsDirectoryProbes(b *testing.B) {
	root := b.TempDir()
	for i := 0; i < 96; i++ {
		warmWalkWrite(b, root, fmt.Sprintf("pkg-%03d/lib/index.js", i))
		warmWalkWrite(b, root, fmt.Sprintf("pkg-%03d/lib/config.json", i))
	}
	old, probes := collectWarmPathsBaseline(root, nil)
	got := collectWarmPaths(root, nil)
	warmWalkWant(b, got, old)
	b.Run("baseline", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			paths, _ := collectWarmPathsBaseline(root, nil)
			if len(paths) != len(old) {
				b.Fatal("baseline path count changed")
			}
		}
		b.ReportMetric(float64(probes), "readlink/op")
	})
	b.Run("entry-type", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if paths := collectWarmPaths(root, nil); len(paths) != len(old) {
				b.Fatal("new path count changed")
			}
		}
		b.ReportMetric(0, "readlink/op")
	})
}
