# 用户可见的失败文案不携带原始错误文本

屏幕上出现的失败说明由 `internal/errcopy` 生成；原始错误只进日志，或进显式的诊断面板。这条规则在 2026-11 的文案整备中确立，并由此前的 #9、#12、#18、#21、#23 逐层落地。

## 转换入口

`errcopy.Classify` 归类，`errcopy.Sentence(err)` 给出当前语言的整句，`errcopy.SentenceText(raw)` 给只留了字符串的载体，`errcopy.Under(lead, err)` 把调用方的动作说明放在原因前面（`分支列表加载失败：找不到需要的文件或位置……`）。

因果表在 `internal/errcopy/errcopy.go`。新增一类失败时在那里加 pattern 与两种语言的句子，不在调用方拼字符串；两个调用方各自拼，就会像 2026-11 的启动页那样，同一件事给出两个不同的下一步。

## 守卫在哪里

| 守卫 | 覆盖 | 失败条件 |
| --- | --- | --- |
| `internal/ui/files/carrier_guard_test.go` | 文件面板 | 命中本包 carrier 正则且含 `.Error()` |
| `internal/ui/chat/copy_test.go` | 对话面 | 同上 |
| `internal/ui/composer/copy_test.go` | 输入区 | 同上 |
| `internal/ui/i18n/carriers_test.go` | 见下表 | 按包一条 carrier 正则 |
| `cmd/whaleisle/carriers_test.go` | 命令层 | 任何非比较、非标记的 `.Error()` |
| `internal/errcopy/carriers_ratchet_test.go` | `internal/store/**`、`internal/app/**`、`internal/ui/sidebar` | 视图状态/快照字段被赋值为 `.Error()` 或 `fmt.Sprint*`/`Errorf` 包装的错误文本即违规；已有点位记在 `testdata/carriers_baseline.txt`，数量只许下降 |

`carriers_test.go` 覆盖 `remote`、`terminal`、`diffsurface`、`theme`、`frame`、`projects`、`im`、`schedule`、`assistant`、`widgets/markdown`、`svgimage`。

## 新增一个 carrier

1. **包已在扫描表里**：把字段名、调用或标签加进该包那一条正则即可。
2. **新包**：在扫描表里加一行 `{dir, carrier}`。不加就永远扫不到——`svgimage` 曾经把解码错误画成占位文字，正是因为表里没有它；守卫只能证明它认识的包，证明不了规则。
3. **命令层不用登记**：严格守卫会直接失败，改好即可。

## 两个标记

- `verbatim-diagnostics`：诊断面板里逐字显示原文的行，例如 diff 进度面板的“复制完整错误”。面板本身是技术细节区，正文仍是本地化句子。
- `raw-ok:`：命令层写日志而非屏幕的记录（崩溃、启动失败、桥接报告）。后面必须写理由。

两个标记都写在那一行末尾，守卫按行跳过，评审时一眼能看到豁免，而不是靠记忆。

## store/app 层：错误不进快照

`internal/store/**`、`internal/app/**` 与 `internal/ui/sidebar` 的视图状态字段携带 `*state.Fault`（`internal/state`）：边界处 `state.FaultOf(err)` 归类，原始错误留在边界日志，界面渲染 `Fault.Sentence(...)`。ratchet 守卫扫描赋值给字段的 `.Error()` 与 `fmt.Sprint*`/`Errorf` 错误文本；减少违规后运行 `go test ./internal/errcopy -run CarriersRatchet -args -update-carriers` 更新基线，新增违规直接失败。

## 验证

```bash
go test ./internal/ui/files/ ./internal/ui/chat/ ./internal/ui/composer/ \
        ./internal/ui/i18n/ ./cmd/whaleisle/ -run 'CarriersNeverTakeARawError|NeverPaintsARawError'
```

格式门（`Main quality` 的 Check formatting）只检查本次改动过的 Go 文件，因此 `internal/ui/i18n/catalog_misc.go` 这类历史上未格式化的文件，一旦被改动就必须整体 gofmt；新增 key 时优先放进已经干净的小目录文件，避免把格式噪声带进文案提交。
