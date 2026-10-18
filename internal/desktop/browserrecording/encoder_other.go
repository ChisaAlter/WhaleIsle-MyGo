//go:build (!windows && !darwin && !linux) || (!amd64 && !arm64)

package browserrecording

import (
	"context"
	"errors"
)

func newEncoder(context.Context) (encoder, error) {
	return nil, errors.New("当前平台不支持网页 WebM 录制")
}
