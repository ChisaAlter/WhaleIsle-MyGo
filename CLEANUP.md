# 清理规则与已核实的保留项

更新：2026-11-18。删除仓库内任何文件前先读本文件。本文件只记录判定规则和已核实结论，本身不授权删除。

## 判定规则

1. 只有证明**没有任何引用**才算死。以下任一成立即视为被引用：被 Go 代码、测试或夹具读写；落在 `//go:embed` 的目录或通配范围内；只在某个构建标签下被引用；被脚本、CI 或文档注释按名、按路径调用；被其它文档以链接引用，**包括经 `ARCHITECTURE.md` 的传递链**。
2. 每一条删除必须写明证明它的检查，指出命令与命中行。"看起来没用"不构成证据。
3. 一个逻辑批次一个提交。同批内部互相引用的文件一起删，否则会留下悬挂链接。
4. `tmp/` 证据目录与 `runtime-*/` 不在清理范围内。

## 已核实的保留项

| 路径 | 保留原因 |
| --- | --- |
| `third_party/mygo/cmd/mygo/testdata/dmg.DS_Store` | 夹具，不是编辑器残留；`third_party/mygo/cmd/mygo/dmg_test.go:26` 读取 |
| `tmp/go.mod` | 模块边界，把临时 Go 源码挡在应用模块外；`.gitignore` 用 `!/tmp/go.mod` 保住 |
| `reference/go.mod`、`reference/README.md` | 没有它 `go build ./...` 会走进只读快照 |
| `internal/ui/settings/assets/onboarding-*.webp` | 无按名引用，但整个目录由 `//go:embed assets/onboarding-*` 内嵌（`internal/ui/settings/onboarding.go:14`） |
| `internal/ui/pluginmanager/assets/*.svg` | 同上，`//go:embed assets/*.svg`（`internal/ui/pluginmanager/artwork.go:19`） |
| `resources/mobile/web/chisacode/parity.js` | 无按名引用，但 `scripts/mobile-runtime.mjs:74` 整目录打包带入，会随包发出；删前需运行时复核 |
| `internal/harness/link_windows.go`、`internal/harness/installctrl/junction_windows.go` | 两份 `mklink /J` 实现都在用：`internal/harness/link.go:43`、`installctrl/rollback.go:213` 及各测试 |
| 8 组重复资产（12.8MB，`icon.png` 与 `whale-head.png` 合计 4 份同内容副本，echarts 两个文件、mermaid、three、icons、`viewer.html` 各 2 份） | 每份由不同 loader 消费，去重属重构而非清理 |
| `docs/route-a-implementation-20261117/`、`docs/review-20261117/`、`docs/fix-plan-20261020.md` | 由 `docs/route-a-system-optimization-20261117.md` 与 `docs/review-20261020/CLOSURE-AUDIT.md:4` 按路径引用，而前者被 `ARCHITECTURE.md` 引用 |

**`docs/review-20261022/`、`docs/review-20261020/`（169MB）：决定保留。** 属被引用的证据而非杂物，压缩仓库不值得切断活的文档链。若日后要删，前置条件是先退役 `docs/route-a-system-optimization-20261117.md` 及其在 `ARCHITECTURE.md` 的入站链接——那是内容决定，不是清理决定；处理后需重跑引用检查。

## 2026-11-18 已合并的清理

| PR | 内容 |
| --- | --- |
| #24 | 零引用代码与脚本：`consumers.go:164 absoluteProjectPath`、`catalog_parity_agents.go:9 AgentsParityFormat`、`scripts/tool-{image,web}-evidence-summary.py` |
| #25 | 九条缺失的忽略规则（`.vscode/`、`.idea/`、`.claude/`、`.codex/`、`node_modules/`、`/coverage/`、`*.test`、`.DS_Store`、`*.tmp`），并用 `!` 反选保住 dmg 夹具 |
| #26 | 整个 2026-10-19 过期批次 27 个文件：12 篇 coverage、7 篇 depth、2 篇 native、4 篇同批记录、2 份 plan |

## 复核方法

- **引用扫描**：`rg` 搜全仓文件名与目录名，同时看文档链接与注释；跨文档引用要顺着 `ARCHITECTURE.md` 往下追一层。
- **忽略规则**：`git check-ignore -v` 逐条确认；`git ls-files -i -c --exclude-standard` 确认没有新被忽略的跟踪文件。
- **重复资产**：`sha256` 分组，再逐组确认每个副本的 loader。
- **过期报告**：统计其引用路径中已不存在的比例；本次 2026-10-19 批次为 25%–71%，超过一半即可判为过期。
