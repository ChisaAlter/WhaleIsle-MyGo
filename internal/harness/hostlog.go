package harness

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"time"
)

// hostLogKeep is how many host-*.log files the logs directory retains.
const hostLogKeep = 10

// hostLogCapBytes bounds one Host log file; past it a marker is written once
// and the rest of the process's output stays only in the in-memory ring.
const hostLogCapBytes = 20 << 20

// hostLogQueue bounds pending lines between s.log and the sink's writer.
// A full queue counts a drop instead of blocking the supervisor on disk.
const hostLogQueue = 1024

// hostLogFlushEvery bounds how long a written line can sit in the buffer.
const hostLogFlushEvery = 250 * time.Millisecond

// hostLogSink persists one Host attempt's output to
// <dir>/host-<yyyyMMdd-HHmmss>-<pid>.log. Every line the in-memory ring
// accepts is offered to a bounded channel; a single goroutine owns the file
// and the buffer so disk latency never reaches the supervisor.
type hostLogSink struct {
	path    string
	queue   chan string
	done    chan struct{}
	dropped atomic.Int64
}

// newHostLogSink opens the log file for pid under dir and starts draining.
// A nil sink is returned when the directory or file cannot be opened: log
// persistence must never fail a Host start.
func newHostLogSink(dir string, pid int) *hostLogSink {
	return openHostLogSink(dir, pid, hostLogCapBytes)
}

func openHostLogSink(dir string, pid int, capBytes int) *hostLogSink {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil
	}
	name := fmt.Sprintf("host-%s-%d.log", time.Now().Format("20060102-150405"), pid)
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return nil
	}
	// Prune after create so the new file counts toward the kept set.
	pruneHostLogs(dir)
	k := &hostLogSink{
		path:  filepath.Join(dir, name),
		queue: make(chan string, hostLogQueue),
		done:  make(chan struct{}),
	}
	go k.drain(f, int64(capBytes))
	return k
}

// pruneHostLogs deletes host-*.log files beyond the newest hostLogKeep.
func pruneHostLogs(dir string) {
	matches, err := filepath.Glob(filepath.Join(dir, "host-*.log"))
	if err != nil || len(matches) <= hostLogKeep {
		return
	}
	sort.Strings(matches) // timestamped names sort oldest first
	for _, stale := range matches[:len(matches)-hostLogKeep] {
		_ = os.Remove(stale)
	}
}

// add offers one line; when the queue is full the line is dropped and the
// drop is counted so the drain can leave a marker. Callers must hand the
// sink over (nil their reference) before close, which is how the supervisor
// keeps add free of any cross-check with the queue's close.
func (k *hostLogSink) add(line string) {
	select {
	case k.queue <- line:
	default:
		k.dropped.Add(1)
	}
}

// close shuts the queue, drains what remains, flushes and closes the file.
func (k *hostLogSink) close() {
	close(k.queue)
	<-k.done
}

// drain is the sink's only file owner.
func (k *hostLogSink) drain(f *os.File, capBytes int64) {
	defer close(k.done)
	defer f.Close()
	w := bufio.NewWriterSize(f, 64*1024)
	flush := time.NewTicker(hostLogFlushEvery)
	defer flush.Stop()
	var written int64
	truncated := false
	write := func(line string) {
		if truncated {
			return
		}
		need := int64(len(line)) + 1
		if written+need > capBytes {
			truncated = true
			// The marker itself may pass the cap by a few bytes; what the cap
			// bounds is the unbounded output that follows.
			w.WriteString("log truncated at 20 MiB")
			w.WriteByte('\n')
			return
		}
		w.WriteString(line)
		w.WriteByte('\n')
		written += need
	}
	reportDrops := func() {
		if n := k.dropped.Swap(0); n > 0 {
			write(fmt.Sprintf("%d lines dropped", n))
		}
	}
	for {
		select {
		case line, ok := <-k.queue:
			if !ok {
				reportDrops()
				w.Flush()
				return
			}
			write(line)
			reportDrops()
		case <-flush.C:
			w.Flush()
		}
	}
}
