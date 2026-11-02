package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"whaleisle/internal/paths"
	"whaleisle/internal/product"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo"
)

// Crash report files: the complete diagnostic of one fatal shell failure,
// written before the recovery surface so that surface can name the file.
// Port of reference/electron/src/main/crash-report.js (crash-report.js:13-23
// for the contract, :106-127 for the writes, :119-135 for retention).
const (
	// CrashReportsRetained is how many reports survive, newest by file name.
	CrashReportsRetained = 10
	// CrashRendererConsoleMaxBytes bounds the captured console tail.
	CrashRendererConsoleMaxBytes = 64 * 1024
	// CrashErrorSectionMaxChars bounds the rendered error section.
	CrashErrorSectionMaxChars = 256 * 1024
)

// Crash report sources. host is the local Harness service, main is this
// process's own startup, web-boot is a hosted web document that never came up.
// MyGo has no renderer process and no renderer console bridge, so renderer is
// accepted only for compatibility: nothing in this shell produces it.
var crashReportSources = map[string]bool{
	"host":     true,
	"web-boot": true,
	"renderer": true,
	"main":     true,
}

// crashReportName is the native CRASH_REPORT_NAME regex
// (reference/electron/src/main/crash-report.js:19).
var crashReportName = regexp.MustCompile(`^crash-\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}-\d{3}Z-(?:host|web-boot|renderer|main)\.log$`)

// CrashReportFileName is the native crashReportFileName
// (reference/electron/src/main/crash-report.js:81-83): an ISO instant with
// ":" and "." replaced by "-", then the source. Unknown sources fall back to
// main so the file always matches the retention regex.
func CrashReportFileName(t time.Time, source string) string {
	if !crashReportSources[source] {
		source = "main"
	}
	utc := t.UTC()
	return fmt.Sprintf("crash-%s-%03dZ-%s.log", utc.Format("2006-01-02T15-04-05"), utc.Nanosecond()/int(time.Millisecond), source)
}

// CrashReportInput is one fatal failure, as the native renderCrashReport takes it.
type CrashReportInput struct {
	Time   time.Time
	Source string
	// Phase stays "error" for a fatal failure, as desktopErrorState reports.
	Phase string
	// Error is the rendered failure; it is bounded by CrashErrorSectionMaxChars.
	Error string
	// HostDiagnostic is the Host process's own account of the failure, when the
	// failure came from the local Harness service.
	HostDiagnostic string
	// RendererConsole holds error-level console lines, oldest first. MyGo has no
	// renderer console bridge, so it is normally empty.
	RendererConsole []string
	// AppName/Version identify the shell; empty values fall back to this build.
	AppName string
	AppVer  string
	Locale  string
}

// RenderCrashReport renders one report, mirroring the native sections: header,
// bounded error, optional host diagnostic, renderer console tail.
func RenderCrashReport(in CrashReportInput) string {
	name := in.AppName
	if name == "" {
		name = product.Name
	}
	version := in.AppVer
	if version == "" {
		version = buildVersion
	}
	phase := in.Phase
	if phase == "" {
		phase = "error"
	}
	locale := in.Locale
	if locale == "" {
		locale = "zh-CN"
	}
	header := []string{
		"time: " + in.Time.UTC().Format(time.RFC3339Nano),
		"source: " + in.Source,
		"phase: " + phase,
		"app: " + name + " " + version,
		"platform: " + runtime.GOOS + " " + runtime.GOARCH,
		// MyGo is a Go shell, not Electron: keep the native keys so the report
		// reads the same, and state honestly what runs here.
		"electron: none (MyGo native shell)",
		"node: none",
		"go: " + runtime.Version(),
		"locale: " + locale,
		fmt.Sprintf("shell pid: %d", os.Getpid()),
	}
	console := "(no error-level renderer console output was captured)"
	if len(in.RendererConsole) > 0 {
		console = strings.Join(in.RendererConsole, "\n")
	}
	sections := []string{
		strings.Join(header, "\n"),
		"",
		"--- error ---",
		boundedCrashError(in.Error),
		"",
	}
	if in.HostDiagnostic != "" {
		sections = append(sections, "--- host diagnostic (as reported by the Host process) ---", in.HostDiagnostic, "")
	}
	sections = append(sections,
		"--- renderer console (error level, oldest first) ---",
		console,
		"",
	)
	return strings.Join(sections, "\n")
}

// boundedCrashError is the native boundedErrorSection
// (reference/electron/src/main/crash-report.js:85-91).
func boundedCrashError(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "(no error details were captured)"
	}
	if len(text) <= CrashErrorSectionMaxChars {
		return text
	}
	return text[:CrashErrorSectionMaxChars] +
		fmt.Sprintf("\n(error section cut at %d characters)", CrashErrorSectionMaxChars)
}

// crashConsoleTail is the native RendererConsoleTail
// (reference/electron/src/main/crash-report.js:29-73): the newest lines whose
// bytes fit, and a single oversized line keeps only its UTF-8-safe suffix.
type crashConsoleTail struct {
	maxBytes int
	lines    []string
	bytes    int
}

func newCrashConsoleTail(maxBytes int) *crashConsoleTail {
	if maxBytes <= 0 {
		maxBytes = CrashRendererConsoleMaxBytes
	}
	return &crashConsoleTail{maxBytes: maxBytes}
}

func (t *crashConsoleTail) Push(line string) {
	if len(line) > t.maxBytes {
		start := len(line) - t.maxBytes
		for start < len(line) && !utf8.RuneStart(line[start]) {
			start++
		}
		line = line[start:]
		t.lines, t.bytes = nil, 0
	}
	if line == "" {
		return
	}
	t.lines = append(t.lines, line)
	t.bytes += len(line)
	for len(t.lines) > 1 && t.bytes > t.maxBytes {
		t.bytes -= len(t.lines[0])
		t.lines = t.lines[1:]
	}
}

func (t *crashConsoleTail) Snapshot() []string {
	return append([]string(nil), t.lines...)
}

// WriteCrashReport writes the report and then prunes old ones. Every failure is
// logged and reported as "no file": a diagnostic aid must never mask the fatal
// error the caller is about to surface.
func WriteCrashReport(dir string, in CrashReportInput) string {
	if in.Time.IsZero() {
		in.Time = time.Now()
	}
	file := filepath.Join(dir, CrashReportFileName(in.Time, in.Source))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Println("crash report directory:", dir, err)
		return ""
	}
	// "wx", as the native writeFile uses: one failure, one file.
	handle, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		log.Println("crash report could not be written:", file, err)
		return ""
	}
	if _, err := handle.WriteString(RenderCrashReport(in)); err != nil {
		_ = handle.Close()
		log.Println("crash report could not be written:", file, err)
		return ""
	}
	if err := handle.Close(); err != nil {
		log.Println("crash report could not be written:", file, err)
		return ""
	}
	PruneCrashReports(dir, CrashReportsRetained)
	return file
}

// PruneCrashReports deletes the oldest reports beyond retained, judged by file
// name order as the native pruneCrashReports does
// (reference/electron/src/main/crash-report.js:119-135). Anything else in the
// directory is left alone.
func PruneCrashReports(dir string, retained int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println("crash report directory could not be listed:", dir, err)
		}
		return
	}
	var reports []string
	for _, entry := range entries {
		if !entry.IsDir() && crashReportName.MatchString(entry.Name()) {
			reports = append(reports, entry.Name())
		}
	}
	sort.Strings(reports)
	excess := len(reports) - max(0, retained)
	for i := 0; i < excess; i++ {
		path := filepath.Join(dir, reports[i])
		if err := os.Remove(path); err != nil {
			log.Println("stale crash report could not be removed:", path, err)
		}
	}
}

// CrashLogDir is where the shell keeps reports: <userData>/logs/crash, the
// native crashLogDir (reference/electron/src/main/paths.js: app logs + crash).
func CrashLogDir() string { return filepath.Join(paths.UserDataDir(), "logs", "crash") }

// recordShellCrash writes one report for a fatal shell failure and returns its
// path for the error surface to name, or "" when nothing could be written.
// Callers treat that as "no file": the failure they are about to show stands.
func recordShellCrash(source, message, hostDiagnostic string) string {
	return WriteCrashReport(CrashLogDir(), CrashReportInput{
		Time:           time.Now(),
		Source:         source,
		Phase:          "error",
		Error:          message,
		HostDiagnostic: hostDiagnostic,
		Locale:         i18n.Locale(),
	})
}

// recordWebBootFailure keeps a diagnostic when a hosted web document never came
// up. MyGo has no boot web process: its boot page is native ui.Content, so this
// is the nearest real web-boot surface (the panel/browse WebView2 host).
func (s *mainState) recordWebBootFailure(message string) {
	if message == "" {
		return
	}
	if path := recordShellCrash("web-boot", message, ""); path != "" {
		s.mu.Lock()
		s.crashReportPath = path
		s.mu.Unlock()
	}
}

// publishCrashReport names a written report where the shell shows failures: the
// launcher's start error and the exported boot log. It never invents an error
// line when there is no failure surface to attach to.
func (s *mainState) publishCrashReport(win *mygo.Window, path string) {
	if path == "" {
		return
	}
	line := fmt.Sprintf(i18n.T(i18n.KeyCrashReportWritten), path)
	s.mu.Lock()
	s.crashReportPath = path
	if s.launcherStartErr != "" {
		s.launcherStartErr += "\n" + line
	}
	s.bootLogs = append(s.bootLogs, line)
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}
