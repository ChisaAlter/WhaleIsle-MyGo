//go:build !windows

package components

import "fmt"

func bridgeAlertProcessIdentity(int, string) (string, error) {
	return "", fmt.Errorf("鲸桥组件提醒需要 Windows")
}
