# 正文文件提及点击

助手已落定正文中的行内代码（如 `report.md`）可直接点击，保留原代码文字/底色，调用与交付卡相同的 `OnOpenDeliverable`。该入口继续使用所属会话 cwd、Files 草稿页签保护和 Host 兜底。渲染不执行任何打开动作。

## 词表依据

- 只接受原始事件中明确的 `data.turn`，与正文事件同轮且序号不晚于正文；不使用邻近行、编辑投影 `TurnSeq`、当前轮次或 `rootCallId` 猜归属。
- 显式 `deliverables/presented` 文件。
- 同轮根级 `tool/call` 与后续成功 append `tool/result` 通过 callId 配对；只接受参数完整的 write、edit、str_replace_editor 修改动作。
- 失败、未完成、读取、无效参数、跨轮和正文之后才确认的文件不进入词表。未知归属/PTC 子调用保持不可点击，等待具备权威 Location 归属后再扩展。
- 精确路径优先；否则只有唯一 basename 才可解析。重名、普通正文、代码块与未确认的扩展名不生成文件点击。

原版依据：`ui-deliverables/src/client/turn-deliverables.ts` 的 mutationPath、producedForClosing、producedFileMentions，以及 `docs/features/surfaces-work-loops.md` User paths 3。

验证：单个核心 fixture 覆盖事实过滤、精确/唯一basename解析、Windows路径basename、缺轮次惰性、真正行内点击回调、被动渲染和代码块惰性。Windows主程序编译由整合者统一执行。
