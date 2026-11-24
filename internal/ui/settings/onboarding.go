package settings

import (
	"context"
	"embed"
	"sync"

	"whaleisle/internal/desktop/media"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
)

//go:embed assets/onboarding-*
var onboardingArt embed.FS
var onboardingIcons sync.Map

// onboardingArtMaxW/H bound the decoded onboarding illustration to 2x its
// 500x420 DIP display box so a high-res lossless WebP never allocates tens of
// megabytes or blocks a UI frame.
const (
	onboardingArtMaxW = 1000
	onboardingArtMaxH = 840
)

type onboardingMediaHolder struct {
	owner *media.Owner
}

// onboardingImage requests an embedded onboarding illustration through the
// asynchronous media pipeline and returns the cached bitmap once decoded.
// Never decodes WebP synchronously on the UI thread.
func onboardingImage(c *ui.Context, name string) *ui.Bitmap {
	var owner *media.Owner
	if c != nil {
		holder := ui.Local(c.Root(), "onboarding-media-owner", func() onboardingMediaHolder {
			return onboardingMediaHolder{owner: media.Default().NewOwner(nil, c.Invalidate)}
		})
		if holder.owner == nil {
			holder.owner = media.Default().NewOwner(nil, c.Invalidate)
		} else {
			holder.owner.SetNotify(c.Invalidate)
		}
		owner = holder.owner
	}
	key := media.Key{ID: "settings:onboarding:" + name}.Sized(onboardingArtMaxW, onboardingArtMaxH)
	src := media.Source{
		Kind: ".webp",
		Load: func(context.Context) ([]byte, error) {
			return onboardingArt.ReadFile("assets/" + name)
		},
	}
	return media.Default().Request(owner, key, src).Bitmap()
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
	t := tk(c)
	colors := theme.Resolve(themeID).Active(systemDark)
	fg, muted, bg := t.Foreground, t.MutedForeground, t.Background
	topPad := max(design.Sp(10), settingsTitleBarTopInset(c.TitleBar()))
	sidePad := design.Sp(10)
	if vw, _ := c.Size(); vw > 0 && vw < 640 {
		sidePad = design.Sp(5)
	}
	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(bg)
		chrome.FrostSurface(panel.WidthPercent(100).HeightPercent(100).MinWidth(0).MinHeight(0), colors, systemDark, bg, 0).Children(func() {
			if s.Loading {
				ui.Column(c).Fill().Center().Padding(topPad, sidePad, design.Sp(10), sidePad).Children(func() {
					ui.Text(c, providerCopy("正在读取设置…", "Loading settings…")).TextColor(muted)
				})
				return
			}
			scroll := ui.Scroll(c.Key("onboarding-scroll")).FillWidth().Grow(1).MinWidth(0).MinHeight(0).
				Padding(topPad, sidePad, design.Sp(10), sidePad)
			scroll.Children(func() {
				anchor := ui.Column(c.Key("desktop-onboarding/" + s.Progress.Step)).FillWidth().MinWidth(0).Gap(design.Sp(5))
				dialog := ui.Local(anchor, "confirm", func() string { return "" })
				busy := s.Saving || s.Loading
				goStep := func(step string) { controller.Update(map[string]string{"step": step}) }
				title, hint := "", ""
				switch s.Progress.Step {
				case "welcome":
					title = providerCopy("欢迎使用 鲸屿 MyGo", "Welcome to Whale Isle MyGo")
					hint = providerCopy("鲸屿 MyGo 会以本地文件夹作为工作区，通过调用各种工具，读写本机文件，完成搜索整理资料、制作文档表格、编写代码、排查问题等各种任务。", "Whale Isle MyGo works in a local folder and uses tools to read and write files on your computer. It can help you research and organize information, create documents and spreadsheets, write code, troubleshoot issues, and more.")
				case "credit":
					title = providerCopy("准备可用额度", "Add credits")
					hint = providerCopy("鲸屿 MyGo 执行任务时，会根据模型与工具使用的 token 量扣除额度。请提前添加额度，避免任务在执行过程中被迫中断。你的余额只会在 Agent 实际执行任务时消耗。", "Whale Isle MyGo charges credits based on the tokens used by models and tools. Add credits in advance to keep your tasks running without interruption. Your balance is only used while the agent is working on a task.")
				case "purpose":
					title = providerCopy("你希望 鲸屿 MyGo 帮你做什么？", "What would you like Whale Isle MyGo to help with?")
					hint = providerCopy("我们会根据你的选择调整界面和工具，以更适合你的工作方式。", "We’ll tailor the interface and tools to fit the way you work.")
				case "process":
					title = providerCopy("你希望看到多少工作过程？", "How much detail would you like to see?")
					hint = providerCopy("这只会影响工作过程的展示方式，不会影响 鲸屿 MyGo 的工作能力。", "This only changes how progress is shown, not what Whale Isle MyGo can do.")
				}
				anchor.Children(func() {
					if s.Progress.Step != "welcome" && s.Progress.Step != "credit" {
						ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(30)).FontWeight(design.WeightSemibold).TextColor(fg).Focusable().AutoFocus()
						ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(15)).TextColor(muted)
					}
					switch s.Progress.Step {
					case "welcome", "credit":
						name := "onboarding-welcome"
						if s.Progress.Step == "credit" {
							name = "onboarding-recharge"
						}
						if i18n.Locale() == i18n.LocaleZH {
							name += "-zh"
						}
						if systemDark {
							name += "-dark"
						}
						art := func() {
							if img := onboardingImage(c, name+".webp"); img != nil {
								ui.Image(c, img).Width(500).MaxWidthPercent(100).Height(420).Fit(ui.Contain)
							} else {
								coss.Skeleton(c).Width(500).MaxWidthPercent(100).Height(420).Radius(design.RadiusXL)
							}
						}
						ui.Row(c).FillWidth().Wrap().Gap(design.Sp(10)).AlignItems(ui.Center).Children(func() {
							if s.Progress.Step == "credit" {
								art()
							}
							ui.Column(c).Width(480).MaxWidthPercent(100).Gap(design.Sp(6)).Children(func() {
								ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(28)).FontWeight(design.WeightNormal).Focusable().AutoFocus().TextColor(fg)
								ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(28)).TextColor(muted)
								ui.Row(c).FillWidth().Wrap().Gap(design.Sp(3)).Children(func() {
									if s.Progress.Step == "welcome" {
										if coss.Button(c, providerCopy("开始设置", "Get started"), coss.ButtonOptions{Size: coss.ButtonLG, Disabled: busy}).Clicked() {
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
										if coss.Button(c, providerCopy("继续", "Continue"), coss.ButtonOptions{Size: coss.ButtonLG, Disabled: busy}).Clicked() {
											goStep("purpose")
										}
										if coss.Button(c, providerCopy("去充值", "Add credits"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonLG, Disabled: busy || !canRecharge}).Clicked() {
											recharge()
										}
									} else {
										if coss.Button(c, providerCopy("去充值", "Add credits"), coss.ButtonOptions{Size: coss.ButtonLG, Disabled: busy || !canRecharge}).Clicked() {
											recharge()
										}
										if coss.Button(c, providerCopy("下一步", "Next"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonLG, Disabled: busy}).Clicked() {
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
						ui.Row(c).FillWidth().Wrap().Gap(design.Sp(4)).Children(func() {
							for _, value := range []string{"office", "development"} {
								selected := s.Progress.Purpose == value || s.Progress.Purpose == "both"
								label, cardHint := providerCopy("办公与创作", "Office & creative work"), providerCopy("编辑文档、整理数据和制作演示文稿等", "Edit documents, organize data, create presentations, and more")
								if value == "development" {
									label, cardHint = providerCopy("代码与开发", "Coding & development"), providerCopy("修改代码、调试问题、运行命令和管理项目文件等", "Edit code, debug issues, run commands, manage project files, and more")
								}
								card := kSurface(c, ui.Column(c.Key(value)).Width(400).MaxWidthPercent(100).MinWidth(0).MinHeight(222).Padding(design.Sp(8)).Gap(design.Sp(3)))
								card.Children(func() {
									onboardingIcon(c, map[string]string{"office": "office", "development": "code"}[value])
									if kCheckbox(c, &selected, label, busy).Changed() {
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
									kBody(c, cardHint).TextColor(muted)
								})
							}
						})
						if s.Progress.Purpose != "" && coss.Button(c, providerCopy("继续", "Continue"), coss.ButtonOptions{Size: coss.ButtonLG, Disabled: busy}).AlignSelf(ui.Start).Clicked() {
							if s.Progress.Purpose == "office" {
								controller.Complete("completed")
							} else {
								goStep("process")
							}
						}
					case "process":
						buttons := map[string]ui.Element{}
						focusTarget := ""
						ui.Row(c).FillWidth().Wrap().Gap(design.Sp(3)).Children(func() {
							for _, option := range []struct{ value, zh, en, zhHint, enHint string }{{"compact", "聚焦结果", "Results only", "只看结果，保持界面简洁明了", "See just the results in a clean, simple interface"}, {"standard", "关键细节", "Key details", "结果优先，仅展示重要步骤和关键操作", "Focus on results, with only the most important steps and actions shown"}, {"detailed", "完整过程", "Full details", "查看完整工作过程，方便调试和排查问题", "See the full process for easier debugging and troubleshooting"}} {
								card := kSurface(c, ui.Column(c.Key(option.value)).Width(320).MaxWidthPercent(100).MinHeight(222).Padding(design.Sp(8)).Gap(design.Sp(2.5)))
								card.Children(func() {
									onboardingIcon(c, option.value)
									label := providerCopy(option.zh, option.en)
									chosen := s.Progress.Process == option.value
									if chosen {
										label = "✓ " + label
									}
									button := kChoiceButton(c, label, chosen, busy).FillWidth()
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
									kBody(c, providerCopy(option.zhHint, option.enHint)).TextColor(muted)
								})
							}
						})
						if next, ok := buttons[focusTarget]; ok {
							next.Focus()
						}
						if s.Progress.Process != "" && coss.Button(c, providerCopy("进入应用", "Enter Whale Isle MyGo"), coss.ButtonOptions{Size: coss.ButtonLG, Disabled: busy}).AlignSelf(ui.Start).Clicked() {
							controller.Complete("completed")
						}
					}
					if s.Error != "" {
						settingsFailureLine(c, "onboarding:error", providerCopy("设置没能完成，请重试。", "Setup could not finish. Try again."), s.Error)
						if kButtonDisabled(c, providerCopy("重试", "Retry"), busy).AlignSelf(ui.Start).Clicked() {
							controller.Retry()
						}
					}
					if s.Progress.Step != "welcome" {
						ui.Row(c).FillWidth().Gap(design.Sp(3)).Children(func() {
							if kButtonDisabled(c, providerCopy("返回", "Back"), busy).Clicked() {
								previous := map[string]string{"credit": "welcome", "purpose": "credit", "process": "purpose"}
								goStep(previous[s.Progress.Step])
							}
							ui.Spacer(c)
							if kGhost(c, providerCopy("跳过设置", "Skip setup"), busy).Clicked() {
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
						chrome.DialogSurfaceTheme(c, mask, box, c.Theme(), 440, design.Sp(7)).Gap(design.Sp(3.5)).Children(func() {
							message := providerCopy("你可以随时前往 设置 → 通用设置 中调整工作过程、性能与用量展示方式或开启代码工作工具。", "You can change how progress, performance, and usage are displayed, or enable Coding Tools, anytime in Settings → General.")
							if kind != "skip" {
								message = providerCopy("没有可用额度时，鲸屿 MyGo 将无法开始新的任务。你可以稍后前往 个人中心 → 账号与余额 进行充值。", "Whale Isle MyGo can’t start new tasks without credits. You can add them later in Profile → Accounts.")
							}
							kBody(c, message)
							ui.Row(c).FillWidth().Wrap().Gap(design.Sp(2)).Children(func() {
								secondary := providerCopy("继续设置", "Continue setup")
								if kind != "skip" {
									secondary = providerCopy("我知道了", "Got it")
								}
								if kButtonDisabled(c, secondary, busy).Clicked() {
									if kind == "credit" {
										goStep("purpose")
									} else if kind == "credit-skip" {
										controller.Complete("skipped")
									}
									*dialog = ""
								}
								if kind != "skip" {
									if kPrimary(c, providerCopy("前往充值", "Add credits"), busy || account == nil || account.OnOpenTopUp == nil).Clicked() {
										*dialog = ""
										account.OnOpenTopUp(account.TopUpURL)
									}
								} else if kPrimary(c, providerCopy("跳过并进入", "Skip and enter"), busy).Clicked() {
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
	})
	return true
}

func onboardingIcon(c *ui.Context, name string) {
	if cached, ok := onboardingIcons.Load(name); ok {
		if icon, _ := cached.(*ui.SVG); icon != nil {
			ui.Icon(c, icon).Size(36, 36)
		}
		return
	}
	data, err := onboardingArt.ReadFile("assets/onboarding-" + name + ".svg")
	if err != nil {
		return
	}
	icon, err := ui.ParseSVG(data)
	if err == nil {
		onboardingIcons.Store(name, icon)
		ui.Icon(c, icon).Size(36, 36)
	}
}
