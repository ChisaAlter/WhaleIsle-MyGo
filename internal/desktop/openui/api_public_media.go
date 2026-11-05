package openui

import (
	"context"
	"encoding/base64"
	"net/http"
)

// publicMedia shares the existing public-address-only resource transport and
// 16 MiB budget. It cannot carry Host credentials, request headers or redirects
// into a private address. Only browser-decodable audio/video bodies cross back.
func (a *ResourceAPI) publicMedia(ctx context.Context, e Event) APIResponse {
	resource, failure := readPublicResource(ctx, e)
	if failure != nil {
		return *failure
	}
	return publicMediaResponse(e, resource)
}

func publicMediaResponse(e Event, resource publicResource) APIResponse {
	if len(resource.data) > 16<<20 {
		return apiError(e, 413, "Media exceeds the 16 MiB byte budget")
	}
	mime := http.DetectContentType(resource.data)
	switch mime {
	case "application/ogg", "audio/mpeg", "audio/ogg", "audio/wave", "audio/wav", "audio/x-wav", "audio/aiff", "audio/midi", "video/mp4", "video/webm", "video/ogg", "video/quicktime":
	default:
		return apiError(e, 415, "Unsupported public audio/video format")
	}
	return apiResult(e, 200, map[string]any{"dataUrl": "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(resource.data), "source": resource.source})
}
