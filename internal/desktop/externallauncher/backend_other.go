//go:build !windows

package externallauncher

import "errors"

func Native() *Controller {
	return New(Dependencies{Discover: Discover, Maintenance: func() (func(), error) { return nil, errors.New("external native launcher requires Windows") }})
}
func Discover() (Installation, error) {
	return Installation{}, errors.New("external native launcher requires Windows")
}
