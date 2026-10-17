package openui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type APIResponse struct {
	InstanceID string          `json:"instanceId"`
	RequestID  string          `json:"requestId"`
	Status     int             `json:"status"`
	Body       json.RawMessage `json:"body"`
}

func ValidateAPIResponse(r APIResponse, instance string) error {
	raw, err := json.Marshal(r)
	if err != nil || len(raw) > MaxAPIResponseBytes || r.InstanceID != instance || r.RequestID == "" || r.Status < 200 || r.Status > 599 || !json.Valid(r.Body) {
		return errors.New("invalid OpenUI API response")
	}
	return nil
}

func validateAPIRequest(e Event) error {
	if e.RequestID == "" || e.Query == nil {
		return errors.New("invalid OpenUI API request")
	}
	key := ""
	switch e.Path {
	case "/api/image":
		key = "src"
	case "/api/wiki":
		key = "title"
	case "/api/image-search":
		key = "q"
	case "/api/street-route":
		key = "coords"
	case "/api/map-tile":
		for k := range e.Query {
			if k != "z" && k != "x" && k != "y" && k != "style" {
				return errors.New("unsupported OpenUI tile parameter")
			}
		}
		return nil
	default:
		return errors.New("unsupported OpenUI API path")
	}
	for k := range e.Query {
		if k != key {
			return errors.New("unsupported OpenUI API parameter")
		}
	}
	return nil
}

// ResourceAPI has its own credential-free client for fixed public adapters.
// The image operation uses a separate public-address-only transport; no caller
// can supply request headers, credentials, or a general response operation.
type ResourceAPI struct {
	client *http.Client
	caches *resourceCaches
}

func NewResourceAPI() *ResourceAPI {
	return &ResourceAPI{caches: &processResourceCaches, client: &http.Client{Transport: http.DefaultTransport.(*http.Transport).Clone(), CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func apiResult(e Event, status int, body any) APIResponse {
	raw, _ := json.Marshal(body)
	return APIResponse{InstanceID: e.InstanceID, RequestID: e.RequestID, Status: status, Body: raw}
}
func apiError(e Event, status int, message string) APIResponse {
	return apiResult(e, status, map[string]any{"error": map[string]string{"message": message}})
}
func (a *ResourceAPI) Handle(ctx context.Context, e Event) APIResponse {
	if err := validateAPIRequest(e); err != nil {
		return apiError(e, 400, err.Error())
	}
	var result APIResponse
	switch e.Path {
	case "/api/image":
		result = a.publicImage(ctx, e)
	case "/api/wiki":
		result = a.wiki(ctx, e)
	case "/api/image-search":
		result = a.images(ctx, e)
	case "/api/street-route":
		result = a.route(ctx, e)
	case "/api/map-tile":
		result = a.mapTile(ctx, e)
	}
	result = a.inlineResourceImages(ctx, e, result)
	if err := ValidateAPIResponse(result, e.InstanceID); err != nil {
		return apiError(e, 502, "Resource response exceeds the OpenUI bridge envelope")
	}
	return result
}
func (a *ResourceAPI) get(ctx context.Context, raw string, timeout time.Duration, out any, missingOK bool) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "open-intelligent-ui (https://github.com/thesysdev/open-intelligent-ui)")
	res, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if missingOK && res.StatusCode == 404 {
		return nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("resource HTTP %d", res.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(res.Body, MaxViewerBytes)).Decode(out)
}
func (a *ResourceAPI) wiki(ctx context.Context, e Event) APIResponse {
	title := strings.TrimSpace(e.Query["title"])
	if title == "" || len([]rune(title)) > 200 {
		return apiError(e, 400, "title is required")
	}
	cache := a.cacheFor("/api/wiki")
	if result, ok := cache.read(title, e); ok {
		return result
	}
	t := url.PathEscape(strings.ReplaceAll(title, " ", "_"))
	var summary struct {
		Thumbnail struct {
			Source string `json:"source"`
		} `json:"thumbnail"`
		Coordinates struct {
			Lat *float64 `json:"lat"`
			Lon *float64 `json:"lon"`
		} `json:"coordinates"`
	}
	var media struct {
		Items []struct {
			Type   string `json:"type"`
			Srcset []struct {
				Src string `json:"src"`
			} `json:"srcset"`
		} `json:"items"`
	}
	if a.get(ctx, "https://en.wikipedia.org/api/rest_v1/page/summary/"+t, 8*time.Second, &summary, true) != nil || a.get(ctx, "https://en.wikipedia.org/api/rest_v1/page/media-list/"+t, 8*time.Second, &media, true) != nil {
		return apiError(e, 502, "Wikipedia unavailable")
	}
	photos := []string{}
	if summary.Thumbnail.Source != "" {
		photos = append(photos, summary.Thumbnail.Source)
	}
	for _, item := range media.Items {
		if item.Type != "image" || len(item.Srcset) == 0 {
			continue
		}
		src := item.Srcset[0].Src
		if skipPhoto(src, false) {
			continue
		}
		if strings.HasPrefix(src, "//") {
			src = "https:" + src
		}
		dup := false
		for _, p := range photos {
			if photoKey(p) == photoKey(src) {
				dup = true
			}
		}
		if !dup {
			photos = append(photos, src)
		}
		if len(photos) >= 3 {
			break
		}
	}
	body := map[string]any{"photos": photos}
	if summary.Coordinates.Lat != nil {
		body["lat"] = *summary.Coordinates.Lat
	}
	if summary.Coordinates.Lon != nil {
		body["lng"] = *summary.Coordinates.Lon
	}
	return cache.retain(title, apiResult(e, 200, body))
}
func photoKey(s string) string {
	s = strings.SplitN(s, "?", 2)[0]
	s = s[strings.LastIndex(s, "/")+1:]
	if v, e := url.PathUnescape(s); e == nil {
		s = v
	}
	if i := strings.Index(s, "px-"); i > 0 {
		if _, e := strconv.Atoi(s[:i]); e == nil {
			s = s[i+3:]
		}
	}
	return s
}
func skipPhoto(s string, search bool) bool {
	s = strings.ToLower(s)
	terms := []string{".svg", "icon", "logo", "map", "flag", "seal"}
	if search {
		terms = append(terms, "coat_of_arms", "signature", "diagram")
	}
	for _, v := range terms {
		if strings.Contains(s, v) {
			return true
		}
	}
	return false
}

type resourcePage struct {
	Index     int    `json:"index"`
	Title     string `json:"title"`
	Thumbnail struct {
		Source string `json:"source"`
	} `json:"thumbnail"`
	ImageInfo []struct {
		ThumbURL string `json:"thumburl"`
		URL      string `json:"url"`
		MIME     string `json:"mime"`
	} `json:"imageinfo"`
}

func (a *ResourceAPI) query(ctx context.Context, host string, params url.Values) ([]resourcePage, error) {
	params.Set("action", "query")
	params.Set("format", "json")
	params.Set("formatversion", "2")
	var body struct {
		Error json.RawMessage `json:"error"`
		Query struct {
			Pages []resourcePage `json:"pages"`
		} `json:"query"`
	}
	if err := a.get(ctx, "https://"+host+"/w/api.php?"+params.Encode(), 8*time.Second, &body, false); err != nil {
		return nil, err
	}
	if len(body.Error) > 0 && string(body.Error) != "null" {
		return nil, errors.New("resource query failed")
	}
	sort.SliceStable(body.Query.Pages, func(i, j int) bool { return body.Query.Pages[i].Index < body.Query.Pages[j].Index })
	return body.Query.Pages, nil
}
func photoWords(s string) []string {
	all := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	out := []string{}
	for _, v := range all {
		if len([]rune(v)) > 3 {
			out = append(out, v)
		}
	}
	return out
}
func (a *ResourceAPI) images(ctx context.Context, e Event) APIResponse {
	q := strings.Join(strings.Fields(e.Query["q"]), " ")
	if len([]rune(q)) < 2 || len([]rune(q)) > 200 {
		return apiError(e, 400, "q must be 2 to 200 characters")
	}
	cache := a.cacheFor("/api/image-search")
	if result, ok := cache.read(q, e); ok {
		return result
	}
	images := []string{}
	add := func(u string) {
		if !strings.HasPrefix(u, "https://") || skipPhoto(u, true) {
			return
		}
		for _, v := range images {
			if v == u {
				return
			}
		}
		images = append(images, u)
	}
	pages, err := a.query(ctx, "en.wikipedia.org", url.Values{"generator": {"search"}, "gsrsearch": {q}, "gsrlimit": {"6"}, "prop": {"pageimages"}, "piprop": {"thumbnail"}, "pithumbsize": {"960"}})
	if err != nil {
		return apiError(e, 502, "Image search unavailable")
	}
	wanted := map[string]bool{}
	for _, v := range photoWords(q) {
		wanted[v] = true
	}
	for _, p := range pages {
		for _, w := range photoWords(p.Title) {
			if wanted[w] {
				add(p.Thumbnail.Source)
				break
			}
		}
	}
	terms := strings.Fields(q)
	tries := []string{q, strings.Join(terms[:min(3, len(terms))], " "), strings.Join(terms[:min(2, len(terms))], " ")}
	seen := map[string]bool{}
	for _, term := range tries {
		if len(images) >= 3 {
			break
		}
		if seen[term] {
			continue
		}
		seen[term] = true
		pages, err = a.query(ctx, "commons.wikimedia.org", url.Values{"generator": {"search"}, "gsrsearch": {term + " filetype:bitmap"}, "gsrnamespace": {"6"}, "gsrlimit": {"6"}, "prop": {"imageinfo"}, "iiprop": {"url|mime"}, "iiurlwidth": {"960"}})
		if err != nil {
			return apiError(e, 502, "Image search unavailable")
		}
		for _, p := range pages {
			if len(p.ImageInfo) > 0 {
				info := p.ImageInfo[0]
				if strings.HasPrefix(info.MIME, "image/") {
					u := info.ThumbURL
					if u == "" {
						u = info.URL
					}
					add(u)
				}
			}
		}
	}
	return cache.retain(q, apiResult(e, 200, map[string]any{"images": images[:min(4, len(images))]}))
}
func (a *ResourceAPI) route(ctx context.Context, e Event) APIResponse {
	pairs := strings.Split(e.Query["coords"], ";")
	if len(pairs) < 2 || len(pairs) > 25 {
		return apiError(e, 400, "coords must be 2–25 lng,lat pairs")
	}
	normalized := []string{}
	for _, pair := range pairs {
		v := strings.Split(pair, ",")
		if len(v) != 2 {
			return apiError(e, 400, "coords must be 2–25 lng,lat pairs")
		}
		nums := [2]float64{}
		for i, s := range v {
			n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) > []float64{180, 90}[i] {
				return apiError(e, 400, "coords must be 2–25 lng,lat pairs")
			}
			nums[i] = n
		}
		normalized = append(normalized, strconv.FormatFloat(nums[0], 'f', -1, 64)+","+strconv.FormatFloat(nums[1], 'f', -1, 64))
	}
	key := strings.Join(normalized, ";")
	cache := a.cacheFor("/api/street-route")
	if result, ok := cache.read(key, e); ok {
		return result
	}
	var body struct {
		Code   string `json:"code"`
		Routes []struct {
			Geometry struct {
				Coordinates [][2]float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"routes"`
	}
	if a.get(ctx, "https://router.project-osrm.org/route/v1/driving/"+key+"?overview=full&geometries=geojson&steps=false", 10*time.Second, &body, false) != nil || body.Code != "Ok" || len(body.Routes) == 0 || len(body.Routes[0].Geometry.Coordinates) < 2 {
		return apiError(e, 502, "Street routing unavailable")
	}
	coords := body.Routes[0].Geometry.Coordinates
	for i := range coords {
		coords[i][0], coords[i][1] = coords[i][1], coords[i][0]
	}
	return cache.retain(key, apiResult(e, 200, map[string]any{"points": coords}))
}
