package harness

import (
	"regexp"
	"strconv"
	"strings"
)

// PluginProgress is parsed harness plugin / overlay load progress for the boot UI.
type PluginProgress struct {
	// Ready is the number of completed steps (overlays mounted or runtime entries activated).
	Ready int
	// Total is the step count when known; zero means indeterminate.
	Total int
	// Phase is a short machine tag: "overlays", "runtime".
	Phase string
	// Detail names the current item when known.
	Detail string
}

// pluginOverlayLine matches the supervisor log after desktop Prepare.
var pluginOverlayLine = regexp.MustCompile(`(\d+) desktop --patch overlay`)

// pluginLoadLine matches an optional progress hint emitted on harness stderr
// (used by tests and compatible with future structured loader logs).
var pluginLoadLine = regexp.MustCompile(`(?i)dsh plugin-load:\s*(\d+)\s*/\s*(\d+)(?:\s+\(([^)]+)\))?`)

// ApplyPluginLogLine updates prog from one harness stdout/stderr line.
// It returns true when prog changed.
func ApplyPluginLogLine(prog *PluginProgress, line string) bool {
	if prog == nil {
		return false
	}
	before := *prog
	if m := pluginOverlayLine.FindStringSubmatch(line); len(m) >= 2 {
		n, _ := strconv.Atoi(m[1])
		if n > 0 {
			prog.Phase = "overlays"
			prog.Total = n
			prog.Ready = n
			prog.Detail = ""
		}
	}
	if m := pluginLoadLine.FindStringSubmatch(strings.TrimSpace(line)); len(m) >= 3 && m[1] != "" {
		ready, _ := strconv.Atoi(m[1])
		total, _ := strconv.Atoi(m[2])
		prog.Phase = "runtime"
		prog.Ready = ready
		prog.Total = total
		if len(m) >= 4 {
			prog.Detail = strings.TrimSpace(m[3])
		}
	}
	return *prog != before
}
