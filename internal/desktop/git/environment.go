package git

import (
	"os"
	"strings"
)

// Match the original desktop Git child's authentication contract: use installed
// credential helpers, but report an unavailable login rather than opening an
// invisible terminal prompt or a separate AskPass window.
func gitChildEnv() []string {
	var env []string
	for _, value := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(value, "=", 2)[0])
		if strings.HasPrefix(key, "NPM_CONFIG_ELECTRON") {
			continue
		}
		switch key {
		case "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES",
			"GIT_CEILING_DIRECTORIES", "GIT_TERMINAL_PROMPT", "GIT_ASKPASS", "GCM_INTERACTIVE", "SSH_ASKPASS", "SSH_ASKPASS_REQUIRE", "LC_ALL":
			continue
		}
		env = append(env, value)
	}
	return append(env, "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=", "GCM_INTERACTIVE=never", "SSH_ASKPASS=", "SSH_ASKPASS_REQUIRE=never", "LC_ALL=C")
}
