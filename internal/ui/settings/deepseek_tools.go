package settings

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	settingsstore "whaleisle/internal/store/settings"
	fonttheme "whaleisle/internal/ui/theme"
)

// DeepSeekToolsAccess is the native page for the Host's DeepSeek application API
// plugin: endpoint, credential reference, key state, three capability switches
// and the default voice. Values come from the Host namespace; edits are staged
// through the same namespace binding the generic editor uses.
type DeepSeekToolsAccess struct {
	Snapshot  settingsstore.DeepSeekToolsView
	Key       settingsstore.DeepSeekKeyView
	OnStage   func(field, value string)
	OnToggle  func(field string, value bool)
	OnSave    func()
	OnReload  func()
	OnRefresh func()
	OnSaveKey func(ref, value string)
}

func stagedText(s settingsstore.DeepSeekToolsView, field, stored string) string {
	if s.DirtyFields[field] {
		return s.Drafts[field]
	}
	return stored
}

func stagedBool(s settingsstore.DeepSeekToolsView, field string, stored bool) bool {
	if s.DirtyFields[field] {
		return s.Drafts[field] == "true"
	}
	return stored
}

func DeepSeekToolsPane(c *ui.Context, a *DeepSeekToolsAccess, fg, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.Snapshot
	body := func(text string) {
		ui.Text(c, text).FillWidth().FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
	}
	ui.Text(c, providerCopy("DeepSeek 应用接口", "DeepSeek app APIs")).FillWidth().FontSize(fonttheme.InterfaceSize(20)).FontWeight(600).TextColor(fg)
	body(providerCopy(
		"让 AI 生成语音、转录音频、识别文档。开启后，对应工具会出现在对话的可用工具里。",
		"Let the assistant generate speech, transcribe audio and read documents. Enabled tools appear in the conversation's tool list."))

	// Billing is stated before any switch, not discovered on the invoice.
	ui.Column(c.Key("deepseek-tools/billing")).FillWidth().MinWidth(0).Padding(12, 14).Gap(4).Radius(10).
		Border(.5, c.Theme().Border.Alpha(0.45)).Background(modelWell(c).Alpha(.55)).Children(func() {
		ui.Text(c, providerCopy("按量计费", "Billed per use")).FillWidth().FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
		body(providerCopy(
			"这些调用按 DeepSeek 官方价格另行计费。语音合成按字符计费，单次最多 10,000 计费字符，一个汉字按 2 个字符计算；转写与识别按上传的文件计费。开启前请确认账号余额。",
			"These calls bill separately at DeepSeek's official rates. Speech is billed per character, at most 10,000 billed characters per call with one Chinese character counted as two; transcription and recognition bill per uploaded file. Check the account balance before enabling."))
	})

	if s.Err != "" {
		ui.Text(c, providerCopy("刚才的更改没有生效。", "The last change did not go through.")+" "+errcopy.SentenceText(s.Err)+" "+
			providerCopy("请重新加载后再试。", "Reload and try again.")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(c.Theme().Danger)
	}

	switch {
	case !s.Connected:
		body(providerCopy(
			"设置还没有连上对话引擎，暂时读不到这项配置。等引擎启动后点「重新加载」；一直不出现就重启应用。",
			"Settings are not connected to the conversation engine yet, so this configuration cannot be read. Press Reload once the engine is up; restart the app if it never appears."))
		return
	case !s.Present:
		body(providerCopy(
			"这次启动的引擎没有加载 DeepSeek 应用接口插件，所以没有可编辑的配置。重启应用后再看；仍然没有的话，说明当前版本还没带上这个插件。",
			"This start did not load the DeepSeek app API plugin, so there is no configuration to edit. Restart the app and look again; if it is still missing, this build does not carry the plugin."))
		if ui.Button(c, providerCopy("重新加载", "Reload")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).
			Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).
			Disabled(s.Loading || a.OnRefresh == nil).Clicked() && a.OnRefresh != nil {
			a.OnRefresh()
		}
		return
	}

	disabled := s.Loading || s.Saving || !s.Writable
	textField := func(field, label, hint, stored string) {
		ui.Column(c.Key("deepseek-tools/" + field)).FillWidth().MinWidth(0).Gap(6).Children(func() {
			modelFieldLabel(c, label)
			value := stagedText(s, field, stored)
			input := ui.TextInput(c, &value).Grow(1).MinWidth(0).Height(36).Radius(8).
				Border(.5, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).
				FontSize(fonttheme.InterfaceSize(14)).Disabled(disabled)
			if input.Changed() && a.OnStage != nil {
				a.OnStage(field, value)
			}
			body(hint)
		})
	}

	textField("baseURL", providerCopy("API 基础地址", "API base URL"),
		providerCopy("以 /v1/tools 结尾，不要在末尾加 /tts 这类操作名。第三方网关要支持同样的接口。",
			"Ends with /v1/tools; do not append an operation such as /tts. A custom gateway must serve the same endpoints."),
		s.BaseURL)
	textField("apiKeyEnv", providerCopy("凭据引用", "Credential reference"),
		providerCopy("默认复用对话模型的密钥。改这里只影响这三个接口，不动对话模型的密钥。",
			"Reuses the chat model's key by default. Changing this affects only these three APIs, not the chat model's key."),
		s.APIKeyEnv)

	ui.Column(c.Key("deepseek-tools/api-key")).FillWidth().MinWidth(0).Gap(6).Children(func() {
		modelFieldLabel(c, providerCopy("API Key", "API key"))
		key := ui.Local(c.Root(), "deepseek-tools/api-key/value", func() string { return "" })
		placeholder := ""
		switch {
		case a.Key.Loading:
			placeholder = providerCopy("正在读取…", "Loading…")
		case a.Key.Configured:
			placeholder = providerCopy("已配置，留空保留", "Configured; leave blank to keep")
		default:
			placeholder = providerCopy("尚未配置", "Not configured yet")
		}
		apiKeyInput(c, "deepseek-tools", key, placeholder, disabled || !a.Key.Writable)
		body(providerCopy("留空表示保留已存的密钥。新密钥写进引擎的凭据存储，同一个引用下的其他提供方也会读到它。",
			"Leave blank to keep the stored key. A new key is written to the engine's credential store, and other providers using the same reference read it too."))
		if *key != "" && ui.Button(c, providerCopy("保存密钥", "Save key")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).
			FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Saving || a.OnSaveKey == nil).Clicked() && a.OnSaveKey != nil {
			a.OnSaveKey(stagedText(s, "apiKeyEnv", s.APIKeyEnv), *key)
			*key = ""
		}
		if a.Key.Err != "" {
			body(errcopy.SentenceText(a.Key.Err))
		}
	})

	toggle := func(field, caption, hint string, stored bool) {
		value := stagedBool(s, field, stored)
		settingsToggleRow(c, "deepseek-tools/"+field, caption, hint, &value, disabled, fg, muted, func(next bool) {
			if a.OnToggle != nil {
				a.OnToggle(field, next)
			}
		})
	}
	toggle("tts", providerCopy("语音合成（TTS）", "Text to speech (TTS)"),
		providerCopy("把文字生成可下载的 WAV 音频。", "Turns text into a downloadable WAV file."), s.TTS)

	ui.Row(c.Key("deepseek-tools/voice")).FillWidth().MinWidth(0).Gap(16).Padding(16, 0).MinHeight(56).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, providerCopy("默认音色", "Default voice")).FontSize(fonttheme.InterfaceSize(13)).TextColor(fg)
			body(providerCopy("不选的话每次沿用这里的音色。", "Used whenever a request does not name a voice."))
		})
		labels := []string{providerCopy("Stella · 女声", "Stella · Female"), providerCopy("Tide · 男声", "Tide · Male")}
		selected := labels[0]
		if stagedText(s, "voice", s.Voice) == "tide" {
			selected = labels[1]
		}
		colors := fonttheme.ResolveWithSystem(fonttheme.IDSystem, c.Theme().Dark).Active(c.Theme().Dark)
		ui.Select(c, &selected, labels).Width(200).Shrink(0).Height(32).Padding(0, 12).Radius(16).BorderWidth(0).
			Background(fonttheme.Color(colors.SelectorFill)).FontSize(fonttheme.InterfaceSize(13)).Disabled(disabled).OnChange(func() {
			if a.OnStage == nil {
				return
			}
			if selected == labels[1] {
				a.OnStage("voice", "tide")
				return
			}
			a.OnStage("voice", "stella")
		})
	})

	toggle("asr", providerCopy("音频转文字（ASR）", "Speech to text (ASR)"),
		providerCopy("上传音频识别文字：PCM WAV 或 Ogg Opus，最多 8 MiB、120 秒。不会替换本地麦克风输入。",
			"Uploads PCM WAV or Ogg Opus audio, up to 8 MiB and 120 seconds. Does not replace the local microphone."), s.ASR)
	toggle("ocr", providerCopy("文档识别（OCR）", "Document recognition (OCR)"),
		providerCopy("上传 PDF、Word、PowerPoint 或图片，最多 100 MiB、1,000 页；完整结果另外存成文件。",
			"Uploads PDF, Word, PowerPoint or images, up to 100 MiB and 1,000 pages; the full result is saved as a file."), s.OCR)
	toggle("filterMarkdown", providerCopy("朗读前移除 Markdown 格式", "Remove Markdown before reading"),
		providerCopy("标题符号、星号这类标记不会被念出来。", "Headings and asterisks are not read out."), s.FilterMarkdown)

	ui.Row(c.Key("deepseek-tools/actions")).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
		status := ""
		switch {
		case s.Loading:
			status = providerCopy("正在读取…", "Loading…")
		case s.Saving:
			status = providerCopy("正在保存…", "Saving…")
		case len(s.DirtyFields) > 0:
			status = providerCopy("有尚未保存的更改", "Unsaved changes")
		}
		if ui.Button(c, providerCopy("保存", "Save")).Height(32).Padding(0, 16).Radius(16).FontSize(fonttheme.InterfaceSize(13)).
			Disabled(disabled || a.OnSave == nil).Clicked() && a.OnSave != nil {
			a.OnSave()
		}
		if ui.Button(c, providerCopy("重新加载", "Reload")).Height(32).Padding(0, 16).Radius(16).
			Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).
			Disabled(s.Saving || a.OnReload == nil).Clicked() && a.OnReload != nil {
			a.OnReload()
		}
		if status != "" {
			body(status)
		}
	})
}
