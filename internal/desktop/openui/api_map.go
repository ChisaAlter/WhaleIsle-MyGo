package openui

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// mapTile supplies the independently authored raster viewer. The opaque iframe
// receives a bitmap, never permission to make network requests or run workers.
func (a *ResourceAPI) mapTile(ctx context.Context, e Event) APIResponse {
	z, ez := strconv.Atoi(e.Query["z"])
	x, ex := strconv.Atoi(e.Query["x"])
	y, ey := strconv.Atoi(e.Query["y"])
	if ez != nil || ex != nil || ey != nil || z < 0 || z > 19 || x < 0 || y < 0 || x >= 1<<z || y >= 1<<z {
		return apiError(e, 400, "Invalid map tile coordinates")
	}
	service := "World_Street_Map"
	switch e.Query["style"] {
	case "", "light":
	case "dark":
		service = "Canvas/World_Dark_Gray_Base"
	default:
		return apiError(e, 400, "Unsupported map tile style")
	}
	url := fmt.Sprintf("https://server.arcgisonline.com/ArcGIS/rest/services/%s/MapServer/tile/%d/%d/%d", service, z, y, x)
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return apiError(e, 502, "Map tile request unavailable")
	}
	req.Header.Set("User-Agent", "WhaleIsle-MyGo OpenUI raster viewer")
	res, err := a.client.Do(req)
	if err != nil {
		return apiError(e, 502, "Map tile unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return apiError(e, 502, "Map tile unavailable")
	}
	limit := int64((MaxAPIResponseBytes - 4096) * 3 / 4)
	data, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil || int64(len(data)) > limit {
		return apiError(e, 502, "Map tile exceeds the resource envelope")
	}
	mime := http.DetectContentType(data)
	if mime != "image/png" && mime != "image/jpeg" {
		return apiError(e, 502, "Map tile is not a supported bitmap")
	}
	return apiResult(e, 200, map[string]any{
		"dataUrl": "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data),
		"z":       z, "x": x, "y": y, "style": e.Query["style"],
		"source":      url,
		"attribution": "Tiles © Esri — Source: Esri and its data contributors",
	})
}
