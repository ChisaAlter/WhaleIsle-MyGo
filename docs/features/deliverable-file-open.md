# 交付文件卡打开

用户明确点击交付卡文件时，使用发起会话的真实 `SelectedCwd` 解析相对路径；不使用应用进程 cwd，也不借配置的工作区或猜测 scratch 目录。

- cwd 内可由本地 Files 处理的既有文件：打开右栏 Files 的对应文件页签，复用目录树选择文件的生命周期。已打开文件保留草稿；重复点击当前文件不重置编辑内容。
- 点击 cwd 根目录：回到 Files 目录，保留已打开文件及其草稿。
- 缺 cwd、远程/异平台路径、越界路径和本地无法处理的路径：保持原路径，仅调用既有 Host `session/openWorkspacePath` 兜底；本地 Files 不同时打开。Host 不可用/失败通过既有会话状态反馈。
- 文件卡不接受 URL。只在点击时解析文件，检查符号链接目标仍在 cwd 内；没有保存或自动执行文件内容的行为。原有 `.git` 写保护和 dirty tab 关闭保护不变。

原版依据：`docs/features/surfaces-work-loops.md` 的 User paths 2、Invariants 的所属 Session cwd / Host fallback / dirty tabs。

范围说明：这里只补交付卡到现有 Files/Host 的路径和草稿闭环。PDF 已补 [应用内只读预览](local-pdf-preview.md)。原版 HTML/HTM/XHTML 交付卡 Browser mini 的呈现方式，以及收尾正文文件提及词表，仍未在此变更中实现，不能视作全部等价。

验证：单个临时文件 fixture 覆盖相对/绝对文件、重复打开、根目录返回、缺 cwd、越界、异平台路径、无效 URL 及 dirty close 保护。Windows 主程序编译由整合者统一执行。
