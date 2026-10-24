# Round174 · 主题渲染路径：配置主题优先，显式外观选择后来居上

round172 把「`theme=deepseek.dark` 仍画浅色」定位到 `syncHostAppearance` 用 Host 默认 scheme 覆盖配置主题；本轮按原生语义修掉这条覆盖路径，并用 150% 临时构建截图验证：`deepseek.dark` 的欢迎窗与启动器全深色，`deepseek.light` 浅色，`system` 跟随系统（当前系统为浅色）。

## 1. 原生语义（结论）

- Electron 启动器 `config.json` 的 `theme` 只有**家族**（默认 `"deepseek"`，见 `reference/electron/src/main/config.js`），不含 light/dark；光暗由 Host 外观设置 `preference`（`light | dark | system`）决定，首启默认 `system`。
- 也就是说原生没有「配置里钉死一个 scheme」的概念：首次运行没有 Host 偏好时就是 `system`（跟随 OS）；用户一改外观，Host preference 持久化并从此生效。
- MyGo 的 `config.theme` 可以携带 scheme（`deepseek.dark` / `deepseek.light`，由设置外观三卡与验收脚本写入），这是 MyGo 的扩展。因此本轮定下的规则与原生一致并可解释：
  - **没有显式外观选择**时，配置里钉的 scheme 生效（原生等价物是「用户已用启动参数指定」）；
  - **一旦用户在外观页做过选择**（Host preference 非 `system`，或在 config 记下 `appearanceExplicit`），该偏好优先，并且之后改 config 也不会把它抢回去；
  - `system` 偏好与无配置时都跟随 OS。

## 2. 修复

| 文件 | 改动 |
|---|---|
| [theme_startup.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/theme_startup.go>)（新增） | `startupTheme(cfg, hostScheme, hostAccepted)`：Host 已接受且（`cfg.AppearanceExplicit` 或 Host scheme ≠ system）→ 用 Host scheme；否则用 `cfg.Theme`（空则 `deepseek`）。`schemeThemeID` 把 `light/dark/system` 映射成 `deepseek.light` / `deepseek.dark` / `deepseek`。 |
| [theme_font_preferences.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/theme_font_preferences.go>) | `syncHostAppearance` 不再无条件用 Host scheme 覆盖 `s.themeID`，改走 `startupTheme`；Host 默认 `system` 不再吞掉配置的深色。 |
| [settings_appearance_theme.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/settings_appearance_theme.go>) | 用户在选择外观时把 `c.Theme = id` 与 `c.AppearanceExplicit = true` 一起写进 config（Host 已接受时同时写 Host `SaveScheme`），从此偏好优先且可解释。 |
| [config.go](<C:/AI/WhaleIsle-MyGo/internal/config/config.go>) | 新增 `AppearanceExplicit bool `json:"appearanceExplicit"``（默认 false，随 config.json 持久化）。 |

## 3. 表测试

[theme_startup_test.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/theme_startup_test.go>) 的 `TestStartupTheme`（10 例）：

| 场景 | 配置 theme | explicit | Host scheme | 结果 |
|---|---|---|---|---|
| 配置深色、Host 无偏好 | deepseek.dark | false | system | **deepseek.dark** |
| 配置浅色、Host 无偏好 | deepseek.light | false | system | deepseek.light |
| Host 显式浅色压过配置深色 | deepseek.dark | false | light | deepseek.light |
| Host 显式深色压过配置浅色 | deepseek.light | false | dark | deepseek.dark |
| 用户先选浅色、后改 config 深色 | deepseek.dark | true | light | deepseek.light |
| 用户显式选 system | deepseek.dark | true | system | deepseek |
| Host 未接受时用配置 | deepseek.dark | false | system | deepseek.dark |
| 配置 system 跟随 OS | deepseek | false | system | deepseek |
| 空 theme 回退 | "" | false | — | deepseek |
| Host scheme 大小写/空白 | deepseek.dark | false | " DARK " | deepseek.dark |

round172 的 `TestResolveNativeFamilySuffixes` 仍覆盖 `codex.dark` 这类未安装家族保留 scheme 的映射。

## 4. 150% 截图验证（临时构建，未覆盖 pass282）

构建：`go build -o %TEMP%\whaleisle-theme.exe ./cmd/whaleisle`；每次用独立 `WHALEISLE_USER_DATA` 临时目录 + 无 BOM 的 `config.json`，`scripts/window-capture.ps1` 抓图（脚本报告 DPI 144 = 150%）。

| 配置 | 窗口 | 截图 | 结果 |
|---|---|---|---|
| `theme: deepseek.dark` | 首启欢迎窗（未放 welcome-shown） | [round174-theme-dark-welcome-150.png](<evidence/screens/round174-theme-dark-welcome-150.png>) | 深色（900×1050 物理像素，DPI 144） |
| `theme: deepseek.dark` | 启动器（放 welcome-shown） | [round174-theme-dark-launcher-150.png](<evidence/screens/round174-theme-dark-launcher-150.png>) | 全深色启动器：侧栏 #1b1b1c、卡片 #232324、主按钮白底黑字（1590×990，DPI 144） |
| `theme: deepseek.light` | 启动器 | [round174-theme-light-launcher-150.png](<evidence/screens/round174-theme-light-launcher-150.png>) | 浅色 |
| `theme: system` | 启动器 | [round174-theme-system-launcher-150.png](<evidence/screens/round174-theme-system-launcher-150.png>) | 与 light 逐字节同尺寸（当前 OS 浅色），即跟随系统 |

（before 证据沿用 round173：同 profile 修前为浅色；修复后逐帧对比见上表。主窗（chat）需要启动 Harness 才会出现，本轮时间盒内未抓；启动器与主窗共用 `s.themeID` + `applyAppearance`，映射已由表测试与启动器截图覆盖。）

## 5. 残余

- **pill/选中导航深色表面**：原生 #353638，MyGo 用 `Field #232324` / `Hover` 叠加（约 #2b2b2c），仍偏暗约 10 个灰阶；候选修法是新增深色 module-platform token，安装版 CSS 为单行打包、token 名未确认，本轮未改（沿用 round172 结论）。
- **Accent**：原生深色选中文字 rgb(143,175,253)，MyGo `#7aaaff` = rgb(122,170,255)，略深一档（未改）。
- **行分隔线**：原生深色对应值比 MyGo `Border rgba(255,255,255,.12)` 略亮（未改）。
- 首次运行把 config scheme 写入 Host 偏好（而不是每次启动都靠 startupTheme 判断）也可以做，但那会改变 Host 数据；当前实现不动 Host 数据，仅在渲染层决定优先级。

## 6. 门禁

`go test ./cmd/whaleisle ./internal/ui/theme ./internal/config ./internal/ui/frame -count=1` → cmd 18.570s、theme 1.500s、config 3.026s、frame 12.482s，全部 ok；`go vet ./cmd/whaleisle ./internal/ui/theme ./internal/config` → 退出 0。临时 exe、临时 profile 与进程已清理；未提交、未构建 passN；未改原生应用、runtime/vendor、third_party。
