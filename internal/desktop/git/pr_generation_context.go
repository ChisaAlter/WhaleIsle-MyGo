package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Bound stdout while still draining the child, so large branch histories/diffs
// cannot grow UI memory. No generation command is streamed as progress output.
type generationCapture struct {
	data      []byte
	limit     int
	truncated bool
}

func (w *generationCapture) Write(p []byte) (int, error) {
	n := len(p)
	left := w.limit - len(w.data)
	if n > left {
		w.truncated = true
		p = p[:left]
	}
	w.data = append(w.data, p...)
	return n, nil
}
func generationGitBytes(ctx context.Context, root string, max int, args ...string) ([]byte, bool, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	for _, entry := range os.Environ() {
		key := strings.SplitN(entry, "=", 2)[0]
		if key == "GIT_DIR" || key == "GIT_WORK_TREE" || key == "GIT_COMMON_DIR" || key == "GIT_INDEX_FILE" || key == "GIT_OPTIONAL_LOCKS" {
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env, "GIT_OPTIONAL_LOCKS=0")
	out := &generationCapture{limit: max}
	stderr := &generationCapture{limit: 2000}
	cmd.Stdout = out
	cmd.Stderr = stderr
	if e := cmd.Run(); e != nil {
		return nil, false, fmt.Errorf("cannot read PR generation context: %w", e)
	}
	return out.data, out.truncated, nil
}
func generationGitText(ctx context.Context, root string, max int, args ...string) (string, error) {
	b, truncated, e := generationGitBytes(ctx, root, max*4, args...)
	if e != nil {
		return "", e
	}
	s := strings.TrimSpace(string(b))
	if truncated {
		s += "\n\n[truncated]"
	}
	return generationLimit(s, max), nil
}

var prTemplateFiles = []string{".github/pull_request_template.md", ".github/PULL_REQUEST_TEMPLATE.md", "pull_request_template.md", "PULL_REQUEST_TEMPLATE.md", "docs/pull_request_template.md", "docs/PULL_REQUEST_TEMPLATE.md"}
var prTemplateDirs = []string{".github/PULL_REQUEST_TEMPLATE", "PULL_REQUEST_TEMPLATE", "docs/PULL_REQUEST_TEMPLATE"}

func readPRGenerationTemplate(ctx context.Context, root, tree string) string {
	args := []string{"ls-tree", "-r", "-z", "--full-tree", tree, "--"}
	args = append(args, prTemplateFiles...)
	args = append(args, prTemplateDirs...)
	data, truncated, e := generationGitBytes(ctx, root, 100000, args...)
	if e != nil || truncated {
		return ""
	}
	entries := map[string]string{}
	for _, record := range strings.Split(string(data), "\x00") {
		parts := strings.SplitN(record, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		meta := strings.Fields(parts[0])
		if len(meta) != 3 || meta[1] != "blob" || (meta[0] != "100644" && meta[0] != "100755") {
			continue
		}
		entries[parts[1]] = meta[2]
	}
	read := func(oid string) string {
		data, truncated, e := generationGitBytes(ctx, root, 8000, "cat-file", "blob", oid)
		if e != nil {
			return ""
		}
		s := strings.TrimSpace(string(data))
		if truncated {
			s += "\n\n[truncated]"
		}
		return s
	}
	for _, path := range prTemplateFiles {
		if oid := entries[path]; oid != "" {
			if s := read(oid); s != "" {
				return s
			}
		}
	}
	for _, dir := range prTemplateDirs {
		var ids []string
		for path, oid := range entries {
			if strings.HasPrefix(path, dir+"/") && !strings.Contains(strings.TrimPrefix(path, dir+"/"), "/") && strings.HasSuffix(strings.ToLower(path), ".md") {
				ids = append(ids, oid)
			}
		}
		if len(ids) > 1 {
			return ""
		}
		if len(ids) == 1 {
			if s := read(ids[0]); s != "" {
				return s
			}
		}
	}
	return ""
}
