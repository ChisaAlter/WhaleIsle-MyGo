package settings

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"math"
	"time"
	"whaleisle/internal/desktop/voiceinput"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func voiceText(zh, en string) string {
	if i18n.Locale() == "en" {
		return en
	}
	return zh
}

type voiceSelectOption struct{ ID, Label string }

func voicePreferenceSelect(c *ui.Context, key, label, current string, options []voiceSelectOption, disabled bool, pick func(string)) {
	root := ui.Row(c.Key(key)).FillWidth().Gap(8).AlignItems(ui.Center)
	choice := ui.Local(root, "value", func() voiceDeviceChoice { return voiceDeviceChoice{Source: current, Value: current} })
	if choice.Source != current {
		choice.Source, choice.Value = current, current
	}
	root.Children(func() {
		ui.Text(c, label).FontSize(13)
		parts := ui.SelectBase(c, &choice.Value)
		caption := current
		for _, option := range options {
			if option.ID == current {
				caption = option.Label
				break
			}
		}
		parts.Trigger.Label(label).Grow(1).MinWidth(0).Disabled(disabled).Children(func() { ui.Text(c, caption).Grow(1).MinWidth(0).SingleLine(); ui.Text(c, "▾") })
		parts.Popup(func(panel ui.Element) {
			for _, option := range options {
				parts.Item(option.ID).Label(option.Label).Children(func() { ui.Text(c, option.Label) })
			}
		})
		if parts.Trigger.Changed() {
			value := choice.Value
			choice.Value = current
			pick(value)
		}
	})
}

// VoicePreparationView consumes the Host's actual readiness mirror; changing
// preferences never installs a model. Prepare and Cancel are explicit actions.
func VoicePreparationView(c *ui.Context, store *session.Store, microphone *voiceinput.Microphone) {
	if store == nil {
		ui.Text(c, voiceText("语音 Host 未连接", "Speech Host offline"))
		return
	}
	owner, state := store.BindSpeechSettings()
	ui.Column(c.Key("voice-preparation:" + owner.Key())).FillWidth().Gap(12).Children(func() {
		for _, message := range []string{state.ConnectionError, state.Error} {
			if message != "" {
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					ui.Text(c, message).FontSize(12).TextColor(c.Theme().Text).Grow(1).MinWidth(0)
					if state.Error == message && ui.Button(c, voiceText("关闭提示", "Dismiss")).Clicked() {
						owner.DismissError()
					}
				})
			}
		}
		if !state.Connected && ui.Button(c, voiceText("重新连接语音服务", "Reconnect speech service")).Clicked() {
			owner.Retry()
		}
		catalog := state.Catalog
		if catalog == nil {
			ui.Text(c, voiceText("正在读取语音服务…", "Loading speech services…"))
			return
		}
		VoiceInputDeviceView(c, microphone)
		providers := []voiceSelectOption{}
		languages := []voiceSelectOption{}
		location := ""
		for _, provider := range catalog.Providers {
			providers = append(providers, voiceSelectOption{provider.Id, provider.Name})
			if provider.Id == catalog.Selection.ProviderId {
				location = provider.Location
				for _, language := range provider.Languages {
					languages = append(languages, voiceSelectOption{language, voiceLanguage(language)})
				}
			}
		}
		voicePreferenceSelect(c, "speech-provider", voiceText("识别服务", "Provider"), catalog.Selection.ProviderId, providers, !state.Connected || state.Saving, func(id string) { owner.Configure(id, "") })
		voicePreferenceSelect(c, "speech-language", voiceText("识别语言", "Language"), catalog.Selection.Language, languages, !state.Connected || state.Saving, func(language string) { owner.Configure("", language) })
		if location == "cloud" {
			ui.Text(c, voiceText("录音将发送至所选云端识别服务。", "Audio is sent to the selected cloud recognizer.")).FontSize(12)
		} else {
			ui.Text(c, voiceText("使用 Host 本地识别服务。", "Recognition runs on the Host.")).FontSize(12)
		}
		for _, provider := range catalog.Providers {
			voicePreparationCard(c, provider.Id, provider.Name, provider.Location, provider.Preparation, provider.DownloadSources, provider.SetupEstimate, state.Connected, state.Submitting[provider.Id], owner)
		}
	})
}

func voiceLanguage(language string) string {
	labels := map[string][2]string{"auto": {"自动识别", "Automatic"}, "zh": {"中文", "Chinese"}, "en": {"英语", "English"}, "yue": {"粤语", "Cantonese"}, "ja": {"日语", "Japanese"}, "ko": {"韩语", "Korean"}}
	if label, ok := labels[language]; ok {
		return voiceText(label[0], label[1])
	}
	return language
}

type voicePreparationState struct {
	Phase, Message, Resource string
	CompletedBytes           float64
	TotalBytes, StartedAt    *float64
	Steps                    []struct {
		Kind, Status string
		StartedAt    *float64
	}
	Download *struct {
		Resource, Source, Reason, Code string
		Status                         *int
	}
}

func voicePhase(phase string) string {
	labels := map[string][2]string{"unprepared": {"首次使用需要准备识别模型", "Prepare recognition models before first use"}, "checking": {"正在检查本地资源…", "Checking local resources…"}, "downloading": {"正在下载…", "Downloading…"}, "loading": {"正在加载…", "Loading…"}, "waking": {"正在唤醒本地语音…", "Waking local speech recognition…"}, "cancelling": {"正在取消…", "Cancelling…"}, "cancelled": {"准备已取消", "Preparation cancelled"}, "failed": {"准备失败", "Preparation failed"}, "ready": {"已就绪", "Ready"}, "standby": {"待机", "Standby"}}
	if label, ok := labels[phase]; ok {
		return voiceText(label[0], label[1])
	}
	return phase
}
func voiceStep(kind string) string {
	labels := map[string][2]string{"check": {"检查资源", "Check resources"}, "model": {"识别模型", "Recognition model"}, "vad": {"语音活动检测模型", "Voice activity model"}, "verify": {"校验资源", "Verify resources"}, "load": {"加载识别服务", "Load recognizer"}}
	if label, ok := labels[kind]; ok {
		return voiceText(label[0], label[1])
	}
	return kind
}

func voiceStepStatus(status string) string {
	labels := map[string][2]string{"pending": {"未开始", "Not started"}, "running": {"进行中", "In progress"}, "complete": {"已完成", "Completed"}, "failed": {"失败", "Failed"}, "cancelled": {"已取消", "Cancelled"}}
	if label, ok := labels[status]; ok {
		return voiceText(label[0], label[1])
	}
	return status
}

func voicePreparationCard(c *ui.Context, id, name, location string, raw json.RawMessage, sources []string, estimate *api.SpeechCatalogProvidersItemSetupEstimate, connected, submitting bool, owner *session.SpeechSettingsOwner) {
	var state voicePreparationState
	if json.Unmarshal(raw, &state) != nil {
		ui.Text(c, voiceText("无法读取语音准备状态", "Could not read speech preparation state"))
		return
	}
	root := ui.Column(c.Key("speech-provider:"+id)).FillWidth().Gap(8).Padding(12).Border(1, c.Theme().Border).Radius(8)
	expanded := ui.Local(root, "steps", func() bool { return false })
	source := ui.Local(root, "source", func() string {
		if len(sources) == 1 {
			return sources[0]
		}
		return ""
	})
	dismissed := ui.Local(root, "dismissed", func() string { return "" })
	selectedSource := ""
	for _, choice := range sources {
		if choice == *source {
			selectedSource = choice
			break
		}
	}
	if len(sources) == 1 {
		selectedSource = sources[0]
	}
	preparing := state.Phase == "checking" || state.Phase == "downloading" || state.Phase == "loading" || state.Phase == "waking" || state.Phase == "cancelling"
	canPrepare := state.Phase == "unprepared" || state.Phase == "cancelled" || state.Phase == "failed"
	root.Children(func() {
		ui.Text(c, name).FontSize(14).Bold()
		if state.Phase == "unprepared" && location == "host-local" && estimate != nil {
			ui.Text(c, fmt.Sprintf(voiceText("本地识别准备估算：磁盘 %.1f GB；内存 %.1f GB；约 %.0f–%.0f 分钟", "Local setup estimate: %.1f GB disk; %.1f GB memory; about %.0f–%.0f minutes"), estimate.RecommendedDiskBytes/1e9, estimate.ExpectedMemoryBytes/1e9, estimate.MinimumMinutes, estimate.MaximumMinutes)).FontSize(12)
			ui.Text(c, voiceText("仅为提供方估算，实际资源与耗时取决于运行 Host 的机器。", "Provider estimates depend on the machine running the Host.")).FontSize(11).TextColor(c.Theme().TextMuted)
		}
		failure := state.Message
		if state.Download != nil {
			download := state.Download
			failure = fmt.Sprintf("%s · %s", download.Resource, download.Reason)
			advice := voiceText("请检查 Host 的网络、磁盘空间与模型目录权限。", "Check the Host network, disk space and model-directory permissions.")
			switch download.Reason {
			case "dns":
				advice = voiceText("请检查 Host 的 DNS 与代理设置。", "Check DNS and proxy settings on the Host.")
			case "certificate":
				advice = voiceText("请检查 Host 系统时间、证书与代理设置。", "Check the Host clock, trusted certificates and proxy settings.")
			case "integrity":
				advice = voiceText("请重试下载；已完成校验的文件会保留。", "Retry the download; verified files are retained.")
			case "storage":
				advice = voiceText("请检查 Host 剩余磁盘空间与模型目录写入权限。", "Check Host disk space and model-directory write permission.")
			case "network", "timeout", "http":
				advice = voiceText("请检查 Host 能否访问下载来源及其模型服务，确认网络与代理后重试。", "Check that the Host can reach the download source and model service, then retry.")
			}
			if download.Status != nil {
				failure += fmt.Sprintf(" (HTTP %d)", *download.Status)
			}
			failure += "\n" + advice + "\n" + voiceText("下载来源：", "Download source: ") + download.Source
			if download.Code != "" {
				failure += "\n" + download.Code
			}
		}
		if state.Phase == "failed" && failure != "" && *dismissed != failure {
			ui.Row(c).FillWidth().Gap(6).Children(func() {
				ui.Text(c, failure).Grow(1).MinWidth(0).FontSize(12).TextColor(c.Theme().Danger).Selectable()
				if ui.Button(c, voiceText("关闭提示", "Dismiss")).Clicked() {
					*dismissed = failure
				}
			})
		}
		summary := voicePhase(state.Phase)
		startedAt := state.StartedAt
		for _, step := range state.Steps {
			if step.Status == "running" {
				summary = voiceStep(step.Kind)
				if step.StartedAt != nil {
					startedAt = step.StartedAt
				}
				break
			}
		}
		if !connected {
			summary = voiceText("正在重新连接…", "Reconnecting…")
		}
		metric := ""
		if state.Phase == "downloading" {
			metric = fmt.Sprintf(voiceText("已下载 %.1f MB", "Downloaded %.1f MB"), state.CompletedBytes/1e6)
			if state.TotalBytes != nil && *state.TotalBytes > 0 {
				metric = fmt.Sprintf("%.1f / %.1f MB · %.0f%%", state.CompletedBytes/1e6, *state.TotalBytes/1e6, math.Floor(state.CompletedBytes / *state.TotalBytes * 100))
			}
		} else if preparing && startedAt != nil {
			metric = fmt.Sprintf(voiceText("已等待 %.0f 秒", "Waiting %.0f seconds"), math.Max(0, float64(time.Now().UnixMilli())-*startedAt)/1000)
			c.After(time.Second)
		}
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			if len(state.Steps) > 0 {
				arrow := "▸ "
				if *expanded {
					arrow = "▾ "
				}
				if ui.Button(c, arrow+summary).Grow(1).MinWidth(0).Clicked() {
					*expanded = !*expanded
				}
			} else {
				ui.Text(c, summary).Grow(1).MinWidth(0).FontSize(12)
			}
			if metric != "" {
				ui.Text(c, metric).FontSize(11).TextColor(c.Theme().TextMuted)
			}
		})
		if *expanded {
			for _, step := range state.Steps {
				ui.Text(c, voiceStep(step.Kind)+" · "+voiceStepStatus(step.Status)).FontSize(12)
			}
		}
		if state.Phase == "downloading" && state.TotalBytes != nil && *state.TotalBytes > 0 {
			ui.Progress(c, math.Max(0, math.Min(1, state.CompletedBytes / *state.TotalBytes))).FillWidth().Label(voiceText("下载进度", "Download progress"))
		}
		if state.Resource != "" && state.Phase == "downloading" {
			ui.Text(c, state.Resource).FontSize(11).TextColor(c.Theme().TextMuted)
		}
		if canPrepare && len(sources) > 0 {
			options := []voiceSelectOption{}
			if len(sources) > 1 {
				options = append(options, voiceSelectOption{"", voiceText("自动选择（推荐）", "Automatic (recommended)")})
			}
			for _, origin := range sources {
				label := origin
				if origin == "https://huggingface.co" {
					label = "Hugging Face"
				}
				if origin == "https://hf-mirror.com" {
					label = voiceText("HF-Mirror（国内镜像）", "HF-Mirror")
				}
				options = append(options, voiceSelectOption{origin, label})
			}
			voicePreferenceSelect(c, "download-source", voiceText("模型下载源", "Model download source"), selectedSource, options, !connected || submitting || len(sources) == 1, func(value string) { *source = value })
			if selectedSource == "" {
				ui.Text(c, voiceText("优先响应较快的源；失败后由提供方尝试其他源。", "Prefer a responding source; the provider may try another if a download fails.")).FontSize(11)
			} else {
				ui.Text(c, voiceText("仅从所选源下载；已下载并通过校验的文件会复用。", "Download only from the selected source; verified files are reused.")).FontSize(11)
			}
		}
		ui.Row(c).Gap(8).Children(func() {
			if canPrepare && ui.Button(c, voiceText("准备或重试准备", "Prepare / retry preparation")).Disabled(!connected || submitting).Clicked() {
				*dismissed = ""
				owner.Prepare(id, selectedSource)
			}
			if preparing && state.Phase != "waking" && ui.Button(c, voiceText("取消准备", "Cancel preparation")).Disabled(!connected || submitting || state.Phase == "cancelling").Clicked() {
				owner.CancelPreparation(id)
			}
		})
	})
}
