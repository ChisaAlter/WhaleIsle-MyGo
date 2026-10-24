package editors

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fakeProbe(goos string, env map[string]string, onPath map[string]string, files map[string]bool) Probe {
	normalized := make(map[string]bool, len(files))
	for name, present := range files {
		normalized[cleanTestPath(name)] = present
	}
	return Probe{
		GOOS: goos,
		Env: func(name string) string {
			return env[strings.ToUpper(name)]
		},
		LookPath: func(name string) (string, bool) {
			path, ok := onPath[name]
			return path, ok
		},
		Exists: func(path string) bool { return normalized[cleanTestPath(path)] },
		Glob: func(pattern string) ([]string, error) {
			pattern = strings.ToLower(filepath.ToSlash(pattern))
			var matches []string
			for name := range normalized {
				if ok, _ := filepath.Match(pattern, strings.ToLower(filepath.ToSlash(name))); ok {
					matches = append(matches, name)
				}
			}
			return matches, nil
		},
	}
}

// cleanTestPath normalizes a fixture path so one test can name both Unix and
// Windows install locations on any host. filepath.Clean alone is host
// dependent: on Windows it rewrites `/Applications/...` to `\Applications\...`,
// which would stop matching a forward-slash fixture key.
func cleanTestPath(path string) string {
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
}

func TestAllMatchesElectronEditorList(t *testing.T) {
	all := All()
	if len(all) != 20 {
		t.Fatalf("editor count = %d, want 20", len(all))
	}
	seen := map[string]bool{}
	for _, editor := range all {
		if editor.ID == "" || editor.Label == "" || len(editor.Commands) == 0 {
			t.Fatalf("incomplete editor definition: %+v", editor)
		}
		if seen[editor.ID] {
			t.Fatalf("duplicate editor id: %s", editor.ID)
		}
		seen[editor.ID] = true
		if len(editor.WindowsPaths) == 0 || len(editor.UnixPaths) == 0 {
			t.Fatalf("editor %s has no install paths", editor.ID)
		}
	}
	want := []string{"cursor", "trae", "kiro", "vscode", "vscode-insiders", "vscodium", "zed", "antigravity",
		"idea", "aqua", "clion", "datagrip", "dataspell", "goland", "phpstorm", "pycharm", "rider", "rubymine", "rustrover", "webstorm"}
	got := make([]string, 0, len(all))
	for _, editor := range all {
		got = append(got, editor.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("editor order = %v, want %v", got, want)
	}
}

func TestArgsGotoLineColumnAndDirectPath(t *testing.T) {
	cursor, _ := Find("cursor")
	if got, want := Args(cursor, `C:\w\a.go`, 7, 3), []string{"--goto", `C:\w\a.go:7:3`}; !reflect.DeepEqual(got, want) {
		t.Fatalf("goto args = %v, want %v", got, want)
	}
	if got, want := Args(cursor, `C:\w\a.go`, 7, 0), []string{"--goto", `C:\w\a.go:7:1`}; !reflect.DeepEqual(got, want) {
		t.Fatalf("goto without column = %v, want %v", got, want)
	}
	if got, want := Args(cursor, `C:\w\a.go`, 0, 0), []string{`C:\w\a.go`}; !reflect.DeepEqual(got, want) {
		t.Fatalf("goto without line = %v, want %v", got, want)
	}

	kiro, _ := Find("kiro")
	if got, want := Args(kiro, "/w/a.go", 2, 5), []string{"ide", "--goto", "/w/a.go:2:5"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("base-arg editor = %v, want %v", got, want)
	}

	goland, _ := Find("goland")
	if got, want := Args(goland, "/w/a.go", 4, 2), []string{"--line", "4", "--column", "2", "/w/a.go"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("line-column args = %v, want %v", got, want)
	}
	if got, want := Args(goland, "/w/a.go", 4, 0), []string{"--line", "4", "/w/a.go"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("line without column = %v, want %v", got, want)
	}
	if got, want := Args(goland, "/w/a.go", 0, 0), []string{"/w/a.go"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("line-column without line = %v, want %v", got, want)
	}

	zed, _ := Find("zed")
	if got, want := Args(zed, "/w/a.go", 9, 9), []string{"/w/a.go"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("direct-path args = %v, want %v", got, want)
	}
}

func TestLocatePrefersExecutableOverBatchShim(t *testing.T) {
	probe := fakeProbe("windows", nil, map[string]string{
		"code":     `C:\bin\code.cmd`,
		"code.exe": `C:\bin\Code.exe`,
	}, nil)
	vscode, _ := Find("vscode")
	if got, ok := Locate(probe, vscode); !ok || got != `C:\bin\Code.exe` {
		t.Fatalf("locate = %q, %v; want the .exe over the .cmd shim", got, ok)
	}
}

func TestLocateSkipsBatchShimWithoutExecutable(t *testing.T) {
	probe := fakeProbe("windows", nil, map[string]string{"codium": `C:\bin\codium.cmd`}, nil)
	codium, _ := Find("vscodium")
	if got, ok := Locate(probe, codium); ok {
		t.Fatalf("locate = %q; batch shims must not be spawned directly", got)
	}
}

func TestLocateExpandsInstallPathsAndGlobsNewestChannel(t *testing.T) {
	files := map[string]bool{`C:\Users\me\AppData\Local\JetBrains\Toolbox\apps\PY\ch-0\243.1\bin\pycharm64.exe`: true}
	probe := fakeProbe("windows", map[string]string{"LOCALAPPDATA": `C:\Users\me\AppData\Local`}, nil, files)
	pycharm, _ := Find("pycharm")
	if got, ok := Locate(probe, pycharm); !ok || !strings.HasSuffix(got, `pycharm64.exe`) {
		t.Fatalf("locate = %q, %v; want the Toolbox install", got, ok)
	}
	matches, err := probe.Glob(filepath.ToSlash(`C:\Users\me\AppData\Local\JetBrains\Toolbox\apps\PY\ch-*\*\bin\pycharm64.exe`))
	if err != nil {
		t.Fatalf("toolbox glob error: %v", err)
	}
	if len(matches) == 0 || !strings.Contains(strings.ToLower(matches[0]), "pycharm64") {
		t.Fatalf("toolbox glob = %v; want the pycharm64 launcher", matches)
	}
}

func TestLocateUnixInstallPath(t *testing.T) {
	probe := fakeProbe("darwin", nil, nil, map[string]bool{"/Applications/Cursor.app/Contents/MacOS/Cursor": true})
	cursor, _ := Find("cursor")
	if got, ok := Locate(probe, cursor); !ok || got != "/Applications/Cursor.app/Contents/MacOS/Cursor" {
		t.Fatalf("locate = %q, %v", got, ok)
	}
}

func TestAvailableKeepsDefinitionOrder(t *testing.T) {
	probe := fakeProbe("linux", nil, map[string]string{
		"cursor":   "/usr/bin/cursor",
		"webstorm": "/usr/bin/webstorm",
	}, nil)
	got := Available(probe)
	if len(got) != 2 || got[0].ID != "cursor" || got[1].ID != "webstorm" {
		t.Fatalf("available = %+v, want cursor then webstorm", got)
	}
}

func TestOpenWithRejectsUnknownAndMissingEditor(t *testing.T) {
	probe := fakeProbe("linux", nil, nil, nil)
	// ValidatePath only accepts host-absolute targets, so build one instead of
	// a Unix literal that is relative on Windows.
	abs := filepath.Join(t.TempDir(), "a.go")
	if err := OpenWith(probe, "nope", abs, 1, 0); !errors.Is(err, ErrUnknownEditor) {
		t.Fatalf("unknown editor error = %v", err)
	}
	if err := OpenWith(probe, "vscode", abs, 1, 0); !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("missing editor error = %v, want ErrNotInstalled", err)
	}
	if err := OpenWith(probe, "vscode", "relative/a.go", 1, 0); err == nil {
		t.Fatal("relative path must be rejected before any spawn")
	}
	if err := OpenWith(probe, "vscode", "", 1, 0); err == nil {
		t.Fatal("empty path must be rejected before any spawn")
	}
}
