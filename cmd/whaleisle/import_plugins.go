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
	"whaleisle/internal/ui/i18n"
)

func (s *mainState) importPluginInstaller(win *mygo.Window, dest, node string) importer.PluginInstaller {
	return func(ctx context.Context, row importer.CatalogItem, spec string) importer.PluginInstallResult {
		s.mu.Lock()
		allowed := s.launcherImportBusy && !s.launcherStartBusy && !s.quitting &&
			(s.sup == nil || s.sup.Status().State == harness.Stopped)
		if allowed {
			s.launcherImport = fmt.Sprintf(i18n.T(i18n.KeyCmdImportPluginProgress), row.ID, spec)
		}
		s.mu.Unlock()
		if !allowed {
			return importer.PluginInstallResult{Error: i18n.T(i18n.KeyCmdImportPluginStateChanged)}
		}
		if win != nil {
			win.Update(func() {})
		}
		return installImportPlugin(ctx, spec, func(allow []string) installctrl.Result {
			return desktopmarket.InstallSpecContext(ctx, spec, allow, desktopmarket.InstallOptions{NodeBin: node, DshHome: dest, ExpectedPackageName: row.ID})
		}, func(builds []string) bool {
			if ctx.Err() != nil || win == nil {
				return false
			}
			answer, err := mygo.Dialog.Message(mygo.MessageOptions{
				Parent: win, Type: mygo.MessageQuestion, Title: i18n.T(i18n.KeyCmdImportBuildTitle),
				Message: fmt.Sprintf(i18n.T(i18n.KeyCmdImportBuildMessage), row.ID, spec, dest, strings.Join(builds, "\n")),
				Buttons: []string{i18n.T(i18n.KeyCmdImportBuildSkip), i18n.T(i18n.KeyCmdImportBuildApprove)}, DefaultButton: 0, CancelButton: 0,
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
		if result.Log != "" {
			logs = append(logs, result.Log)
		}
		out := importer.PluginInstallResult{OK: result.OK && !result.NeedsAllowBuilds, Error: result.Error, Log: strings.Join(logs, "\n")}
		if !result.NeedsAllowBuilds {
			return out
		}
		if err := ctx.Err(); err != nil {
			out.Error = err.Error()
			return out
		}
		builds := installctrl.NormalizeAllowBuilds(result.AllowBuilds)
		if len(builds) == 0 {
			out.Error = i18n.T(i18n.KeyCmdImportBuildListInvalid)
			return out
		}
		slices.Sort(builds)
		key := strings.Join(builds, "\x00")
		if seenBuilds[key] {
			out.Error = i18n.T(i18n.KeyCmdImportBuildRepeated)
			return out
		}
		if approve == nil || !approve(append([]string(nil), builds...)) {
			out.Error = i18n.T(i18n.KeyCmdImportBuildDeclined)
			return out
		}
		seenBuilds[key] = true
		approved = builds
	}
}
