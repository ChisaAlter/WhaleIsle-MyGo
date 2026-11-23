package assistant

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
)

// decodeBotAvatarImage is the validation plus decode the media worker runs.
func decodeBotAvatarImage(v pluginrpc.DshbotAvatarValues) (*ui.Bitmap, error) {
	data, err := botAvatarBytes(v)
	if err != nil {
		return nil, err
	}
	return ui.DecodeBitmap(data)
}
