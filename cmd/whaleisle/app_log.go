package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// appLogFile holds the log's file open for the life of the process; without a
// reference the finalizer would close it under the writers.
var appLogFile *os.File

// appLogPath is where a launch writes its process log.
func appLogPath(userData string) string {
	return filepath.Join(userData, "logs", "app-"+time.Now().Format("2006-01-02")+".log")
}

// startAppLog sends the process log to the profile's logs folder, the same
// directory 设置 → 关于 → 打开日志目录 opens. A GUI launch has no console, so
// without this the copy a failure sentence points at does not exist.
func startAppLog(userData string) error {
	if strings.TrimSpace(os.Getenv("WHALEISLE_LOG_FILE")) != "" {
		return nil
	}
	path := appLogPath(userData)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	appLogFile = f
	log.SetOutput(io.MultiWriter(os.Stderr, f))
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("app log: %s", path)
	return nil
}
