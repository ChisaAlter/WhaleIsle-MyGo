package petgrowth

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed scan.cjs
var scanScript string

// Scan reads only the authorized app DSH home's sessions directory. The caller
// supplies existing native config, never URL/page parameters. Node executes our
// embedded scanner, not a package from disk. No IPC/Host token is passed.
func Scan(ctx context.Context, node, home string) (map[string]int64, error) {
	if node == "" || home == "" || !filepath.IsAbs(home) {
		return nil, errors.New("pet scanner runtime unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "--max-old-space-size=512", "--input-type=commonjs", "-", filepath.Join(home, "sessions"))
	hideScanner(cmd)
	cmd.Stdin = strings.NewReader(scanScript)
	// Avoid NODE_OPTIONS/NODE_PATH loaders and unrelated credentials inherited
	// by the shell. The fixed builtin-only script needs just platform temp/root.
	cmd.Env = []string{}
	for _, key := range []string{"SystemRoot", "WINDIR", "TEMP", "TMP", "TMPDIR", "LANG"} {
		if value, ok := os.LookupEnv(key); ok {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	var out cappedOutput
	cmd.Stdout = &out
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Run(); err != nil {
		return nil, errors.New("pet usage scan unavailable or incomplete")
	}
	var samples map[string]int64
	if err := json.Unmarshal(out.Bytes(), &samples); err != nil {
		return nil, errors.New("invalid pet usage scan result")
	}
	if _, err := ValidateSamples(samples); err != nil {
		return nil, err
	}
	return samples, nil
}

type cappedOutput struct{ bytes.Buffer }

func (b *cappedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 32*1024*1024 {
		return 0, errors.New("pet usage scan result limit")
	}
	return b.Buffer.Write(p)
}
