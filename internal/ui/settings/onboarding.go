package settings

import (
	"embed"
	"sync"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

//go:embed assets/onboarding-*
var onboardingArt embed.FS
var onboardingImages sync.Map

func onboardingImage(name string) *ui.Bitmap {
	if image, ok := onboardingImages.Load(name); ok {
		bitmap, _ := image.(*ui.Bitmap)
		return bitmap
	}
	data, err := onboardingArt.ReadFile("assets/" + name)
	if err != nil {
		return nil
	}
	bitmap, err := ui.DecodeBitmap(data)
	if err != nil {
		return nil
	}
	onboardingImages.Store(name, bitmap)
	return bitmap
}

// DesktopOnboarding renders the installation's original resumable introduction.
// The returned logical visibility lets the shell relinquish its modal/input
// ownership. Account and top-up actions keep their existing application owner.
func DesktopOnboarding(c *ui.Context, controller *store.DesktopOnboarding, account *AccountAccess, themeID string, systemDark bool) bool {
	if controller == nil {
		return false
	}
	s := controller.Snapshot()
	if !s.Visible {
		return false
	}
	colors := theme.Resolve(themeID).Active(systemDark)
	fg, muted, bg := chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.BG)
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(bg)
		panel.WidthPercent(100).HeightPercent(100).MinWidth(0).Padding(40).Gap(20).Radius(0).Background(bg).Children(func() {
			if s.Loading {
				ui.Column(c).Fill().Center().Children(func() { ui.Text(c, providerCopy("正在加载设置…", "Loading settings…")).TextColor(muted) })
				return
			}
			anchor := ui.Column(c.Key("desktop-onboarding/" + s.Progress.Step)).FillWidth().MinWidth(0).Gap(20)
			dialog := ui.Local(anchor, "confirm", func() string { return "" })
			busy := s.Saving || s.Loading
			goStep := func(step string) { controller.Update(map[string]string{"step": step}) }
			title, hint := "", ""
			switch s.Progress.Step {
			case "welcome":
				title = providerCopy("欢迎使用 Whale Isle", "Welcome to Whale Isle")
				hint = providerCopy("Whale Isle 会以本地文件夹作为工作区，通过调用各种工具，读写本机文件，完成搜索整理资料、制作文档表格、编写代码、排查问题等各种任务。", "Whale Isle works in a local folder and uses tools to read and write files on your computer. It can help you research and organize information, create documents and spreadsheets, write code, troubleshoot issues, and more.")
			case "credit":
				title = providerCopy("准备可用额度", "Add credits")
				hint = providerCopy("Whale Isle 执行任务时，会根据模型与工具使用的 token 量扣除额度。请提前添加额度，避免任务在执行过程中被迫中断。你的余额只会在 Agent 实际执行任务时消耗。", "Whale Isle charges credits based on the tokens used by models and tools. Add credits in advance to keep your tasks running without interruption. Your balance is only used while the agent is working on a task.")
			case "purpose":
				title = providerCopy("你希望 Whale Isle 帮你做什么？", "What would you like Whale Isle to help with?")
				hint = providerCopy("我们会根据你的选择调整界面和工具，以更适合你的工作方式。", "We’ll tailor the interface and tools to fit the way you work.")
			case "process":
				title = providerCopy("你希望看到多少工作过程？", "How much detail would you like to see?")
				hint = providerCopy("这只会影响工作过程的展示方式，不会影响 Whale Isle 的工作能力。", "This only changes how progress is shown, not what Whale Isle can do.")
			}
			anchor.Children(func() {
				if s.Progress.Step != "welcome" && s.Progress.Step != "credit" {
					ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(30)).FontWeight(600).TextColor(fg).Focusable().AutoFocus()
					ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(15)).TextColor(muted)
				}
				if s.Loading {
					ui.Text(c, providerCopy("正在读取设置…", "Loading settings…")).TextColor(muted)
				}
				switch s.Progress.Step {
				case "welcome", "credit":
					name := "onboarding-welcome"
					if s.Progress.Step == "credit" {
						name = "onboarding-recharge"
					}
					if i18n.Locale() == "zh" {
						name += "-zh"
					}
					if systemDark {
						name += "-dark"
					}
					art := func() {
						if image := onboardingImage(name + ".webp"); image != nil {
							ui.Image(c, image).Width(500).MaxWidthPercent(100).Height(420).Fit(ui.Contain)
						}
					}
					ui.Row(c).FillWidth().Wrap().Gap(40).AlignItems(ui.Center).Children(func() {
						if s.Progress.Step == "credit" {
							art()
						}
						ui.Column(c).Width(480).MaxWidthPercent(100).Gap(24).Children(func() {
							ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(28)).FontWeight(400).Focusable().AutoFocus().TextColor(fg)
							ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(28)).TextColor(muted)
							ui.Row(c).FillWidth().Wrap().Gap(12).Children(func() {
								if s.Progress.Step == "welcome" {
									if ui.PrimaryButton(c, providerCopy("开始设置", "Get started")).Disabled(busy).Clicked() {
										goStep("credit")
									}
									return
								}
								canRecharge := account != nil && account.OnOpenTopUp != nil
								recharge := func() {
									if canRecharge {
										account.OnOpenTopUp(account.TopUpURL)
									}
								}
								if s.CreditFunded {
									if ui.PrimaryButton(c, providerCopy("继续", "Continue")).Disabled(busy).Clicked() {
										goStep("purpose")
									}
									if ui.Button(c, providerCopy("去充值", "Add credits")).Disabled(busy || !canRecharge).Clicked() {
										recharge()
									}
								} else {
									if ui.PrimaryButton(c, providerCopy("去充值", "Add credits")).Disabled(busy || !canRecharge).Clicked() {
										recharge()
									}
									if ui.Button(c, providerCopy("下一步", "Next")).Disabled(busy).Clicked() {
										if s.CreditKnown && !s.CurrentCreditFunded {
											*dialog = "credit"
										} else {
											goStep("purpose")
										}
									}
								}
							})
						})
						if s.Progress.Step == "welcome" {
							art()
						}
					})
				case "purpose":
					ui.Row(c).FillWidth().Wrap().Gap(16).Children(func() {
						for _, value := range []string{"office", "development"} {
							selected := s.Progress.Purpose == value || s.Progress.Purpose == "both"
							label, hint := providerCopy("办公与创作", "Office & creative work"), providerCopy("编辑文档、整理数据和制作演示文稿等", "Edit documents, organize data, create presentations, and more")
							if value == "development" {
								label, hint = providerCopy("代码与开发", "Coding & development"), providerCopy("修改代码、调试问题、运行命令和管理项目文件等", "Edit code, debug issues, run commands, manage project files, and more")
							}
							ui.Column(c.Key(value)).Width(400).MaxWidthPercent(100).MinWidth(0).MinHeight(222).Padding(32).Gap(12).Radius(32).Background(chrome.Color(colors.Field)).Border(1, chrome.Color(colors.Border)).Children(func() {
								onboardingIcon(c, map[string]string{"office": "office", "development": "code"}[value])
								if ui.Checkbox(c, &selected, label).Disabled(busy).Changed() {
									office, development := s.Progress.Purpose == "office" || s.Progress.Purpose == "both", s.Progress.Purpose == "development" || s.Progress.Purpose == "both"
									if value == "office" {
										office = selected
									} else {
										development = selected
									}
									purpose := ""
									if office && development {
										purpose = "both"
									} else if office {
										purpose = "office"
									} else if development {
										purpose = "development"
									}
									controller.Update(map[string]string{"purpose": purpose})
								}
								ui.Text(c, hint).FillWidth().TextColor(muted)
							})
						}
					})
					if s.Progress.Purpose != "" && ui.PrimaryButton(c, providerCopy("继续", "Continue")).Disabled(busy).Clicked() {
						if s.Progress.Purpose == "office" {
							controller.Complete("completed")
						} else {
							goStep("process")
						}
					}
				case "process":
					buttons := map[string]ui.Element{}
					focusTarget := ""
					ui.Row(c).FillWidth().Wrap().Gap(12).Children(func() {
						for _, option := range []struct{ value, zh, en, zhHint, enHint string }{{"compact", "聚焦结果", "Results only", "只看结果，保持界面简洁明了", "See just the results in a clean, simple interface"}, {"standard", "关键细节", "Key details", "结果优先，仅展示重要步骤和关键操作", "Focus on results, with only the most important steps and actions shown"}, {"detailed", "完整过程", "Full details", "查看完整工作过程，方便调试和排查问题", "See the full process for easier debugging and troubleshooting"}} {
							ui.Column(c.Key(option.value)).Width(320).MaxWidthPercent(100).MinHeight(222).Padding(32).Gap(10).Radius(32).Background(chrome.Color(colors.Field)).Border(1, chrome.Color(colors.Border)).Children(func() {
								onboardingIcon(c, option.value)
								label := providerCopy(option.zh, option.en)
								if s.Progress.Process == option.value {
									label = "✓ " + label
								}
								button := ui.Button(c, label).FillWidth().Disabled(busy)
								buttons[option.value] = button
								if button.Clicked() {
									controller.Update(map[string]string{"process": option.value})
								}
								for _, key := range []ui.Key{ui.KeyLeft, ui.KeyRight, ui.KeyUp, ui.KeyDown} {
									if button.Shortcut(0, key) {
										values := []string{"compact", "standard", "detailed"}
										index := 0
										for i, v := range values {
											if v == option.value {
												index = i
											}
										}
										delta := 1
										if key == ui.KeyLeft || key == ui.KeyUp {
											delta = -1
										}
										controller.Update(map[string]string{"process": values[(index+delta+3)%3]})
										focusTarget = values[(index+delta+3)%3]
									}
								}
								ui.Text(c, providerCopy(option.zhHint, option.enHint)).FillWidth().TextColor(muted)
							})
						}
					})
					if next, ok := buttons[focusTarget]; ok {
						next.Focus()
					}
					if s.Progress.Process != "" && ui.PrimaryButton(c, providerCopy("进入应用", "Enter Whale Isle")).Disabled(busy).Clicked() {
						controller.Complete("completed")
					}
				}
				if s.Error != "" {
					ui.Text(c, s.Error).FillWidth().TextColor(c.Theme().Danger)
					if ui.Button(c, providerCopy("重试", "Retry")).Disabled(busy).Clicked() {
						controller.Retry()
					}
				}
				if s.Progress.Step != "welcome" {
					ui.Row(c).FillWidth().Gap(12).Children(func() {
						if ui.Button(c, providerCopy("返回", "Back")).Disabled(busy).Clicked() {
							previous := map[string]string{"credit": "welcome", "purpose": "credit", "process": "purpose"}
							goStep(previous[s.Progress.Step])
						}
						ui.Spacer(c)
						if ui.Button(c, providerCopy("跳过设置", "Skip setup")).Disabled(busy).Clicked() {
							*dialog = "skip"
							if s.Progress.Step == "credit" {
								*dialog = "credit-skip"
							}
						}
					})
				}
			})
			if *dialog != "" {
				kind := *dialog
				confirm := true
				ui.DialogBase(c, &confirm, func(mask, box ui.Element) {
					mask.Background(ui.RGBA(0, 0, 0, .28))
					box.Width(440).MaxWidthPercent(90).Padding(28).Gap(14).Radius(16).Background(bg).Children(func() {
						message := providerCopy("你可以随时前往 设置 → 通用设置 中调整工作过程、性能与用量展示方式或开启代码工作工具。", "You can change how progress, performance, and usage are displayed, or enable Coding Tools, anytime in Settings → General.")
						if kind != "skip" {
							message = providerCopy("没有可用额度时，Whale Isle 将无法开始新的任务。你可以稍后前往 个人中心 → 账号与余额 进行充值。", "Whale Isle can’t start new tasks without credits. You can add them later in Profile → Accounts.")
						}
						ui.Text(c, message).FillWidth().TextColor(fg)
						ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
							secondary := providerCopy("继续设置", "Continue setup")
							if kind != "skip" {
								secondary = providerCopy("我知道了", "Got it")
							}
							if ui.Button(c, secondary).Disabled(busy).Clicked() {
								if kind == "credit" {
									goStep("purpose")
								} else if kind == "credit-skip" {
									controller.Complete("skipped")
								}
								*dialog = ""
							}
							if kind != "skip" {
								if ui.PrimaryButton(c, providerCopy("前往充值", "Add credits")).Disabled(busy || account == nil || account.OnOpenTopUp == nil).Clicked() {
									*dialog = ""
									account.OnOpenTopUp(account.TopUpURL)
								}
							} else if ui.PrimaryButton(c, providerCopy("跳过并进入", "Skip and enter")).Disabled(busy).Clicked() {
								controller.Complete("skipped")
								*dialog = ""
							}
						})
					})
				})
				if !confirm {
					*dialog = ""
				}
			}
		})
	})
	return true
}

func onboardingIcon(c *ui.Context, name string) {
	data, err := onboardingArt.ReadFile("assets/onboarding-" + name + ".svg")
	if err != nil {
		return
	}
	icon, err := ui.ParseSVG(data)
	if err == nil {
		ui.Icon(c, icon).Size(40, 40)
	}
}
