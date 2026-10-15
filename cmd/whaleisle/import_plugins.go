package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/importer"
	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/harness"
	"whaleisle/internal/harness/installctrl"
)

func (s *mainState) importPluginInstaller(win *mygo.Window, dest, node string) importer.PluginInstaller {
	return func(ctx context.Context, row importer.CatalogItem, spec string) importer.PluginInstallResult {
		s.mu.Lock()
		allowed := s.launcherImportBusy && !s.launcherStartBusy && !s.quitting &&
			(s.sup == nil || s.sup.Status().State == harness.Stopped)
		if allowed {
			s.launcherImport = "正在重新安装插件 " + row.ID + "；来源 " + spec
		}
		s.mu.Unlock()
		if !allowed {
			return importer.PluginInstallResult{Error: "导入状态或运行时状态已改变，插件未安装。"}
		}
		if win != nil { win.Update(func() {}) }
		return installImportPlugin(ctx, spec, func(allow []string) installctrl.Result {
			return desktopmarket.InstallSpecContext(ctx, spec, allow, desktopmarket.InstallOptions{NodeBin: node, DshHome: dest, ExpectedPackageName: row.ID})
		}, func(builds []string) bool {
			if ctx.Err() != nil || win == nil { return false }
			answer, err := mygo.Dialog.Message(mygo.MessageOptions{
				Parent: win, Type: mygo.MessageQuestion, Title: "批准插件构建脚本",
				Message: fmt.Sprintf("插件：%s\n来源：%s\n目标：%s\n\n以下包要求在本机运行构建脚本，可能访问本机文件和网络：\n%s\n\n仅批准本次列出的脚本并重试该插件安装？", row.ID, spec, dest, strings.Join(builds, "\n")),
				Buttons: []string{"跳过此插件", "批准并继续"}, DefaultButton: 0, CancelButton: 0,
			})
			return err == nil && answer.Button == 1 && ctx.Err() == nil
		})
	}
}

// installImportPlugin never widens an approval or retries an unchanged rejected
// build set. CLI execution is injected so fixtures cannot start a Host or install.
func installImportPlugin(ctx context.Context, spec string, run func([]string) installctrl.Result, approve func([]string) bool) importer.PluginInstallResult {
	var approved []string
	var logs []string
	seenBuilds := map[string]bool{}
	for {
		if err := ctx.Err(); err != nil {
			return importer.PluginInstallResult{Error: err.Error(), Log: strings.Join(logs, "\n")}
		}
		result := run(append([]string(nil), approved...))
		if result.Log != "" { logs = append(logs, result.Log) }
		out := importer.PluginInstallResult{OK: result.OK && !result.NeedsAllowBuilds, Error: result.Error, Log: strings.Join(logs, "\n")}
		if !result.NeedsAllowBuilds { return out }
		if err := ctx.Err(); err != nil {
			out.Error = err.Error()
			return out
		}
		builds := installctrl.NormalizeAllowBuilds(result.AllowBuilds)
		if len(builds) == 0 {
			out.Error = "安装器未返回有效的构建脚本清单；未批准。"
			return out
		}
		slices.Sort(builds)
		key := strings.Join(builds, "\x00")
		if seenBuilds[key] {
			out.Error = "已批准的构建脚本仍被拒绝；请检查安装日志后重试。"
			return out
		}
		if approve == nil || !approve(append([]string(nil), builds...)) {
			out.Error = "未批准构建脚本，跳过此插件。"
			return out
		}
		seenBuilds[key] = true
		approved = builds
	}
}
