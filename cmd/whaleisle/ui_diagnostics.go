package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"whaleisle/internal/store/session"
)

// ui_diagnostics.go holds the opt-in diagnostics used to hunt UI-thread stalls.
// Every hook is inert unless its environment variable is set, so normal runs pay
// one getenv each and nothing else.
//
//	WHALEISLE_LOG_FILE         redirect the process log (std log package) to a file
//	WHALEISLE_CPU_PROFILE      write a pprof CPU profile to a file
//	   _DELAY seconds before it starts (default 0), _SECONDS how long (default 120)
//	WHALEISLE_UI_SAMPLE        sample goroutine 1's stack into a file
//	   _SAMPLE_MS sampling interval in milliseconds (default 250)
var (
	uiDiagFrames      atomic.Int64
	uiDiagLastFrameNs atomic.Int64
	uiDiagSlowFrames  atomic.Int64
)

func uiDiagSeconds(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

// startUIDiagnostics installs the hooks named by the environment. It is called
// once from main before the application loop starts.
func startUIDiagnostics() {
	if path := strings.TrimSpace(os.Getenv("WHALEISLE_LOG_FILE")); path != "" {
		if f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600); err == nil {
			log.SetOutput(f)
			log.SetFlags(log.Ltime | log.Lmicroseconds)
		}
	}
	if path := strings.TrimSpace(os.Getenv("WHALEISLE_CPU_PROFILE")); path != "" {
		delay := uiDiagSeconds("WHALEISLE_CPU_PROFILE_DELAY", 0)
		seconds := uiDiagSeconds("WHALEISLE_CPU_PROFILE_SECONDS", 120)
		go func() {
			if delay > 0 {
				time.Sleep(time.Duration(delay) * time.Second)
			}
			f, err := os.Create(path)
			if err != nil {
				log.Printf("ui diagnostics: cpu profile: %v", err)
				return
			}
			if err := pprof.StartCPUProfile(f); err != nil {
				log.Printf("ui diagnostics: cpu profile: %v", err)
				_ = f.Close()
				return
			}
			time.Sleep(time.Duration(seconds) * time.Second)
			pprof.StopCPUProfile()
			_ = f.Close()
			log.Printf("ui diagnostics: cpu profile written to %s after %ds", path, seconds)
		}()
	}
	if path := strings.TrimSpace(os.Getenv("WHALEISLE_ALLOC_PROFILE")); path != "" {
		delay := uiDiagSeconds("WHALEISLE_ALLOC_PROFILE_DELAY", 60)
		go func() {
			time.Sleep(time.Duration(delay) * time.Second)
			f, err := os.Create(path)
			if err != nil {
				log.Printf("ui diagnostics: alloc profile: %v", err)
				return
			}
			_ = pprof.Lookup("allocs").WriteTo(f, 0)
			_ = f.Close()
			log.Printf("ui diagnostics: alloc profile written to %s", path)
		}()
	}
	if path := strings.TrimSpace(os.Getenv("WHALEISLE_UI_SAMPLE")); path != "" {
		interval := time.Duration(uiDiagSeconds("WHALEISLE_UI_SAMPLE_MS", 250)) * time.Millisecond
		if interval < 20*time.Millisecond {
			interval = 20 * time.Millisecond
		}
		go func() {
			f, err := os.Create(path)
			if err != nil {
				log.Printf("ui diagnostics: ui sample: %v", err)
				return
			}
			defer f.Close()
			w := bufio.NewWriterSize(f, 1<<20)
			defer w.Flush()
			start := time.Now()
			buf := make([]byte, 4<<20)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for range ticker.C {
				n := runtime.Stack(buf, true)
				frames := uiDiagFrames.Load()
				last := uiDiagLastFrameNs.Load()
				builds, hits := uiDiagSnapshotCounters()
				fmt.Fprintf(w, "=== t=%dms wall=%s frames=%d slowFrames=%d lastViewUs=%d snapBuilds=%d snapHits=%d\n%s\n",
					time.Since(start).Milliseconds(), time.Now().Format("15:04:05.000"), frames, uiDiagSlowFrames.Load(), last/1000, builds, hits,
					uiDiagGoroutineOne(string(buf[:n])))
				if trace := session.ViewTraceReport(); trace != "" {
					fmt.Fprintf(w, "### viewTrace\n%s\n", trace)
				}
				_ = w.Flush()
			}
		}()
	}
}

// recordViewFrame records one mainView build for the sampler. It is deferred
// first in mainView so it covers the whole build, overlay included.
func recordViewFrame(start time.Time) {
	uiDiagFrames.Add(1)
	took := time.Since(start)
	uiDiagLastFrameNs.Store(int64(took))
	if took > 16*time.Millisecond {
		uiDiagSlowFrames.Add(1)
	}
}

// uiDiagSnapshotCounters reports the session store's projected-versus-cached
// snapshots, so a frame that rebuilds every time is visible in the samples.
func uiDiagSnapshotCounters() (uint64, uint64) {
	state.mu.Lock()
	s := state.sessions
	state.mu.Unlock()
	if s == nil {
		return 0, 0
	}
	return s.SnapshotStats()
}

// uiDiagGoroutineOne returns the first 18 lines of goroutine 1's stack, which is
// the window thread: whatever it is inside when a ping waits is the answer.
func uiDiagGoroutineOne(all string) string {
	start := -1
	if strings.HasPrefix(all, "goroutine 1 ") {
		start = 0
	} else if i := strings.Index(all, "\ngoroutine 1 "); i >= 0 {
		start = i + 1
	}
	if start < 0 {
		return "(goroutine 1 not found)"
	}
	block := all[start:]
	if end := strings.Index(block, "\n\n"); end >= 0 {
		block = block[:end]
	}
	lines := strings.Split(block, "\n")
	if len(lines) > 18 {
		lines = lines[:18]
	}
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}
	return strings.Join(lines, "\n")
}
