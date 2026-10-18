package htmldocument

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	_ "golang.org/x/image/webp"
)

const (
	MaxAssetBytes = 4 << 20
	MaxGraphBytes = 16 << 20 // source document and unique referenced resources
	MaxAssets     = 64
	MaxReferences = 512
	MaxCSSDepth   = 4
)

// ResourceReport contains no file paths, data, credentials or capability URLs.
// It is frozen with the document so unsupported dependencies remain visible.
type ResourceReport struct {
	Files, Bytes, Skipped int
	Warnings              []string
}

func (r ResourceReport) Summary() string {
	s := fmt.Sprintf("已内联 %d 个静态资源（%d KiB）", r.Files, (r.Bytes+1023)/1024)
	if r.Skipped > 0 {
		s += fmt.Sprintf("；跳过 %d 项：%s", r.Skipped, strings.Join(r.Warnings, "、"))
	}
	return s
}

type resourceFile struct {
	data []byte
	mime string
}
type resourceGraph struct {
	root          *os.Root
	canonicalRoot string
	valid         func() bool
	remoteRead    func(path string, limit int) ([]byte, error)
	files         map[string]resourceFile
	active        map[string]bool
	references    int
	report        ResourceReport
}

func newResourceGraph(root *os.Root, sourceBytes int, valid func() bool) (*resourceGraph, error) {
	canonical, err := filepath.EvalSymlinks(root.Name())
	if err != nil {
		return nil, err
	}
	return &resourceGraph{root: root, canonicalRoot: canonical, valid: valid, files: map[string]resourceFile{}, active: map[string]bool{}, report: ResourceReport{Bytes: sourceBytes}}, nil
}
func (g *resourceGraph) warn(reason string) {
	g.report.Skipped++
	for _, old := range g.report.Warnings {
		if old == reason {
			return
		}
	}
	if len(g.report.Warnings) < 6 {
		g.report.Warnings = append(g.report.Warnings, reason)
	}
}

// relativeAsset interprets URL paths once, relative to the declaring file. It
// rejects site-root URLs as the original token-prefixed preview also does.
func relativeAsset(base, reference string) (string, error) {
	ref, err := relativeAssetPath(reference)
	if err != nil {
		return "", err
	}
	rel := filepath.Clean(filepath.Join(filepath.Dir(base), filepath.FromSlash(ref)))
	if outsideAssetRoot(rel) || gitPath(rel) {
		return "", errors.New("资源越出授权工作区或位于 .git")
	}
	return rel, nil
}

func relativeAssetPath(reference string) (string, error) {
	if len(reference) > 8192 || strings.ContainsAny(reference, "\\\x00\r\n") {
		return "", errors.New("不支持的资源路径")
	}
	u, err := url.Parse(strings.TrimSpace(reference))
	if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil || u.Opaque != "" || u.Path == "" || strings.HasPrefix(u.Path, "/") || strings.ContainsAny(u.Path, "\\\x00\r\n:") {
		return "", errors.New("外部或绝对资源路径")
	}
	for _, r := range u.Path {
		if r < 32 || r == 127 {
			return "", errors.New("不支持的资源路径")
		}
	}
	return u.Path, nil
}
func outsideAssetRoot(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel)
}
func gitPath(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if strings.EqualFold(part, ".git") {
			return true
		}
	}
	return false
}

func (g *resourceGraph) read(base, reference string, cssOnly bool) (resourceFile, string, string, error) {
	kind := "asset"
	if cssOnly {
		kind = "css"
	}
	return g.readKind(base, reference, kind)
}

func (g *resourceGraph) readKind(base, reference, kind string) (resourceFile, string, string, error) {
	if !g.valid() {
		return resourceFile{}, "", "", errors.New("预览所属会话已改变")
	}
	g.references++
	if g.references > MaxReferences {
		return resourceFile{}, "", "", errors.New("资源引用数量超过限制")
	}
	if g.remoteRead != nil {
		return g.readRemote(base, reference, kind)
	}
	rel, err := relativeAsset(base, reference)
	if err != nil {
		return resourceFile{}, "", "", err
	}
	ext := strings.ToLower(filepath.Ext(rel))
	if !resourceKindAllows(kind, ext) {
		return resourceFile{}, "", "", errors.New("不支持的资源类型")
	}
	info, err := g.root.Stat(rel)
	if err != nil || !info.Mode().IsRegular() {
		return resourceFile{}, "", "", errors.New("资源缺失或不可读取")
	}
	// os.Root enforces the actual opened file boundary; canonical identity also
	// de-duplicates aliases and rejects aliases into .git before reading bytes.
	canonical, err := filepath.EvalSymlinks(filepath.Join(g.root.Name(), rel))
	if err != nil {
		return resourceFile{}, "", "", errors.New("资源路径无法解析")
	}
	canonicalRel, err := filepath.Rel(g.canonicalRoot, canonical)
	if err != nil || outsideAssetRoot(canonicalRel) || gitPath(canonicalRel) {
		return resourceFile{}, "", "", errors.New("资源越出授权工作区或位于 .git")
	}
	if cached, ok := g.files[canonical]; ok {
		if !resourceKindMIME(kind, cached.mime) {
			return resourceFile{}, "", "", errors.New("资源类型别名不匹配")
		}
		return cached, rel, canonical, nil
	}
	if len(g.files) >= MaxAssets {
		return resourceFile{}, "", "", errors.New("资源文件数量超过限制")
	}
	if info.Size() < 1 || info.Size() > MaxAssetBytes || info.Size() > int64(MaxGraphBytes-g.report.Bytes) {
		return resourceFile{}, "", "", errors.New("资源大小超过限制")
	}
	f, err := g.root.Open(rel)
	if err != nil {
		return resourceFile{}, "", "", errors.New("资源无法在工作区内打开")
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return resourceFile{}, "", "", errors.New("资源读取前已改变")
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxAssetBytes+1))
	if err != nil {
		return resourceFile{}, "", "", errors.New("资源读取失败")
	}
	after, err := f.Stat()
	if err != nil || !after.ModTime().Equal(info.ModTime()) || after.Size() != info.Size() || int64(len(data)) != info.Size() || len(data) > MaxAssetBytes || len(data) > MaxGraphBytes-g.report.Bytes {
		return resourceFile{}, "", "", errors.New("资源读取时改变或超出大小限制")
	}
	file, err := g.rememberResource(canonical, ext, data)
	return file, rel, canonical, err
}

func (g *resourceGraph) rememberResource(key, ext string, data []byte) (resourceFile, error) {
	mime, err := resourceMIME(ext, data)
	if err != nil {
		return resourceFile{}, err
	}
	if ext == ".svg" && (bytes.Contains(bytes.ToLower(data), []byte("<style")) || bytes.Contains(bytes.ToLower(data), []byte("<image"))) {
		g.warn("SVG 内部嵌套依赖不递归展开")
	}
	if !g.valid() {
		return resourceFile{}, errors.New("预览所属会话已改变")
	}
	file := resourceFile{data: data, mime: mime}
	g.files[key] = file
	g.report.Files++
	g.report.Bytes += len(data)
	return file, nil
}
func (g *resourceGraph) stylesheet(base, reference string, depth int) (string, error) {
	if depth > MaxCSSDepth {
		return "", errors.New("CSS 依赖层级超过限制")
	}
	file, rel, key, err := g.read(base, reference, true)
	if err != nil {
		return "", err
	}
	if g.active[key] {
		return "", errors.New("CSS 循环引用")
	}
	g.active[key] = true
	defer delete(g.active, key)
	return inlineCSS(string(file.data), rel, depth, g.stylesheet, g.asset, g.warn)
}
func (g *resourceGraph) asset(base, reference string) (string, error) {
	return g.binary(base, reference, false)
}
func (g *resourceGraph) image(base, reference string) (string, error) {
	return g.binary(base, reference, true)
}
func (g *resourceGraph) binary(base, reference string, imageOnly bool) (string, error) {
	reference = strings.TrimSpace(reference)
	if strings.HasPrefix(reference, "#") {
		return reference, nil
	}
	if strings.HasPrefix(strings.ToLower(reference), "data:") {
		// Already-embedded bytes remain bounded by the HTML/CSS source budget.
		mime := strings.ToLower(strings.SplitN(strings.SplitN(reference, ";", 2)[0], ",", 2)[0])
		if strings.HasPrefix(mime, "data:image/") || (!imageOnly && strings.HasPrefix(mime, "data:font/")) {
			return reference, nil
		}
		return "", errors.New("不支持的嵌入资源类型")
	}
	file, _, _, err := g.read(base, reference, false)
	if err != nil {
		return "", err
	}
	if imageOnly && !strings.HasPrefix(file.mime, "image/") {
		return "", errors.New("图片类型不匹配")
	}
	return "data:" + file.mime + ";base64," + base64.StdEncoding.EncodeToString(file.data), nil
}
func binaryExtension(ext string) bool {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".woff", ".woff2", ".ttf", ".otf":
		return true
	}
	return false
}
func resourceMIME(ext string, data []byte) (string, error) {
	bad := errors.New("资源内容与允许类型不符")
	if ext == ".css" || ext == ".js" || ext == ".mjs" {
		if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
			return "", bad
		}
		if ext != ".css" {
			return "text/javascript", nil
		}
		return "text/css", nil
	}
	if ext == ".svg" {
		d := xml.NewDecoder(bytes.NewReader(data))
		roots := 0
		depth := 0
		tokens := 0
		for {
			tok, err := d.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", bad
			}
			tokens++
			if tokens > 100000 {
				return "", bad
			}
			switch v := tok.(type) {
			case xml.Directive:
				return "", bad
			case xml.StartElement:
				if depth == 0 {
					roots++
					if roots != 1 || v.Name.Local != "svg" || (v.Name.Space != "" && v.Name.Space != "http://www.w3.org/2000/svg") {
						return "", bad
					}
				}
				depth++
				if depth > 256 {
					return "", bad
				}
			case xml.EndElement:
				depth--
			}
		}
		if roots != 1 || depth != 0 {
			return "", bad
		}
		return "image/svg+xml", nil
	}
	if len(data) < 4 {
		return "", bad
	}
	sig := string(data[:4])
	switch ext {
	case ".woff":
		if sig == "wOFF" && len(data) >= 44 {
			return "font/woff", nil
		}
		return "", bad
	case ".woff2":
		if sig == "wOF2" && len(data) >= 48 {
			return "font/woff2", nil
		}
		return "", bad
	case ".ttf":
		if (sig == "\x00\x01\x00\x00" || sig == "true") && len(data) >= 12 {
			return "font/ttf", nil
		}
		return "", bad
	case ".otf":
		if sig == "OTTO" && len(data) >= 12 {
			return "font/otf", nil
		}
		return "", bad
	}
	want := map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif", ".webp": "image/webp"}[ext]
	if want == "" || http.DetectContentType(data) != want {
		return "", bad
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 20000 || config.Height > 20000 || int64(config.Width)*int64(config.Height) > 40_000_000 {
		return "", errors.New("图片尺寸无效或超过限制")
	}
	return want, nil
}
