//go:build !windows

package voicecapture

import "context"

func InputDevices() ([]Device, error)                                  { return nil, ErrUnsupported }
func StartPreview(ctx context.Context, device Device) (Preview, error) { return nil, ErrUnsupported }
func StartDevice(ctx context.Context, limits Limits, device Device) (Capture, error) {
	if device.ID != "" {
		return nil, ErrUnsupported
	}
	return Start(ctx, limits)
}
