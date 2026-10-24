// Package editors ports the Electron shell's external-editor launcher
// (reference/electron/src/main/editors.js): the same probed editor list, the
// same argument assembly (`--goto path:line:col`, JetBrains `--line/--column`)
// and a caller-side fallback to the system default handler when no editor is
// installed.
package editors

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// ErrUnknownEditor reports an editor id that is not part of the probed list.
var ErrUnknownEditor = errors.New("unknown editor")

// ErrNotInstalled reports a known editor that could not be found on PATH or in
// its common install locations.
var ErrNotInstalled = errors.New("editor is not installed")

// Style is how an editor receives a workspace file position.
type Style int

const (
	// StyleGoto passes `--goto path:line:column` (VS Code family).
	StyleGoto Style = iota
	// StyleLineColumn passes `--line N [--column N] path` (JetBrains family).
	StyleLineColumn
	// StyleDirectPath passes only the path.
	StyleDirectPath
)

// Editor is one launchable editor definition. WindowsPaths and UnixPaths are
// probed after PATH; `%VAR%` names are expanded and `*` segments are globbed.
type Editor struct {
	ID           string
	Label        string
	Commands     []string
	BaseArgs     []string
	Style        Style
	WindowsPaths []string
	UnixPaths    []string
}

// All is the Electron editors list (file manager excluded: the shell already
// reveals paths through desktop/components.OpenInFileManager).
func All() []Editor {
	return []Editor{
		{ID: "cursor", Label: "Cursor", Commands: []string{"cursor"}, Style: StyleGoto,
			WindowsPaths: []string{`%LOCALAPPDATA%\Programs\cursor\Cursor.exe`, `%PROGRAMFILES%\Cursor\Cursor.exe`},
			UnixPaths:    []string{"/Applications/Cursor.app/Contents/MacOS/Cursor", "/usr/bin/cursor", "/opt/Cursor/cursor"}},
		{ID: "trae", Label: "Trae", Commands: []string{"trae"}, Style: StyleGoto,
			WindowsPaths: []string{`%LOCALAPPDATA%\Programs\Trae\Trae.exe`, `%PROGRAMFILES%\Trae\Trae.exe`},
			UnixPaths:    []string{"/Applications/Trae.app/Contents/MacOS/Trae", "/usr/bin/trae"}},
		{ID: "kiro", Label: "Kiro", Commands: []string{"kiro"}, BaseArgs: []string{"ide"}, Style: StyleGoto,
			WindowsPaths: []string{`%LOCALAPPDATA%\Programs\Kiro\Kiro.exe`, `%PROGRAMFILES%\Kiro\Kiro.exe`},
			UnixPaths:    []string{"/Applications/Kiro.app/Contents/MacOS/Kiro", "/usr/bin/kiro"}},
		{ID: "vscode", Label: "VS Code", Commands: []string{"code"}, Style: StyleGoto,
			WindowsPaths: []string{`%LOCALAPPDATA%\Programs\Microsoft VS Code\Code.exe`, `%PROGRAMFILES%\Microsoft VS Code\Code.exe`},
			UnixPaths:    []string{"/Applications/Visual Studio Code.app/Contents/MacOS/Electron", "/usr/bin/code", "/snap/bin/code", "/opt/visual-studio-code/code"}},
		{ID: "vscode-insiders", Label: "VS Code Insiders", Commands: []string{"code-insiders"}, Style: StyleGoto,
			WindowsPaths: []string{`%LOCALAPPDATA%\Programs\Microsoft VS Code Insiders\Code - Insiders.exe`},
			UnixPaths:    []string{"/Applications/Visual Studio Code - Insiders.app/Contents/MacOS/Electron", "/usr/bin/code-insiders", "/snap/bin/code-insiders"}},
		{ID: "vscodium", Label: "VSCodium", Commands: []string{"codium"}, Style: StyleGoto,
			WindowsPaths: []string{`%LOCALAPPDATA%\Programs\VSCodium\VSCodium.exe`, `%PROGRAMFILES%\VSCodium\VSCodium.exe`},
			UnixPaths:    []string{"/Applications/VSCodium.app/Contents/MacOS/Electron", "/usr/bin/codium"}},
		{ID: "zed", Label: "Zed", Commands: []string{"zed", "zeditor"}, Style: StyleDirectPath,
			WindowsPaths: []string{`%LOCALAPPDATA%\Programs\Zed\zed.exe`},
			UnixPaths:    []string{"/Applications/Zed.app/Contents/MacOS/zed", "/usr/bin/zed", "/usr/local/bin/zed"}},
		{ID: "antigravity", Label: "Antigravity", Commands: []string{"agy"}, Style: StyleGoto,
			WindowsPaths: []string{`%LOCALAPPDATA%\Programs\Antigravity\Antigravity.exe`, `%PROGRAMFILES%\Antigravity\Antigravity.exe`},
			UnixPaths:    []string{"/Applications/Antigravity.app/Contents/MacOS/Antigravity", "/usr/bin/agy"}},
		jetBrains("idea", "IntelliJ IDEA", "idea", "IntelliJ IDEA Ultimate", "IntelliJ IDEA Community Edition", "IDEA-U", "IDEA-C", "idea64.exe", "IntelliJ IDEA.app", "idea"),
		jetBrains("aqua", "Aqua", "aqua", "Aqua", "", "AQA", "", "aqua64.exe", "Aqua.app", "aqua"),
		jetBrains("clion", "CLion", "clion", "CLion", "", "CL", "", "clion64.exe", "CLion.app", "clion"),
		jetBrains("datagrip", "DataGrip", "datagrip", "DataGrip", "", "DG", "", "datagrip64.exe", "DataGrip.app", "datagrip"),
		jetBrains("dataspell", "DataSpell", "dataspell", "DataSpell", "", "DS", "", "dataspell64.exe", "DataSpell.app", "dataspell"),
		jetBrains("goland", "GoLand", "goland", "GoLand", "", "GO", "", "goland64.exe", "GoLand.app", "goland"),
		jetBrains("phpstorm", "PhpStorm", "phpstorm", "PhpStorm", "", "PS", "", "phpstorm64.exe", "PhpStorm.app", "phpstorm"),
		jetBrains("pycharm", "PyCharm", "pycharm", "PyCharm Professional Edition", "PyCharm Community Edition", "PY", "", "pycharm64.exe", "PyCharm.app", "pycharm"),
		jetBrains("rider", "Rider", "rider", "Rider", "", "RD", "", "rider64.exe", "Rider.app", "rider"),
		jetBrains("rubymine", "RubyMine", "rubymine", "RubyMine", "", "RM", "", "rubymine64.exe", "RubyMine.app", "rubymine"),
		jetBrains("rustrover", "RustRover", "rustrover", "RustRover", "", "RR", "", "rustrover64.exe", "RustRover.app", "rustrover"),
		jetBrains("webstorm", "WebStorm", "webstorm", "WebStorm", "", "WS", "", "webstorm64.exe", "WebStorm.app", "webstorm"),
	}
}

// jetBrains builds one JetBrains IDE definition: the launcher is a PATH
// command, the common per-user install directory and the Toolbox channel glob.
func jetBrains(id, label, command, full, community, toolbox, toolboxCommunity, exe, bundle, macBinary string) Editor {
	paths := []string{
		`%LOCALAPPDATA%\Programs\` + full + `\bin\` + exe,
		`%LOCALAPPDATA%\Programs\` + full + `\bin\` + strings.TrimSuffix(exe, "64.exe") + ".exe",
		`%LOCALAPPDATA%\JetBrains\Toolbox\apps\` + toolbox + `\ch-*\*\bin\` + exe,
	}
	if community != "" {
		paths = append(paths,
			`%LOCALAPPDATA%\Programs\`+community+`\bin\`+exe,
			`%LOCALAPPDATA%\Programs\`+community+`\bin\`+strings.TrimSuffix(exe, "64.exe")+".exe")
	}
	if toolboxCommunity != "" {
		paths = append(paths, `%LOCALAPPDATA%\JetBrains\Toolbox\apps\`+toolboxCommunity+`\ch-*\*\bin\`+exe)
	}
	return Editor{ID: id, Label: label, Commands: []string{command}, Style: StyleLineColumn, WindowsPaths: paths,
		UnixPaths: []string{
			"/Applications/" + bundle + "/Contents/MacOS/" + macBinary,
			"~/Applications/" + bundle + "/Contents/MacOS/" + macBinary,
			"/usr/bin/" + command,
			"/usr/local/bin/" + command,
			"/snap/bin/" + command,
			"/opt/" + command + "/bin/" + command,
		}}
}

// Find returns the editor with the given id.
func Find(id string) (Editor, bool) {
	for _, editor := range All() {
		if editor.ID == id {
			return editor, true
		}
	}
	return Editor{}, false
}

// Args assembles launch arguments for one file position. It mirrors
// resolveEditorArgs: line is 1-based and a missing column becomes 1 for goto.
func Args(editor Editor, path string, line, column int) []string {
	args := append([]string(nil), editor.BaseArgs...)
	switch editor.Style {
	case StyleGoto:
		if line > 0 {
			if column <= 0 {
				column = 1
			}
			return append(args, "--goto", path+":"+strconv.Itoa(line)+":"+strconv.Itoa(column))
		}
	case StyleLineColumn:
		if line > 0 {
			args = append(args, "--line", strconv.Itoa(line))
			if column > 0 {
				args = append(args, "--column", strconv.Itoa(column))
			}
			return append(args, path)
		}
	}
	return append(args, path)
}

// Probe pins the environment lookups so detection is testable and so tests
// never launch a real editor.
type Probe struct {
	GOOS     string
	Env      func(string) string
	LookPath func(string) (string, bool)
	Exists   func(string) bool
	Glob     func(string) ([]string, error)
}

// DefaultProbe resolves through the real environment.
func DefaultProbe() Probe {
	return Probe{
		GOOS: runtime.GOOS,
		Env:  os.Getenv,
		LookPath: func(name string) (string, bool) {
			path, err := exec.LookPath(name)
			return path, err == nil
		},
		Exists: func(path string) bool {
			info, err := os.Stat(path)
			return err == nil && !info.IsDir()
		},
		Glob: filepath.Glob,
	}
}

// Available lists installed editors in definition order.
func Available(p Probe) []Editor {
	found := make([]Editor, 0, len(All()))
	for _, editor := range All() {
		if _, ok := Locate(p, editor); ok {
			found = append(found, editor)
		}
	}
	return found
}

// Locate resolves the first executable for an editor. On Windows a `.exe` on
// PATH wins over a `.cmd`/`.bat` shim because os/exec cannot spawn a batch file
// directly. Globbed install paths prefer the newest channel directory.
func Locate(p Probe, editor Editor) (string, bool) {
	if p.GOOS == "" {
		p.GOOS = runtime.GOOS
	}
	for _, command := range editor.Commands {
		if p.GOOS == "windows" {
			if path, ok := p.LookPath(command + ".exe"); ok {
				return path, true
			}
		}
		if path, ok := p.LookPath(command); ok && !isBatchShim(path) {
			return path, true
		}
	}
	return locatePaths(p, editor)
}

func locatePaths(p Probe, editor Editor) (string, bool) {
	paths := editor.UnixPaths
	if p.GOOS == "windows" {
		paths = editor.WindowsPaths
	}
	for _, candidate := range paths {
		for _, path := range expandPath(p, candidate) {
			if p.Exists(path) {
				return path, true
			}
		}
	}
	return "", false
}

// expandPath substitutes `%VAR%` and `~`, then globs `*` segments newest-first.
func expandPath(p Probe, candidate string) []string {
	path := expandVars(p, candidate)
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		path = filepath.Join(home, path[2:])
	}
	if !strings.ContainsAny(path, "*?") {
		return []string{path}
	}
	matches, err := p.Glob(path)
	if err != nil || len(matches) == 0 {
		return nil
	}
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))
	return matches
}

// expandVars replaces every `%NAME%` with its value; an unknown name yields no
// candidates. Replacements are not rescanned, so a value containing `%` cannot
// restart the walk.
func expandVars(p Probe, path string) string {
	var out strings.Builder
	for {
		start := strings.IndexByte(path, '%')
		if start < 0 {
			out.WriteString(path)
			return out.String()
		}
		end := strings.IndexByte(path[start+1:], '%')
		if end < 0 {
			out.WriteString(path)
			return out.String()
		}
		value := p.Env(path[start+1 : start+1+end])
		if value == "" {
			return ""
		}
		out.WriteString(path[:start])
		out.WriteString(value)
		path = path[start+1+end+1:]
	}
}

// isBatchShim reports cmd/batch launchers, which Windows cannot start directly.
func isBatchShim(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".cmd", ".bat":
		return true
	}
	return false
}

// ValidatePath rejects empty and relative targets before any spawn happens.
func ValidatePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("no file to open")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("not an absolute path: %s", path)
	}
	return nil
}

// Open launches path in the editor, at line/column when the editor supports it.
// Callers fall back to the system default handler on ErrNotInstalled.
func Open(id, path string, line, column int) error {
	return OpenWith(DefaultProbe(), id, path, line, column)
}

// OpenWith is Open with a pinned probe. Tests pin the probe and stay on the
// resolution paths, so no editor is ever started by the test binary.
func OpenWith(p Probe, id, path string, line, column int) error {
	if err := ValidatePath(path); err != nil {
		return err
	}
	editor, ok := Find(id)
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownEditor, id)
	}
	exe, ok := Locate(p, editor)
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotInstalled, editor.ID)
	}
	return start(exe, Args(editor, path, line, column))
}
