package projectbridge

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceAuthoritySensitiveSubdirRefusalAndEnvClean(t *testing.T) {
	nodeBin, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node binary unavailable")
	}

	base := t.TempDir()
	homeDir := filepath.Join(base, "home")
	appDataDir := filepath.Join(base, "AppDataRoaming")
	localAppDataDir := filepath.Join(base, "AppDataLocal")
	dshHome := filepath.Join(appDataDir, "WhaleIsle", "dsh-home")
	xdgCfg := filepath.Join(base, "xdg-cfg")
	xdgData := filepath.Join(base, "xdg-data")

	mustMkdir := func(parts ...string) string {
		p := filepath.Join(parts...)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}

	mustMkdir(homeDir)
	mustMkdir(appDataDir)
	mustMkdir(localAppDataDir)
	mustMkdir(dshHome)
	mustMkdir(xdgCfg)
	mustMkdir(xdgData)

	sshSub := mustMkdir(homeDir, ".ssh", "keys")
	gnupgSub := mustMkdir(homeDir, ".gnupg", "private")
	awsSub := mustMkdir(homeDir, ".aws", "profiles")
	kubeSub := mustMkdir(homeDir, ".kube", "clusters")
	appSub := mustMkdir(appDataDir, "VendorApp")
	safeProject := mustMkdir(homeDir, "src", "project-alpha")

	script := `
const { workspaceRootRefusal, filterRegisteredWorkspaceRoots } = require('./source/main/workspace-authority.js');
const { gitChildEnv } = require('./source/main/git-exec.js');
const payload = JSON.parse(process.argv[1]);
const refusals = {};
for (const [k, p] of Object.entries(payload.checks)) {
  refusals[k] = workspaceRootRefusal(p);
}
const filtered = filterRegisteredWorkspaceRoots(Object.values(payload.checks));
const cleaned = gitChildEnv();
process.stdout.write(JSON.stringify({ refusals, filtered, cleaned }));
`
	argBytes, err := json.Marshal(map[string]any{
		"checks": map[string]string{
			"home":        homeDir,
			"sshSub":      sshSub,
			"gnupgSub":    gnupgSub,
			"awsSub":      awsSub,
			"kubeSub":     kubeSub,
			"appSub":      appSub,
			"dshHome":     dshHome,
			"safeProject": safeProject,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(nodeBin, "-e", script, string(argBytes))
	env := []string{}
	for _, entry := range os.Environ() {
		k, _, _ := strings.Cut(entry, "=")
		u := strings.ToUpper(k)
		switch {
		case u == "HOME", u == "USERPROFILE", u == "APPDATA", u == "LOCALAPPDATA",
			u == "XDG_CONFIG_HOME", u == "XDG_DATA_HOME", u == "DSH_HOME", u == "DSHD_HOME",
			u == "ELECTRON_RUN_AS_NODE", u == "GIT_DIR", u == "GIT_WORK_TREE", u == "GIT_INDEX_FILE",
			strings.HasPrefix(u, "MAGPIE_"):
			continue
		}
		env = append(env, entry)
	}
	env = append(env,
		"HOME="+homeDir,
		"USERPROFILE="+homeDir,
		"APPDATA="+appDataDir,
		"LOCALAPPDATA="+localAppDataDir,
		"XDG_CONFIG_HOME="+xdgCfg,
		"XDG_DATA_HOME="+xdgData,
		"DSH_HOME="+dshHome,
		"DSHD_HOME="+dshHome,
		"ELECTRON_RUN_AS_NODE=1",
		"MAGPIE_TOKEN=secret",
		"GIT_DIR=/bad/gitdir",
		"GIT_WORK_TREE=/bad/worktree",
		"GIT_INDEX_FILE=/bad/index",
	)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node failed: %v\n%s", err, out)
	}

	var res struct {
		Refusals map[string]string `json:"refusals"`
		Filtered []string          `json:"filtered"`
		Cleaned  map[string]string `json:"cleaned"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("unmarshal node output: %v\n%s", err, out)
	}

	for _, key := range []string{"home", "sshSub", "gnupgSub", "awsSub", "kubeSub", "appSub", "dshHome"} {
		if res.Refusals[key] == "" {
			t.Errorf("workspaceRootRefusal(%s) = empty, want refusal", key)
		}
	}
	if res.Refusals["safeProject"] != "" {
		t.Errorf("workspaceRootRefusal(safeProject) = %q, want empty", res.Refusals["safeProject"])
	}
	if len(res.Filtered) != 1 || !strings.EqualFold(filepath.Clean(res.Filtered[0]), filepath.Clean(safeProject)) {
		t.Errorf("filterRegisteredWorkspaceRoots = %v, want only [%q]", res.Filtered, safeProject)
	}
	for _, strippedKey := range []string{"ELECTRON_RUN_AS_NODE", "MAGPIE_TOKEN", "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE"} {
		if v, exists := res.Cleaned[strippedKey]; exists {
			t.Errorf("gitChildEnv leaked %s=%q", strippedKey, v)
		}
	}
}
