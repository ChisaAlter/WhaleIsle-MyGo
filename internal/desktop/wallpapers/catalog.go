// Package wallpapers implements the original e9 wallpaper catalog contracts.
package wallpapers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Source struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}
type Item struct {
	ID        string `json:"id"`
	SourceID  string `json:"sourceId"`
	Title     string `json:"title"`
	Copyright string `json:"copyright,omitempty"`
	ThumbURL  string `json:"thumbUrl"`
	ImageURL  string `json:"imageUrl"`
}
type Query struct {
	Source     Source
	Search     string
	Categories string
	Year       int
	Page       int
}
type Result struct {
	Items    []Item
	NextPage int
	Warning  string
}

func Sources() []Source {
	return []Source{{ID: "bing", Kind: "bing", Name: "必应"}, {ID: "wallhaven", Kind: "wallhaven", Name: "Wallhaven"}}
}

// Retain the original catalog's HTTPS/public-host, redirect, byte and timeout boundaries.
func blocked(host string) bool {
	host = strings.ToLower(strings.Trim(host, "[]"))
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true
		}
		if v := ip.To4(); v != nil {
			return v[0] == 0 || (v[0] == 100 && v[1] >= 64 && v[1] <= 127) || (v[0] == 198 && (v[1] == 18 || v[1] == 19))
		}
	}
	return false
}
func URL(raw string) (*url.URL, error) {
	u, e := url.Parse(strings.TrimSpace(raw))
	if e != nil || u.Scheme != "https" || u.User != nil || blocked(u.Hostname()) {
		return nil, fmt.Errorf("壁纸地址必须是公开 HTTPS 地址")
	}
	return u, nil
}
func allowed(ctx context.Context, u *url.URL) error {
	if _, err := URL(u.String()); err != nil {
		return err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, u.Hostname())
	if err != nil {
		return err
	}
	for _, ip := range ips {
		if blocked(ip.IP.String()) {
			return fmt.Errorf("壁纸地址解析到不允许的内网地址")
		}
	}
	return nil
}
func Fetch(ctx context.Context, raw string, image bool) ([]byte, error) {
	u, err := URL(raw)
	if err != nil {
		return nil, err
	}
	if err = allowed(ctx, u); err != nil {
		return nil, err
	}
	limit := int64(4000000)
	timeout := 30 * time.Second
	if image {
		limit = 12 * 1024 * 1024
		timeout = 20 * time.Second
	}
	client := &http.Client{Timeout: timeout, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 4 {
			return fmt.Errorf("壁纸重定向过多")
		}
		return allowed(req.Context(), req.URL)
	}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Deepseek-Harness-Desktop")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("壁纸请求失败 (%d)", resp.StatusCode)
	}
	if image {
		mime := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
		switch mime {
		case "image/png", "image/jpeg", "image/jpg", "image/webp", "image/gif":
		default:
			return nil, fmt.Errorf("壁纸不是可用的图片")
		}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err == nil && int64(len(data)) > limit {
		err = fmt.Errorf("壁纸文件过大")
	}
	return data, err
}
func resolve(base, relative string) string {
	b, e := url.Parse(base)
	if e != nil {
		return ""
	}
	u, e := url.Parse(relative)
	if e != nil || relative == "" {
		return ""
	}
	v := b.ResolveReference(u).String()
	if _, e = URL(v); e != nil {
		return ""
	}
	return v
}
func List(ctx context.Context, q Query) Result {
	page := max(1, q.Page)
	urls := []string{}
	archive := false
	switch q.Source.Kind {
	case "bing":
		if q.Year > 0 {
			archive = true
			urls = []string{fmt.Sprintf("https://bing.npanuhin.me/CN-zh.%d.json", q.Year)}
		} else {
			urls = []string{"https://cn.bing.com/HPImageArchive.aspx?format=js&idx=0&n=8&mkt=zh-CN", "https://cn.bing.com/HPImageArchive.aspx?format=js&idx=8&n=8&mkt=zh-CN"}
		}
	case "wallhaven":
		categories := q.Categories
		if categories != "010" && categories != "001" {
			categories = "100"
		}
		v := url.Values{"purity": {"100"}, "categories": {categories}, "sorting": {"toplist"}, "atleast": {"1920x1080"}, "page": {strconv.Itoa(page)}}
		if q.Search != "" {
			v.Set("q", q.Search)
		}
		urls = []string{"https://wallhaven.cc/api/v1/search?" + v.Encode()}
	case "catalog":
		urls = []string{q.Source.URL}
	default:
		return Result{Warning: "请选择图库来源"}
	}
	result := Result{}
	seen := map[string]bool{}
	for _, u := range urls {
		data, err := Fetch(ctx, u, false)
		if err != nil {
			result.Warning += err.Error() + "；"
			continue
		}
		var rows []map[string]json.RawMessage
		var body struct {
			Images []map[string]json.RawMessage `json:"images"`
			Items  []map[string]json.RawMessage `json:"items"`
			Data   []map[string]json.RawMessage `json:"data"`
			Meta   struct {
				Current int `json:"current_page"`
				Last    int `json:"last_page"`
			} `json:"meta"`
		}
		kind := q.Source.Kind
		if archive {
			err = json.Unmarshal(data, &rows)
		} else {
			err = json.Unmarshal(data, &body)
			if body.Images != nil {
				rows = body.Images
				kind = "bing"
			} else if body.Items != nil {
				rows = body.Items
				kind = "catalog"
			} else if body.Data != nil && kind == "wallhaven" {
				rows = body.Data
				if body.Meta.Current < body.Meta.Last {
					result.NextPage = body.Meta.Current + 1
				}
			} else if err == nil {
				err = fmt.Errorf("不是壁纸目录")
			}
		}
		if err != nil {
			result.Warning += err.Error() + "；"
			continue
		}
		for _, r := range rows[:min(500, len(rows))] {
			s := func(key string) string { var v string; _ = json.Unmarshal(r[key], &v); return v }
			item := Item{SourceID: q.Source.ID, Title: s("title"), Copyright: s("copyright")}
			switch {
			case archive:
				item.ID = "bing-" + s("date")
				item.ImageURL = resolve(u, s("bing_url"))
				if item.ImageURL == "" {
					item.ImageURL = resolve(u, s("url"))
				}
				item.ThumbURL = item.ImageURL
			case kind == "bing":
				if string(r["wp"]) == "false" {
					continue
				}
				base := s("urlbase")
				if base != "" {
					item.ImageURL = resolve(u, base+"_1920x1080.jpg")
					item.ThumbURL = resolve(u, base+"_400x240.jpg")
				} else {
					item.ImageURL = resolve(u, s("url"))
					item.ThumbURL = item.ImageURL
				}
				item.ID = s("hsh")
				if item.ID == "" {
					item.ID = item.ImageURL
				}
				if item.Title == "" {
					item.Title = item.Copyright
				}
			case kind == "wallhaven":
				id := s("id")
				item.ID = "wallhaven-" + id
				item.Title = id
				item.ImageURL = resolve(u, s("path"))
				var thumbs map[string]string
				_ = json.Unmarshal(r["thumbs"], &thumbs)
				item.ThumbURL = resolve(u, thumbs["large"])
				if item.ThumbURL == "" {
					item.ThumbURL = resolve(u, thumbs["small"])
				}
			default:
				item.ID = s("id")
				item.ImageURL = resolve(u, s("imageUrl"))
				item.ThumbURL = resolve(u, s("thumbUrl"))
			}
			if item.ID == "" || item.ImageURL == "" || item.ThumbURL == "" || seen[item.ID] {
				continue
			}
			seen[item.ID] = true
			if q.Source.Kind != "wallhaven" && q.Search != "" && !strings.Contains(strings.ToLower(item.Title+" "+item.Copyright+" "+item.ID), strings.ToLower(q.Search)) {
				continue
			}
			result.Items = append(result.Items, item)
		}
	}
	return result
}
