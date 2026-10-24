# Round172 · 主题 id 映射与深色一致性

结论：round167 里「`theme=deepseek.dark` 仍是浅色」不是 `theme.Resolve` 的表格漏项，而是**主题来源被 Host 偏好覆盖**加上**未安装家族的 scheme 后缀被丢弃**两处叠加。本轮修掉后者（in-scope）、用表测试钉住 id 映射，并把前者以 file:line 结论留给 cmd 侧决定；MyGo 截图测量因时间盒与 profile 入口未定位而标记 blocked（未对真实 profile 做任何写入）。

## 1. 原生 id 事实

- Host 设置（`external/electron` 同源 `host-settings.json`）：`preference` = `light | dark | system`，另有两个家族字段 `activeLightThemeId` / `activeDarkThemeId`（默认 `"deepseek"`）。设置「外观」的三张卡写的是 `preference`；主题库卡片写的是家族 id（实机见 DeepSeek / Codex / Claude / 午夜 / 青瓷 / 暮紫）。
- 启动器 `config.json` 的 `theme` 只存家族（默认 `"deepseek"`），不存 `light/dark`。
- 因此原生可能持久化的组合是：家族（`deepseek`、`codex`、`claude`…）+ preference（light/dark/system），以及 MyGo 自己写进 `config.theme` 的复合 id（`deepseek.dark` 等，见 `internal/ui/settings/appearance.go` 的模式卡与 `cmd/whaleisle/settings_appearance_theme.go`）。

## 2. MyGo id 映射（修复后）

| 输入 id | family | scheme | 说明 |
|---|---|---|---|
| `""`、`deepseek`、`system` | deepseek | system | 跟随系统（首启默认） |
| `deepseek.dark`、`dark`、`DEEPSEEK.DARK` | deepseek | dark | 固定深色 |
| `deepseek.light`、`light` | deepseek | light | 固定浅色 |
| `codex.dark`、`claude.light`、`midnight.dark` | deepseek 调色板 | 取后缀 | **本轮修复**：未安装家族保留 scheme 后缀，不再退回 system |
| 无后缀的未知 id（`not-a-theme`） | deepseek | system | 回退，不猜 scheme |

`Resolve` 修改：[theme.go](<C:/AI/WhaleIsle-MyGo/internal/ui/theme/theme.go>) 的 `default` 分支（round172 前是 `keep system scheme so the window tracks the OS`，会把 `codex.dark` 画成浅色）。

## 3. 根因（file:line）

1. **Host 偏好覆盖 config 主题**：`cmd/whaleisle/theme_font_preferences.go:88-105` 的 `syncHostAppearance` 在 Host 设置就绪后无条件把 `s.themeID` 设为 `deepseek.light` / `deepseek.dark` / `deepseek`（来自 `h.Scheme`），并 `applyAppearance(acceptedScheme)`。Host 首启默认 `preference: "system"`（`internal/config/host_preferences.go:49`），所以 throwaway profile 里写 `config.theme = "deepseek.dark"`，一旦 Host 接受（`hostAppearanceAccepted = true`，同文件 :71）就被改成 system → 浅色。这正是 round167 的现象。
2. **「`theme=dark` 曾出现深色启动器」**：启动器 `BackgroundColor` 直接取 `theme.Resolve(cfg.Theme)`（`cmd/whaleisle/launcher_window.go:72`），`dark` 与 `deepseek.dark` 都得到 `#151517`；主窗随后被上面的 Host 同步改回 system，两个窗口因此判若两色——不是两个 id 行为不同。
3. **未安装家族丢 scheme**：旧的 `Resolve` default 分支对任何虚线 id 返回 system，因此原生可能持久化的 `codex.dark` 之类即使在 Host 侧已生效，MyGo 也画浅色。本轮已修。
4. `host_theme.go:90-101` 的 `SaveScheme` 已经把 `deepseek.dark`/`dark` 归一成 `light|dark|system` 再写 Host，这里没有 bug。

**留给 cmd 的决定**（不在本片写权限内）：`config.theme` 的复合 id 与 Host preference 谁优先。原生只有 Host preference 一个权威，所以 MyGo 在 Host 就绪后以 Host 为准是合理的；若仍希望 `config.theme=deepseek.dark` 在首启生效，需要在 cmd 侧把首次 Host 默认值与用户显式选择区分开（例如首启时把 config 的 scheme 写入 Host 一次），而不是在 `Resolve` 里绕。

## 4. 深色 token 对照（round168 原生值 vs MyGo 调色板）

| 元素 | 原生深色 | MyGo `deepseekDark` | 结论 |
|---|---|---|---|
| 前景/对话文字 | rgb(249,250,251) | `FG #f9fafb` | equal |
| 次要文字（行描述） | rgb(173,178,184) | `Muted #adb2b8` | equal |
| 强调/选中字 | rgb(143,175,253) | `Accent #7aaaff` = rgb(122,170,255) | 近似，偏暗一档（未改） |
| 输入框底 | rgba(35,35,36,.8) | `Field #232324` = rgb(35,35,36) | equal（alpha 由控件叠加） |
| 输入框描边 | rgba(255,255,255,.2) | `BorderL4 rgba(255,255,255,.20)` | equal |
| 胶囊/选中导航底 | rgb(53,54,56) | `Field #232324`（胶囊） / `Hover rgba(255,255,255,.08)`（选中导航，叠加后约 #2b2b2c） | **差异**：偏暗约 10 个灰阶；候选修法是给深色补一个 `module-platform #353638` 表面 token，但安装版 CSS 是单行打包、本轮未能在时间盒内确认 token 名，故未改调色板 |
| 停用徽标 | 底 rgb(53,54,56) / 字 rgb(207,211,214) | 同胶囊 / `Muted` | 随胶囊差异 |
| 行分隔线 | rgba(255,255,255,.2)（round160 浅色 1px 的深色对应） | `Border rgba(255,255,255,.12)` | 略淡（未改） |

浅色侧在 round160/164/168 已逐项对齐；本轮只改映射，不动调色板。

## 5. 截图测量：blocked

- 未生成临时 exe、未启动窗口：本轮 45 分钟盒内先完成了 id 映射修复与表测试，剩余时间不足以定位 `cmd/whaleisle` 的用户数据目录入口（`config.Store` 的 dir 由启动逻辑传入，未在 `main.go` 顶部找到可覆盖的 flag/env），而**不愿**用真实 profile 启动以免写入用户配置。
- 复现步骤（下一片可直接执行）：`go build -o %TEMP%\whaleisle-theme.exe ./cmd/whaleisle`；用临时 user-data 目录启动（先确认 dir 入口，必要时加仅测试用的 `--user-data-dir`）；分别写 `{"theme":"deepseek.dark"}`、`"dark"`、`"system"`，用 `scripts/window-capture.ps1 <Hwnd> <out.png>` 在 150% 下截启动器、主窗、设置；对照 §4 表。预期：修好后 `deepseek.dark` 与 `dark` 的首帧启动器同为深色；主窗颜色仍取决于 Host preference（§3.1），这是待决行为而非渲染 bug。

## 6. 测试与门禁

- [theme_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/theme/theme_test.go>) 新增 `TestResolveNativeFamilySuffixes`：12 个 id 的 scheme 表（含大小写/空白、`codex.dark`、`claude.light`、`midnight.dark`、未知无后缀），并钉住「固定深色在浅色系统下仍画 `#151517`」。
- [config_test.go](<C:/AI/WhaleIsle-MyGo/internal/config/config_test.go>) 新增 `TestLoadKeepsNativeThemeIDs`（9 个原生可持久化 id 原样保留）与 `TestLoadRejectsMalformedThemeID`（非法 id 回退 `DefaultTheme`）。
- `go test ./internal/ui/theme ./internal/config ./internal/ui/settings ./internal/ui/frame -count=1` → theme 2.472s、config 4.714s、settings 14.869s、frame 18.771s，全部 ok；`go vet ./internal/ui/theme ./internal/config ./internal/ui/settings` → 退出 0。
- 未提交、未构建 passN exe；未改 cmd/runtime/vendor/third_party；未启动原生应用。
