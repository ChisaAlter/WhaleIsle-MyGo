package update

import (
	"fmt"
	"strings"
)

// NativeReleasePrompt summarizes the already fetched, prepared Versions catalog.
// It performs no network, download or installation operation.
func NativeReleasePrompt(current string, release NativeRelease, rows []NativeRelease, chinese bool) string {
	text := func(zh, en string) string {
		if chinese {
			return zh
		}
		return en
	}
	lines := []string{fmt.Sprintf(text("当前版本：%s\n可用版本：%s", "Current version: %s\nAvailable version: %s"), current, release.Version)}
	methods := []string{}
	seen := map[string]bool{}
	for _, row := range rows {
		if row.Tag != release.Tag || row.Platform != release.Platform || row.Arch != release.Arch || row.UnavailableReason != "" {
			continue
		}
		kind := text("全量", "Full")
		size := row.Size
		if row.IsDelta() {
			if row.Delta == nil {
				continue
			}
			kind = fmt.Sprintf(text("增量（基于 %s）", "Delta (from %s)"), row.Delta.BaseVersion)
			size = row.Delta.DownloadSize
		}
		variant := text("仅桌面端", "desktop only")
		if row.RuntimeIncluded {
			variant = text("包含运行时", "includes runtime")
		}
		method := kind + " · " + variant
		if size > 0 {
			method += fmt.Sprintf(" · %.1f MiB", float64(size)/(1<<20))
		}
		if !seen[method] {
			seen[method] = true
			methods = append(methods, method)
		}
	}
	if len(methods) == 0 {
		methods = append(methods, text("请在版本页检查可用安装包", "Review available packages on the Versions page"))
	}
	lines = append(lines, text("可用下载方式：\n", "Available downloads:\n")+strings.Join(methods, "\n"))
	notes := strings.TrimSpace(release.Notes)
	if notes == "" {
		notes = text("此发布未提供说明。", "No release notes were provided.")
	}
	if chars := []rune(notes); len(chars) > 3000 {
		notes = string(chars[:3000]) + text("\n…完整说明见版本页。", "\n…Full notes are on the Versions page.")
	}
	lines = append(lines, text("发布说明：\n", "Release notes:\n")+notes)
	lines = append(lines, text("继续将打开版本页，可选择全量或可用的增量包；现在不会下载或安装。下载校验完成后，点击安装按钮才会确认安装。", "Continue to Versions to choose a full or available delta package. Nothing is downloaded or installed now. After verification, the install button confirms installation."))
	return strings.Join(lines, "\n\n")
}
