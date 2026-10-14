package harness

import "whaleisle/internal/desktop/projectbridge"

// NewProjectBridge uses the same resolved Node executable as the Host launcher.
func NewProjectBridge(nodeBin, home, workspace string, openPath func(string) error) *projectbridge.Bridge {
	node, err := resolveNode(nodeBin)
	if err != nil {
		node = ""
	}
	return projectbridge.New(projectbridge.Options{NodeBin: node, Home: home, Workspace: workspace, OpenPath: openPath})
}
