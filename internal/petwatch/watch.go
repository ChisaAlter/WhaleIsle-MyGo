// Package petwatch reads only structural events from the current owned Host's
// authorized data home. It does not connect to Host, run models or execute logs.
package petwatch

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"whaleisle/internal/petgrowth"
)

//go:embed watch.cjs
var script string

type Event struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Text      string `json:"text"`
	SessionID string `json:"sessionID,omitempty"`
	MediaPath string `json:"mediaPath,omitempty"`
	Pinned    bool   `json:"pinned,omitempty"`
}
type Snapshot struct {
	State  string  `json:"state"`
	Events []Event `json:"events"`
}

// Run blocks until cancellation or scanner exit. The owner serializes runs for
// one home, and gates every callback on the originating Host generation. Ack
// only receipts the renderer has visibly shown and dismissed/finished showing.
func Run(ctx context.Context, node, home, stickerRoot string, acknowledgements <-chan string, report func(Snapshot)) error {
	if node == "" || !filepath.IsAbs(home) {
		return errors.New("pet event source unavailable")
	}
	cmd := exec.CommandContext(ctx, node, "--max-old-space-size=256", "-e", script, "--", "--pet-event-worker", home, stickerRoot, petgrowth.MagpieUsageFile())
	hideScanner(cmd)
	cmd.Env = []string{}
	for _, key := range []string{"SystemRoot", "WINDIR", "TEMP", "TMP", "TMPDIR", "LANG"} {
		if value, ok := os.LookupEnv(key); ok {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	cmd.WaitDelay = 2 * time.Second
	output, err := cmd.StdoutPipe()
	if err != nil {
		return errors.New("pet event reader unavailable")
	}
	input, err := cmd.StdinPipe()
	if err != nil {
		return errors.New("pet event reader unavailable")
	}
	cmd.Stderr = io.Discard
	if err = cmd.Start(); err != nil {
		return errors.New("pet event runtime unavailable")
	}
	done := make(chan struct{})
	go func() {
		defer input.Close()
		encoder := json.NewEncoder(input)
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case id, ok := <-acknowledgements:
				if !ok {
					return
				}
				if len(id) == 32 {
					if encoder.Encode(map[string]string{"ack": id}) != nil {
						return
					}
				}
			}
		}
	}()
	scan := bufio.NewScanner(output)
	scan.Buffer(make([]byte, 65536), 2*1024*1024)
	for scan.Scan() {
		if ctx.Err() != nil {
			break
		}
		var snap Snapshot
		if json.Unmarshal(scan.Bytes(), &snap) != nil || len(snap.Events) > 256 {
			continue
		}
		valid := true
		for _, e := range snap.Events {
			if len(e.ID) != 32 || len(e.Text) > 4096 || len(e.SessionID) > 2048 {
				valid = false
				break
			}
		}
		if valid && report != nil {
			report(snap)
		}
	}
	close(done)
	if scan.Err() != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	err = cmd.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil || scan.Err() != nil {
		return errors.New("pet event reader stopped")
	}
	return nil
}
