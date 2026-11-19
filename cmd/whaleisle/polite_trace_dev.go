//go:build !release

package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/egoist/mygo"
)

type raiseTraceLog struct {
	sync.Mutex
	file *os.File
}

var raiseTraceFile raiseTraceLog

// startRaiseTrace writes the caller chain of every attempt to raise a window at
// the moment it happens, straight to a file and flushed: the run that needs it
// usually ends in a forced kill, and a buffered log dies with the process.
func startRaiseTrace() {
	path := os.Getenv("MYGO_TRACE_FILE")
	if path == "" {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		log.Println("polite trace:", err)
		return
	}
	raiseTraceFile.file = file
	mygo.RaiseTrace = func(call string, inactiveOnly bool, stack []byte) {
		raiseTraceFile.write("mygo."+call, fmt.Sprintf("inactiveOnly=%v", inactiveOnly), stack)
	}
	mygo.RaiseTraceNative = func(call string, stack []byte) {
		raiseTraceFile.write("native."+call, "", stack)
	}
}

func (f *raiseTraceLog) write(call, detail string, stack []byte) {
	f.Lock()
	defer f.Unlock()
	if f.file == nil {
		return
	}
	_, _ = fmt.Fprintf(f.file, "polite trace: %s %s\n%s\n", call, detail, stack)
	_ = f.file.Sync()
}
