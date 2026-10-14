//go:build !windows

package components

import "fmt"

func openBridgeProcess(int, string) (bridgeProcess, error) {
	return nil, fmt.Errorf("鲸桥组件进程管理仅支持 Windows")
}
