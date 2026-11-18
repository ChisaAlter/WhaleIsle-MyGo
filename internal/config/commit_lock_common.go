package config

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"os"
	"time"
)

const (
	commitLockRetryBudget = 160 * time.Millisecond
	commitLockMaxDelay    = 30 * time.Millisecond
)

type commitLockMetadata struct {
	PID       int   `json:"pid"`
	CreatedAt int64 `json:"createdAt"`
}

func acquireWithRetry(tryLock func() (func(), error)) (func(), error) {
	deadline := time.Now().Add(commitLockRetryBudget)
	backoff := 3 * time.Millisecond
	for {
		release, err := tryLock()
		if err == nil {
			return release, nil
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, err
		}
		wait := jitteredDuration(backoff)
		if wait > remaining {
			wait = remaining
		}
		time.Sleep(wait)
		if backoff < commitLockMaxDelay {
			backoff *= 2
			if backoff > commitLockMaxDelay {
				backoff = commitLockMaxDelay
			}
		}
	}
}

func writeCommitLockMetadata(file *os.File) {
	payload, err := json.Marshal(commitLockMetadata{
		PID:       os.Getpid(),
		CreatedAt: time.Now().UnixMilli(),
	})
	if err != nil {
		return
	}
	payload = append(payload, '\n')
	if _, err := file.Seek(0, 0); err != nil {
		return
	}
	_ = file.Truncate(0)
	_, _ = file.Write(payload)
}

func jitteredDuration(base time.Duration) time.Duration {
	if base <= 0 {
		return time.Millisecond
	}
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return base
	}
	fraction := float64(binary.LittleEndian.Uint16(b[:])) / 65535.0
	return base + time.Duration(float64(base)*0.5*fraction)
}
