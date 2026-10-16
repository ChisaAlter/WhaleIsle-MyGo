//go:build !windows

package appuninstall

func DetectCurrent() (Plan, error) { return Plan{}, ErrUnsupported }
func Start(Plan) error             { return ErrUnsupported }
