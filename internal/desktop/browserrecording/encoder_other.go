//go:build !windows || (!amd64 && !arm64)

package browserrecording

import (
	"context"
	"errors"
)

func newEncoder(context.Context) (encoder, error) {
	return nil, errors.New("网页视频录制需要 Windows WebView2 的 WebM MediaRecorder")
}
