# Round125 · 定时任务详情卡片

本轮把详情卡片从目标扩展到定时任务（`schedule_create` / `schedule_update` / `schedule_list` / `schedule_delete`），构建为 [pass268](<C:/AI/WhaleIsle-MyGo/whaleisle-pass268.exe>) 并更新 [默认程序](<C:/AI/WhaleIsle-MyGo/whaleisle.exe>)。整体「布局、UI、功能、动效完全一致」目标仍在进行；上一轮提交为 `db93b61e`，参考 Electron、后端服务及 GUI 3080 未改动。

原版把这四个工具也渲染成同一张详情卡片，但内容不同：单条提醒有标题、`计划时间`、`重复`、`状态` 三个字段；`schedule_list` 每条一卡并在折叠行显示「N 个定时任务」，空列表显示「没有定时任务」；`schedule_delete` 以任务 id 为标题并显示「已删除」。本轮按安装版 `scheduleItem` 逐条实现，见 [模型](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_model.go>)、[新增文案](<C:/AI/WhaleIsle-MyGo/internal/ui/i18n/catalog_tool_details.go>)。

| 规则 | 实现 |
|---|---|
| 通用校验 | 记录须有非空 `id`/`prompt`、`scheduledAt` 能完整往返 ISO、`deliveryMode` 为 `host`/`session-local`、`state` 为 `scheduled`/`overdue` |
| `at` | 「单次」 |
| `after` | `afterSeconds` 为非负安全整数且非 0，否则整条无效 |
| `every` | `everySeconds` 同上；间隔按 86400/3600/60 整除选天/时/分，否则秒，组合成「每 N 小时」 |
| `daily` | 需要非空 `time`、`timeZone`；时间去掉多余的 `:00.000` → 「每天 09:30（Asia/Shanghai）」 |
| `weekly` | 再加 `weekdays` 必须是 1–7 的整数数组，按语言分隔符连接（`Mon, Wed` / `周一、周三`） |
| `cron` | 需要非空 `expression`、`timeZone` → 「Cron 0 9 * * *（Asia/Shanghai）」 |
| 标题 | 有 `title` 用它，否则用 `prompt`；两者都保留原始空格 |
| 状态 | 「等待触发」/「已到期，等待会话恢复」 |
| 列表 | 逐条校验，任一无效则整张卡片退回原始 OUT；汇总为「N 个定时任务」，空数组给汇总 + 空文案 |
| 删除 | `deleted:true` 才成卡；否则退回原始输出 |

时间文本没有 Intl 可用，Go 侧按 Node 的实际输出形状手工组合（en：`Oct 24, 2026, 03:59:30 PM GMT+8`；zh-CN：`2026年10月24日 GMT+8 15:59:30`），包含 12/24 小时换算、午夜/正午、GMT 偏移与秒级时区。我用 Node 24 在 `TZ=Asia/Shanghai` 下生成了参考样本并逐字节对照，含 `00:00:00`/`12:00:00`/`03:59:30` 三组边界，见 [安装版权威记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-details-pass268-schedule-authority.json>)。测试把 `time.Local` 固定为 +08:00，避免机器时区影响断言。

卡片渲染沿用上一轮已做好的结构：`4/0/4/4` 外边距、`.5px` 描边、radius16、`padding 10/14`、`5.5em + 1fr` 字段网格、条目间 `.5px` 分隔、列表 320 上限滚动，以及右上 Inspect。为支持列表汇总，模型新增 `Summary` 字段：有值时折叠行用它，否则用首条标题，再否则用空文案。[视图回归](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_view_test.go>) 覆盖：两条提醒（一条 `title`、一条回退 `prompt`）的折叠汇总「2 reminders」、展开后的三字段与两种状态；空列表「0 reminders」+「No reminders」且没有字段；删除卡片显示 id 与 Status/Deleted；非数组输入退回 OUT。模型侧 [golden 回归](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_details_model_test.go>) 覆盖七种频率、中英字段名、周几连接与越界、零/负数/不安全计数、无法往返的时间串、错误模式与状态、标题回退保留空格、列表空/无效、删除、root 与嵌套收据投影、独立切片与源不可变。

回归：Chat 63.777s、Frame 15.400s、i18n 0.769s 全通过，`mygo vet` 生命周期检查与受影响包 `go vet` 通过，见 [检查记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-details-pass268-schedule-checks.json>)。未修改 MyGo 框架。默认与版本化产物均为 86425600 字节、SHA256 `8C5CCB655BE5AF0B31CD475458FBD6B78A3C9EAC6DE45BDDA86E8ECD1990D04B`，见 [构建记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-details-pass268-build.json>)；无二进制提交。

边界与未完成项：时间文本只对 en / zh-CN 在 +08:00 时区做了 Node 对照，其它语言、其它时区、夏令时缩写与历史时区未验证；ICU 的字母时区缩写（如 `EDT`、`JST`）在本实现中会显示为等效的数字 GMT 偏移，只有 `GMT+8` 这类本来就与 Intl 一致；JS 可解析的扩展年份（如 `+010000-…`）Go 解析器不接受，这类记录会退回原始输出而不出卡；`weekdays` 的重复值按原版原样连接。本轮没有新的原生窗口运行（提醒行复用上一轮已验证的卡片与 Inspect 交互），也没有浏览器验收。其余详情族（作业、终端、团队、会话、Cordis、workflow/ralph 等）、`ToolDetails` 的 badge/description/lines/groups/unchanged/code 通用条目、Inspect 面板本身，以及字体基线、选择复制、滚动条、tooltip、presence 动效、Host 主页/物理输入、外壳材质、完整启动关闭等仍在后续轮次，不能据本轮判定全产品对齐完成。
