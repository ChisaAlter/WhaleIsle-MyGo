// Package mathrender renders TeX offline with the bundled MathJax engine and
// the product's existing Node runtime. It never invokes a user's TeX install.
package mathrender

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"whaleisle/internal/paths"
)

//go:embed renderer.cjs
var renderer string

const MaxFormulaBytes = 32 * 1024
const maxOutputBytes = 4*1024*1024 + 64*1024

type Formula struct {
	TeX     string `json:"tex"`
	Display bool   `json:"display"`
}

type Result struct {
	SVG      string  `json:"svg"`
	WidthEm  float64 `json:"widthEm"`
	HeightEm float64 `json:"heightEm"`
	DepthEm  float64 `json:"depthEm"`
	Error    string  `json:"error,omitempty"`
}

// BundledNode deliberately does not fall back to PATH or an unrequested install.
func BundledNode() (string, error) {
	name := "node"
	if runtime.GOOS == "windows" {
		name = "node.exe"
	}
	var candidates []string
	if selected := paths.SelectedHostRuntime(); selected != "" {
		candidates = append(candidates, filepath.Join(selected, "node", name))
	}
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(base, "runtime", "node", name), filepath.Join(base, "node", name))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate, nil
		}
	}
	return "", errors.New("offline math renderer requires the product's bundled Node runtime")
}

// Render is a blocking worker operation; UI callers must invoke it off-thread.
// Each conversion has a fresh MathJax document and a bounded child lifetime.
func Render(ctx context.Context, node string, formula Formula) (Result, error) {
	if len(formula.TeX) > MaxFormulaBytes {
		return Result{}, errors.New("formula exceeds 32 KiB")
	}
	if node == "" || !filepath.IsAbs(node) {
		return Result{}, errors.New("an absolute bundled Node path is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	request, err := json.Marshal(formula)
	if err != nil {
		return Result{}, err
	}
	cmd := exec.CommandContext(ctx, node, "--max-old-space-size=256", "--input-type=commonjs", "-")
	hideWindow(cmd)
	// No NODE_OPTIONS/NODE_PATH, account tokens, shell startup or user scripts.
	cmd.Env = []string{}
	for _, key := range []string{"SystemRoot", "WINDIR", "TEMP", "TMP", "TMPDIR", "LANG"} {
		if value, ok := os.LookupEnv(key); ok {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	// Passing the embedded code on stdin avoids Windows' command-line limit.
	// TeX is JSON data and is never interpolated as executable JavaScript.
	cmd.Stdin = io.MultiReader(strings.NewReader(renderer), strings.NewReader("\nprocess.stdout.write(JSON.stringify(globalThis.whaleisleRenderMath("), bytes.NewReader(request), strings.NewReader(")));\n"))
	output := &limitedBuffer{remaining: maxOutputBytes}
	stderr := &limitedBuffer{remaining: 16 * 1024}
	cmd.Stdout, cmd.Stderr = output, stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		return Result{}, fmt.Errorf("offline math renderer failed: %w", err)
	}
	var result Result
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		return Result{}, fmt.Errorf("invalid math renderer response: %w", err)
	}
	if result.Error != "" {
		return Result{}, errors.New(result.Error)
	}
	if result.SVG == "" || !finitePositive(result.WidthEm) || !finitePositive(result.HeightEm) || math.IsNaN(result.DepthEm) || math.IsInf(result.DepthEm, 0) || result.DepthEm < 0 || result.DepthEm > result.HeightEm {
		return Result{}, errors.New("invalid equation geometry")
	}
	return result, nil
}

func finitePositive(n float64) bool { return n > 0 && !math.IsNaN(n) && !math.IsInf(n, 0) }

type limitedBuffer struct {
	bytes.Buffer
	remaining int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.remaining {
		return 0, errors.New("math renderer output limit exceeded")
	}
	b.remaining -= len(p)
	return b.Buffer.Write(p)
}
