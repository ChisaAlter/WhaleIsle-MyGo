# 对话组件画廊（软件渲染，非实机窗口）

本目录的 `transcript-*.png` 由 `ui.NewTester` 软件渲染生成，用于给此前只有测试、没有视觉证据的对话组件留下可复核的图。**它们不是实机截图**：字体走同一套 theme token，但没有真实文本整形、系统阴影与 DPI 缩放；焦点环、指针 hover 态也不会出现。

生成方式（默认 skip，必须显式给目录）：

```powershell
$env:CHAT_GALLERY_DIR = (Resolve-Path docs\review-20261020\evidence\after).Path
go test -count=1 ./internal/ui/chat        -run TestTranscriptGallery
go test -count=1 ./internal/ui/interaction -run TestTranscriptGallery
```

| PNG | 画布 | 内容 | 生成测试 |
|---|---|---|---|
| `transcript-user-attachments.png` | 1200×760 | 用户行：240×64 PDF 附件卡（类型图标 + 文件名 + `pdf · 2.3 MB`）、坏图 danger 失败卡（`图片不存在或无法读取`）、两张 64 DIP 多图瓦片与气泡文本 | `internal/ui/chat/gallery_test.go` `TestTranscriptGalleryUserAttachments` |
| `transcript-assistant-markdown.png` | 1200×1150 | 助手 Markdown 块阶梯：h2、无序列表（18px 标记槽）、引用（2px 左线 + 14px 缩进）、行内代码 chip、Go 代码卡（语言标签 + 换行/复制图标 + 4 空格缩进 + 语法高亮）、三列表格（表头底色、内容比例列宽） | `TestTranscriptGalleryAssistantMarkdown` |
| `transcript-tools.png` | 1200×520 | 展开的 bash 工具行（单行 disclosure 保留摘要 + `命令`/`输出` 结构化 body + 右下角复制座位）与折叠的 read 行（图标 + 标题 + 圆点 + 摘要 + chevron，24px） | `TestTranscriptGalleryToolRows` |
| `transcript-approval-card.png` | 1200×360 | 审批卡：warn 左边条 + warn 边框、StateDot + `等待你的确认`、15/24 标题、`bash` 等宽命令块（含复制座位）、右对齐 `拒绝`（outline→danger）/ `允许`（primary） | `internal/ui/interaction/gallery_test.go` `TestTranscriptGalleryApprovalCard` |
| `transcript-question-card.png` | 1200×700 | 提问卡：StateDot 表头、15/24 问题、三个选项卡（20px 序号徽章 + 描述）、自定义回答占位、`‹ 1 / 3 ›` 翻页器、`跳过此题` + `下一步`（未作答禁用）、`暂时隐藏` / `拒绝` | `TestTranscriptGalleryQuestionCard` |
| `transcript-queue-dock.png` | 1200×260 | 队列 dock：标题 + 条数、两行 preview（单行截断）+ 图标动作（编辑 / 插话 / 删除） | `internal/ui/chat/gallery_test.go` `TestTranscriptGalleryQueueDock` |
| `transcript-turn-rail.png` | 1200×300 | 回合轨：4 个刻度（12px 座位内 4px 条），当前回合 accent 实心 6px | `TestTranscriptGalleryTurnRail` |

## 本轮按图修复的缺陷

1. `chat/image.go` 坏图不再回落成 `[image] path` 原始文本，改为 danger 失败卡（`KeyChatImageMissing` + 文件名）。
2. `chat/image.go` `paintImageFailure` / 灯箱遮罩的 `ui.RGBA` 误传 0–255，导致实心红块与纯黑遮罩；改为 0–1 alpha。
3. `widgets/markdown/codeblock.go` 代码卡展开制表符为 4 空格、标签页不再吞掉缩进。
4. `widgets/markdown/highlight.go` 去掉末尾多余换行 span，代码卡底部不再多出一行空白。
5. `chat/toolcard.go` 工具复制动作从整行宽按钮改为右对齐小座位。
6. `chat/toolrow.go` 展开后保留摘要（`KeepSummaryWhenOpen`），展开行仍能读出执行的命令。
7. `chat/turn_rail.go` 刻度改为 12px 座位内 4px 条，命中区域从 14×3 提升到 28×12。
8. `chat/queue_dock.go` 标题与计数去重（`待发送` + `2 条`）。
9. `interaction/card.go` 提问卡改中性表面（避免 accent 底像预选答案），审批卡保留 warn 底。
10. `interaction/card_chrome.go` 审批命令块自带复制座位，不再重复展开原始 args。
11. `interaction/question_navigation.go` 顶部与翻页器重复的 `1 / 3` 只保留翻页器。
12. `interaction/question_editing.go` 自定义回答加占位符。

## 仍未覆盖 / 需要 store 字段

- 提问卡的「推荐」徽章：`interaction.QuestionRowView` 没有 recommended 字段（store 不在本任务写入范围）。
- 工具行的终端块没有独立底色容器（命令有 `Field` 底、输出无底），与原版 `TerminalBlock` 的统一容器仍有差距。
- 表格单元格内边距只有水平 6px 左右，长中文列仍会折行；原版按内容自适应更宽。
