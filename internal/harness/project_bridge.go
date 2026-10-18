package harness

import (
	"context"
	"whaleisle/internal/desktop/projectbridge"
)

// NewProjectBridge uses the same resolved Node executable as the Host launcher.
func NewProjectBridge(nodeBin, home, workspace string, openPath func(string) error, selectDirectory ...func(context.Context, string) (string, error)) *projectbridge.Bridge {
	node, err := resolveNode(nodeBin)
	if err != nil {
		node = ""
	}
	var selectPath func(context.Context, string) (string, error)
	if len(selectDirectory) > 0 {
		selectPath = selectDirectory[0]
	}
	return projectbridge.New(projectbridge.Options{SelectDirectory: selectPath, NodeBin: node, Home: home, Workspace: workspace, OpenPath: openPath})
}
