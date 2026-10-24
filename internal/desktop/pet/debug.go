package pet

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"
)

// petDebug appends one diagnostic line to the file named by WHALEISLE_PET_DEBUG.
// It is inert unless the variable names a path, so shipped builds pay one
// getenv per lifecycle step and no I/O.
func petDebug(format string, args ...any) {
	path := os.Getenv("WHALEISLE_PET_DEBUG")
	if path == "" {
		return
	}
	line := time.Now().UTC().Format("2006-01-02T15:04:05.000Z") + "  " + fmt.Sprintf(format, args...) + "\r\n"
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line)
}

// petDebugStack records who closed the companion.
func petDebugStack(format string, args ...any) {
	petDebug(format+"\n%s", append(args, debug.Stack())...)
}
