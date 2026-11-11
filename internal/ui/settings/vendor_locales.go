package settings

import (
	"strings"

	"whaleisle/internal/ui/i18n"
)

// settingsVendorLocaleEntry holds aligned zh/en strings from the 0.3.13 vendor locales.
type settingsVendorLocaleEntry struct {
	ZH string
	EN string
}

var settingsVendorLocales = map[string]map[string]settingsVendorLocaleEntry{
	"dsh-client-ui-agent-preset": {
		"modeExplanation": {
			ZH: "模式说明",
			EN: "Mode details",
		},
		"howToUse": {
			ZH: "如何使用",
			EN: "How to use",
		},
		"guideSections": {
			ZH: "帮助内容",
			EN: "Guide sections",
		},
		"guideExampleTask": {
			ZH: "示例任务",
			EN: "Example task",
		},
		"guideCopy": {
			ZH: "复制",
			EN: "Copy",
		},
		"guideCopied": {
			ZH: "已复制",
			EN: "Copied",
		},
		"guideFootnotes": {
			ZH: "脚注",
			EN: "Footnotes",
		},
		"guideStandardIntro": {
			ZH: "新建任务时选择「标准模式」，说明要完成什么、相关文件在哪里，以及怎样判断任务完成。",
			EN: "Choose Standard mode when starting a new task. Describe what you want to accomplish, point to the relevant files, and explain how to check the result.",
		},
		"guideStandardExplanation": {
			ZH: `### 工作方式

Agent 直接调用工具来读写文件、检索资料和执行终端命令。包含 Skills、计划、目标、子 Agent、工作流和上下文压缩等能力。

### 什么时候选

日常编程、文件处理和资料整理可以从这里开始。标准模式也能编写脚本、批量处理文件；PTC 改变的是工具调用方式，批量任务并不必须使用 PTC。`,
			EN: `### How it works

The agent calls tools directly to read and edit files, search, and run terminal commands. It includes Skills, planning, goals, subagents, workflows, and context compaction.

### When to choose it

Start here for everyday coding, file work, and research. Standard mode can also write scripts and process files in batches. PTC changes how tool calls are organized; it is not required for batch tasks.`,
		},
		"guideStandardUsage": {
			ZH: `### 修复一个问题

> 搜索表单连续提交两次后，结果会消失。请定位原因、修复问题并运行相关测试，最后说明原因和修改内容。

预期产出：代码修改、相关测试结果，以及问题原因说明。

### 整理项目资料

> 阅读项目中的 Markdown 记录，整理已经达成的结论和仍待确认的问题，并附上对应文件链接。

预期产出：一份带来源引用的总结，方便回到原文核对。`,
			EN: `### Fix a bug

> Find out why submitting the search form twice makes the results disappear. Fix it and run the relevant tests. Explain the cause and what changed.

Expected output: a code change, the relevant test results, and an explanation of the cause.

### Organize project notes

> Read the Markdown notes in this project. Summarize the agreed decisions and open questions, with links to the source files.

Expected output: a summary with references that you can check against the original notes.`,
		},
		"guidePtcIntro": {
			ZH: "新建任务时选择「PTC 模式」，说明输入文件、处理规则和输出格式。代码由 Agent 编写。",
			EN: "Choose PTC mode when starting a new task. Specify the input files, processing rules, and output format. The agent writes the code.",
		},
		"guidePtcExplanation": {
			ZH: `### 怎样调用工具

PTC 是 Programmatic Tool Calling，即通过程序调用工具。当前内置预设让 Agent 通过 run_code 编写 TypeScript 程序，使用生成的工具 SDK 发起调用。程序可以组织循环、条件判断、错误处理，以及适合并发执行的调用。

### 哪些结果交给模型

工具返回的数据先交给程序，经过筛选、计算或合并，再通过输出或返回值交给模型；图片结果会另行附加。程序中的工具调用仍会被记录，也仍受工具权限约束。

### 与标准模式的区别

两种模式都能编程、批量处理文件。标准模式直接向模型提供各个工具；PTC 让模型用代码组织工具调用。当前 PTC 预设未启用 workflow 工具。速度和 token 用量取决于具体任务与结果处理方式。`,
			EN: `### How tools are called

PTC means Programmatic Tool Calling. In this built-in preset, the agent uses run_code to write a TypeScript program that calls tools through a generated SDK. The program can use loops, conditions, error handling, and concurrent calls where appropriate.

### What reaches the model

Tool results first reach the program, which can filter and combine them. The model receives what the program prints or returns; image results are attached separately. Nested tool calls are still recorded and remain subject to tool permissions.

### Compared with Standard mode

Both modes can handle coding and batch tasks. Standard mode exposes individual tools directly; PTC organizes tool calls in code. The current PTC preset leaves the workflow tool disabled. Speed and token use depend on the task and how the program handles its results.`,
		},
		"guidePtcUsage": {
			ZH: `### 批量检查配置文件

> 检查 configs/ 下所有 JSON 文件，按照 schema.json 找出缺失字段和不合法的值。每个问题写成 CSV 中的一行；读取失败的文件也记入报告，继续检查其余文件。保留原文件。

预期产出：问题汇总和一份 CSV 报告。程序可以对多份文件执行相同检查，处理单个文件的失败，再汇总结果。

### 汇总错误日志

> 分析 logs/ 下的日志，按服务和错误类型统计次数，列出出现最多的十类错误，每类保留一条示例。完整统计另存为 CSV。

预期产出：高频错误摘要和完整统计表。中间数据可以先在程序中聚合，再把汇总交给模型。`,
			EN: `### Check a set of configuration files

> Check all JSON files in configs/. List missing required fields and invalid values against schema.json. Save a CSV with one row per issue. Include unreadable files in the report and keep checking the rest. Leave the original files unchanged.

Expected output: an issue summary and a CSV report. The program can repeat the same checks, handle individual failures, and collect the results.

### Summarize error logs

> Analyze the log files in logs/. Group errors by service and error type. Show the ten most frequent groups and one example from each. Save the full counts to a CSV.

Expected output: the top error groups and a complete count table. Intermediate data can be aggregated in the program before the summary reaches the model.`,
		},
		"guideMinimalIntro": {
			ZH: "新建任务时选择「极简模式」。做对照测试时，保持模型、权限、任务输入和工作区起始状态一致。",
			EN: "Choose Minimal mode for a new task. For a comparison, hold the model, permissions, input, and starting workspace state constant across runs.",
		},
		"guideMinimalExplanation": {
			ZH: `### 保留哪些能力

提供持久 Shell 和 read_image 工具，并使用固定系统提示词。读取图片需要支持图片输入的模型。上下文压力达到阈值时自动压缩较早历史；Agent 空闲时可输入 /compact 手动压缩。内置预设不加载 Skills、计划，也不注入标准运行时上下文。

### 什么时候选

适合作为实验和对照测试的基线。Agent 仍能通过终端命令读写文件、运行脚本，但缺少管理长任务的内置辅助能力。工具少，不代表对新手更容易。`,
			EN: `### What is included

One persistent shell tool, read_image, and a fixed system prompt. Image reading requires an image-capable model. Older history is compacted automatically under context pressure; use /compact while the agent is idle to compact it manually. The built-in preset does not load Skills, planning, or the standard runtime context.

### When to choose it

Use it as a baseline for experiments and comparisons. It can still read files and execute scripts through shell commands, but offers fewer built-in ways to manage a long task. Fewer tools does not necessarily make it easier for a beginner.`,
		},
		"guideMinimalUsage": {
			ZH: `### 对比基础修复表现

> 运行这个项目的测试，找出失败原因，做最小修复，再运行相关测试并报告结果。

分别用标准模式和极简模式，从相同的工作区状态执行这条任务，对比完成情况、工具调用和最终修改。极简模式会通过终端命令完成这些操作。`,
			EN: `### Compare performance on a small bug fix

> Run the tests for this project, find the cause of the failure, and make the smallest fix. Run the relevant tests again and report the result.

Run the same task separately in Standard and Minimal modes from the same starting state. Compare task completion, tool calls, and the resulting changes. Minimal mode performs the work through terminal commands.`,
		},
		"guideCordisIntro": {
			ZH: "新建任务时选择「创造模式」，说明希望增加什么能力、从哪里使用，以及怎样验证效果。",
			EN: "Choose Creator mode for a new task. Describe the capability you want, where it should appear, and how you will verify it.",
		},
		"guideCordisExplanation": {
			ZH: `### 可以创造什么

创造模式具备标准任务工具，并增加运行时检查、持久化插件管理，以及 Cordis 插件和 Agent 预设的开发指引。可以编写插件来添加功能或界面，也可以组合工具和提示词，创建适合特定任务的模式。

### 插件与模式的关系

插件为 DSH 增加能力，例如工具、服务连接或界面入口。模式是一份 Agent 预设，用来选择任务可用的工具，并约定 Agent 的工作方式。自定义模式中也可以使用自己开发的插件。

### 怎样让成果生效

可以要求 Agent 完成安装并验证实际效果。插件可能即时加载，也可能需要重启，取决于修改内容；新建的模式在创建新任务时选择。`,
			EN: `### What you can create

Creator mode includes the standard task tools plus runtime inspection, persistent plugin management, and guidance for authoring Cordis plugins and agent presets. It can create a plugin that adds a capability or UI, or a preset that combines tools and prompts for a particular job.

### Plugins and modes

A plugin adds capabilities to DSH, such as a tool, a service connection, or a UI entry. A mode is an agent preset that selects tools and defines how the agent works in a task. A plugin can be included in a custom preset.

### How the result takes effect

Ask the agent to install and verify the result, not just generate source code. A plugin may load immediately or require a restart, depending on what it changes. A newly created preset is selected when starting a new task.`,
		},
		"guideCordisUsage": {
			ZH: `### 添加一个界面

> 帮我写一个 DSH 插件，在侧栏增加「项目笔记」入口，列出当前工作区的 Markdown 文件，点击后能预览内容。完成安装并验证页面能打开。

预期产出：带侧栏入口和预览页的插件，以及仍需完成的生效步骤。

### 添加一个工具

> 写一个插件，提供读取项目测试报告、汇总失败用例的工具。注册工具，并用一份示例报告验证调用结果。

预期产出：可调用的新工具，以及一次示例调用的验证结果。

### 创建自己的模式

> 基于标准模式创建「代码审查」模式，优先检查潜在错误和测试缺口，指出文件与行号，修改文件前先询问我。保存成可选择的预设。

预期产出：可在新任务中选择的自定义模式。审查要求用于指导 Agent，实际可执行的操作仍由权限设置决定。`,
			EN: `### Add a UI

> Create a DSH plugin that adds a project notes entry to the sidebar. Let me browse the Markdown files in this workspace and preview a selected note. Install it and verify that the page opens.

Expected output: an installed plugin with a working entry and preview page, plus any remaining activation steps.

### Add a tool

> Create a plugin with a tool that reads this project’s test report and summarizes the failed tests. Register it and verify it with a sample report.

Expected output: a plugin with a callable tool and a verified sample call.

### Create my own mode

> Create a “Code review” mode based on Standard mode. Have it prioritize potential bugs and test gaps, cite file paths and lines, and ask before modifying files. Save it as a selectable preset.

Expected output: a custom preset for new tasks. These review instructions guide the agent; permission settings determine which actions it can execute.`,
		},
		"builtInGroup": {
			ZH: "内置",
			EN: "Built-in",
		},
		"customGroup": {
			ZH: "自定义",
			EN: "Custom",
		},
		"sectionIntro": {
			ZH: "选择 Agent 的工具和工作方式。日常任务用「标准模式」，扩展 DSH 的能力用「创造模式」。",
			EN: "Choose the agent’s tools and how it works. Use Standard mode for everyday tasks, or Creator mode to add capabilities to DSH.",
		},
		"seatHint": {
			ZH: "选择新任务使用的 Agent 预设",
			EN: "Choose the agent preset for your new task",
		},
		"headerHint": {
			ZH: "本任务的 Agent 预设，在任务开始时确定",
			EN: "The agent preset chosen when this task started",
		},
		"nav": {
			ZH: "Agent 预设",
			EN: "Agent presets",
		},
		"setDefault": {
			ZH: "设为新任务默认",
			EN: "Set as new task default",
		},
		"view": {
			ZH: "查看配置",
			EN: "View configuration",
		},
		"presetStandardName": {
			ZH: "标准模式",
			EN: "Standard mode",
		},
		"presetStandardDescription": {
			ZH: "处理代码、文件和资料，适合大多数任务。Agent 会按需使用检索、编辑和终端等工具。",
			EN: "Work with code, files, and information. Suitable for most tasks, with search, editing, terminal commands, and other tools available as needed.",
		},
		"presetPtcName": {
			ZH: "PTC 模式",
			EN: "PTC mode",
		},
		"presetPtcDescription": {
			ZH: "包含标准模式的所有能力，更适合批量调用工具，并对结果进行筛选、整理、去重、统计或汇总的任务。",
			EN: "Includes all Standard mode capabilities. Better suited to tasks that call tools in batches and then filter, organize, deduplicate, count, or summarize the results.",
		},
		"presetMinimalName": {
			ZH: "极简模式",
			EN: "Minimal mode",
		},
		"presetMinimalDescription": {
			ZH: "Agent 使用终端和图片读取工具，并支持上下文压缩，适合测试和对比其基础表现。",
			EN: "The agent uses a terminal and image reading, with context compaction. Useful for testing and comparing its basic performance.",
		},
		"presetCordisName": {
			ZH: "创造模式",
			EN: "Creator mode",
		},
		"presetCordisDescription": {
			ZH: "用对话定制 DSH：让 Agent 编写插件，添加新功能或界面；也能组合工具和提示词，创建自己的模式。",
			EN: "Customize DSH through conversation. Let the agent write plugins that add features or UI, or combine tools and prompts to create your own mode.",
		},
		"inUse": {
			ZH: "新任务默认",
			EN: "New task default",
		},
		"noDescription": {
			ZH: "暂无描述。",
			EN: "No description.",
		},
		"brokenBadge": {
			ZH: "加载失败",
			EN: "Failed to load",
		},
		"switchRefused": {
			ZH: "无法切换到「{name}」：{reason}",
			EN: "Could not switch to {name}: {reason}",
		},
		"standardUnavailable": {
			ZH: "标准模式不可用，请恢复该模式或选择其他可用模式。",
			EN: "Standard mode is unavailable. Restore it or choose another available mode.",
		},
		"close": {
			ZH: "关闭",
			EN: "Close",
		},
		"creatorDraft": {
			ZH: "让 Agent 帮我创建预设模式",
			EN: "Let the agent help me create a preset",
		},
		"createPlugin": {
			ZH: "让 Agent 创建插件",
			EN: "Let the agent create a plugin",
		},
		"createPluginDescription": {
			ZH: "进入创造模式，制作属于你的 DSH 插件",
			EN: "Enter Creator mode and make your own DSH plugin",
		},
		"createPluginChecking": {
			ZH: "正在确认创造模式是否可用",
			EN: "Checking whether Creator mode is available",
		},
		"createPluginUnavailable": {
			ZH: "暂时不可用，请重新打开菜单重试",
			EN: "Temporarily unavailable. Reopen this menu to retry",
		},
		"createPluginMissing": {
			ZH: "当前配置未提供创造模式",
			EN: "Creator mode is not included in this configuration",
		},
	},
	"dsh-client-ui-settings-account": {
		"modelSignInRequired": {
			ZH: "当前模型暂不可用，请登录后再试",
			EN: "Model unavailable. Please sign in and try again.",
		},
		"sessionExpired": {
			ZH: "登录信息已失效，请重新登录",
			EN: "You have signed out of your account, please log in again.",
		},
		"onboardingArtworkLocale": {
			ZH: "zh",
			EN: "en",
		},
		"onboardingWelcome": {
			ZH: "欢迎使用",
			EN: "Welcome to",
		},
		"onboardingBrand": {
			ZH: "鲸屿 MyGo",
			EN: "Whale Isle MyGo",
		},
		"onboardingIntroduction": {
			ZH: "鲸屿 MyGo 会以本地文件夹作为工作区，通过调用各种工具，读写本机文件，完成搜索整理资料、制作文档表格、编写代码、排查问题等各种任务。",
			EN: "Whale Isle MyGo works in a local folder and uses tools to read and write files on your computer. It can help you research and organize information, create documents and spreadsheets, write code, troubleshoot issues, and more.",
		},
		"onboardingStart": {
			ZH: "开始设置",
			EN: "Get started",
		},
		"onboardingCredit": {
			ZH: "准备可用额度",
			EN: "Add credits",
		},
		"onboardingCreditDescription": {
			ZH: "鲸屿 MyGo 执行任务时，会根据模型与工具使用的 token 量扣除额度。请提前添加额度，避免任务在执行过程中被迫中断。你的余额只会在 Agent 实际执行任务时消耗。",
			EN: "Whale Isle MyGo charges credits based on the tokens used by models and tools. Add credits in advance to keep your tasks running without interruption. Your balance is only used while the agent is working on a task.",
		},
		"onboardingTopUp": {
			ZH: "去充值",
			EN: "Add credits",
		},
		"onboardingLater": {
			ZH: "下一步",
			EN: "Next",
		},
		"onboardingFundedTopUp": {
			ZH: "去充值",
			EN: "Add credits",
		},
		"onboardingPurposePrefix": {
			ZH: "你希望",
			EN: "What would you like",
		},
		"onboardingPurposeSuffix": {
			ZH: "帮你做什么？",
			EN: "to help with?",
		},
		"onboardingPurposeDescription": {
			ZH: "我们会根据你的选择调整界面和工具，以更适合你的工作方式。",
			EN: "We’ll tailor the interface and tools to fit the way you work.",
		},
		"onboardingOffice": {
			ZH: "办公与创作",
			EN: "Office & creative work",
		},
		"onboardingOfficeDescription": {
			ZH: "编辑文档、整理数据和制作演示文稿等",
			EN: "Edit documents, organize data, create presentations, and more",
		},
		"onboardingDevelopment": {
			ZH: "代码与开发",
			EN: "Coding & development",
		},
		"onboardingDevelopmentDescription": {
			ZH: "修改代码、调试问题、运行命令和管理项目文件等",
			EN: "Edit code, debug issues, run commands, manage project files, and more",
		},
		"onboardingContinue": {
			ZH: "继续",
			EN: "Continue",
		},
		"onboardingProcess": {
			ZH: "你希望看到多少工作过程？",
			EN: "How much detail would you like to see?",
		},
		"onboardingProcessDescription": {
			ZH: "这只会影响工作过程的展示方式，不会影响 鲸屿 MyGo 的工作能力。",
			EN: "This only changes how progress is shown, not what Whale Isle MyGo can do.",
		},
		"onboardingCompact": {
			ZH: "聚焦结果",
			EN: "Results only",
		},
		"onboardingCompactDescription": {
			ZH: "只看结果，保持界面简洁明了",
			EN: "See just the results in a clean, simple interface",
		},
		"onboardingStandard": {
			ZH: "关键细节",
			EN: "Key details",
		},
		"onboardingStandardDescription": {
			ZH: "结果优先，仅展示重要步骤和关键操作",
			EN: "Focus on results, with only the most important steps and actions shown",
		},
		"onboardingDetailed": {
			ZH: "完整过程",
			EN: "Full details",
		},
		"onboardingDetailedDescription": {
			ZH: "查看完整工作过程，方便调试和排查问题",
			EN: "See the full process for easier debugging and troubleshooting",
		},
		"onboardingEnter": {
			ZH: "进入应用",
			EN: "Open app",
		},
		"onboardingBack": {
			ZH: "上一步",
			EN: "Back",
		},
		"onboardingSkip": {
			ZH: "跳过",
			EN: "Skip",
		},
		"onboardingSkipTitle": {
			ZH: "跳过设置？",
			EN: "Skip setup?",
		},
		"onboardingSkipDescription": {
			ZH: "你可以随时前往 设置 → 通用设置 中调整工作过程、性能与用量展示方式或开启代码工作工具。",
			EN: "You can change how progress, performance, and usage are displayed, or enable Coding Tools, anytime in Settings → General.",
		},
		"onboardingKeepSetting": {
			ZH: "继续设置",
			EN: "Continue setup",
		},
		"onboardingNoCreditTitle": {
			ZH: "暂时跳过充值？",
			EN: "Skip adding credits?",
		},
		"onboardingNoCreditDescription": {
			ZH: "没有可用额度时，鲸屿 MyGo 将无法开始新的任务。你可以稍后前往 个人中心 → 账号与余额 进行充值。",
			EN: "Whale Isle MyGo can’t start new tasks without credits. You can add them later in Profile → Accounts.",
		},
		"onboardingUnderstood": {
			ZH: "我知道了",
			EN: "Got it",
		},
		"onboardingGoTopUp": {
			ZH: "前往充值",
			EN: "Add credits",
		},
		"onboardingSaveFailed": {
			ZH: "未能保存设置，请重试。",
			EN: "Could not save settings. Try again.",
		},
		"onboardingRetry": {
			ZH: "重试",
			EN: "Retry",
		},
		"onboardingLoading": {
			ZH: "正在加载设置…",
			EN: "Loading settings…",
		},
		"close": {
			ZH: "关闭",
			EN: "Close",
		},
		"addApiKey": {
			ZH: "添加 API Key",
			EN: "Add API Key",
		},
		"retry": {
			ZH: "重新登录",
			EN: "Sign in again",
		},
		"loginTitle": {
			ZH: "开始使用",
			EN: "Get started",
		},
		"loginDescription": {
			ZH: "登录 DeepSeek 账号，或添加 API Key，即可开始使用。你的项目和文件保存在本地。",
			EN: "Sign in to your DeepSeek account or add an API Key to get started. Your projects and files are stored locally.",
		},
		"browserTitle": {
			ZH: "等待登录",
			EN: "Waiting for sign in",
		},
		"browserPrompt": {
			ZH: "没有自动打开新页面？",
			EN: "Page did not open automatically? ",
		},
		"copyLink": {
			ZH: "复制登录链接",
			EN: "Copy sign-in link",
		},
		"copiedLink": {
			ZH: "链接已复制",
			EN: "Link copied",
		},
		"copyFailed": {
			ZH: "复制失败",
			EN: "Copy failed",
		},
		"browserDescription": {
			ZH: "，手动打开登录页完成登录。",
			EN: ", then open it in your browser to finish signing in.",
		},
		"timeoutTitle": {
			ZH: "登录已超时",
			EN: "Sign in timed out",
		},
		"timeoutDescription": {
			ZH: "请重新登录后继续操作。",
			EN: "Sign in again to continue.",
		},
		"failureTitle": {
			ZH: "登录失败",
			EN: "Could not sign in",
		},
		"platformFailed": {
			ZH: "操作未完成，请重试",
			EN: "Could not complete the operation. Try again.",
		},
		"platformRetry": {
			ZH: "重试",
			EN: "Retry",
		},
		"loading": {
			ZH: "加载中…",
			EN: "Loading…",
		},
		"backToHarness": {
			ZH: "返回 鲸屿 MyGo",
			EN: "Back to Whale Isle MyGo",
		},
		"settings": {
			ZH: "设置",
			EN: "Settings",
		},
		"contactUs": {
			ZH: "意见反馈",
			EN: "Feedback",
		},
		"contactUsSignedOut": {
			ZH: "联系我们",
			EN: "Contact us",
		},
		"menu": {
			ZH: "账号菜单",
			EN: "Account menu",
		},
		"nav": {
			ZH: "账号与余额",
			EN: "Account",
		},
		"signedIn": {
			ZH: "已登录 DeepSeek",
			EN: "Signed in to DeepSeek",
		},
		"signedOut": {
			ZH: "尚未登录",
			EN: "Not signed in",
		},
		"signIn": {
			ZH: "登录",
			EN: "Sign in",
		},
		"signOut": {
			ZH: "退出登录",
			EN: "Sign out",
		},
		"signOutUnknownDescription": {
			ZH: "暂时无法确认任务状态。退出登录可能会中断使用此账号的任务，是否继续？",
			EN: "Could not check running tasks. Signing out may interrupt tasks using this account. Sign out now?",
		},
		"signOutDescription": {
			ZH: "退出登录不会丢失任何数据，你仍可以登录此账号。",
			EN: "Signing out will not delete any data. You can sign in to this account again.",
		},
		"signOutRunningDescription": {
			ZH: "当前有进程正在执行中，点击“退出登录”，将会打断进程，是否即刻退出登录？",
			EN: "Tasks are currently running. Signing out will interrupt them. Sign out now?",
		},
		"cancel": {
			ZH: "取消",
			EN: "Cancel",
		},
		"open": {
			ZH: "打开浏览器",
			EN: "Open browser",
		},
		"initializing": {
			ZH: "正在发起登录…",
			EN: "Starting sign in…",
		},
		"waiting": {
			ZH: "请在浏览器中继续",
			EN: "Continue in your browser",
		},
		"completing": {
			ZH: "正在完成登录…",
			EN: "Completing sign in…",
		},
		"expired": {
			ZH: "登录已过期，请重试。",
			EN: "Sign in expired. Try again.",
		},
		"failed": {
			ZH: "操作未完成，请重试。",
			EN: "Could not complete the operation. Try again.",
		},
		"noResponse": {
			ZH: "操作未完成，可能是网络异常，请检查你的网络状况后重试",
			EN: "Something went wrong. Please check your network connection and try again.",
		},
		"settingsSignedOutTitle": {
			ZH: "当前未登录 DeepSeek 账号",
			EN: "You are not signed in to DeepSeek",
		},
		"settingsSignedOutDescription": {
			ZH: "登录 DeepSeek 账号获取专属 API Key",
			EN: "Sign in to DeepSeek to get your dedicated API Key",
		},
		"signInDescription": {
			ZH: "登录 DeepSeek 账号以开始使用",
			EN: "Use your DeepSeek account to get started.",
		},
		"profileUnavailable": {
			ZH: "账号资料暂不可用",
			EN: "Account details are not available yet.",
		},
		"balance": {
			ZH: "充值余额",
			EN: "Topped-up balance",
		},
		"bonusBalance": {
			ZH: "赠金余额",
			EN: "Granted balance",
		},
		"balanceUnavailable": {
			ZH: "前往开放平台查看",
			EN: "View on Platform",
		},
		"balanceSignedOut": {
			ZH: "登录后查看",
			EN: "Sign in to view",
		},
		"accountInfo": {
			ZH: "更多账号信息",
			EN: "More account information",
		},
		"more": {
			ZH: "更多",
			EN: "More",
		},
		"launcherLabel": {
			ZH: "设置与账号",
			EN: "Settings & account",
		},
		"usage": {
			ZH: "查询用量",
			EN: "View usage",
		},
		"topUp": {
			ZH: "充值",
			EN: "Top up",
		},
		"quotaTitle": {
			ZH: "当前无可用余额",
			EN: "No balance available",
		},
		"quotaDescription": {
			ZH: "没有可用额度时，鲸屿 MyGo 无法开始新的任务，是否前往充值？或者可以稍后前往 设置 → 账号与余额 进行充值。",
			EN: "Whale Isle MyGo cannot start a new task with this account when no balance is available. Would you like to top up? You can also top up later in Settings → Account.",
		},
		"quotaTopUp": {
			ZH: "去充值",
			EN: "Top up",
		},
		"bonusNoticeTitle": {
			ZH: "赠金已到账",
			EN: "Bonus credited",
		},
	},
	"dsh-client-ui-settings-agent-loop": {
		"title": {
			ZH: "Agent 循环",
			EN: "Agent loop",
		},
		"description": {
			ZH: "控制 Agent 派发工具调用的方式。",
			EN: "Control how the Agent dispatches tool calls.",
		},
		"maxParallel": {
			ZH: "并行工具调用数",
			EN: "Parallel tool calls",
		},
		"maxParallelHint": {
			ZH: "同一步内最多同时运行多少个可并行的调用。",
			EN: "Upper bound on parallel-safe calls running at once within one step.",
		},
		"overridden": {
			ZH: "已覆盖",
			EN: "Overridden",
		},
		"reset": {
			ZH: "恢复默认",
			EN: "Reset to default",
		},
		"readOnly": {
			ZH: "本部署的设置为只读。",
			EN: "This deployment stores settings read-only.",
		},
		"unavailable": {
			ZH: "该插件当前未加载，暂时无法配置。",
			EN: "This plugin is not loaded, so it cannot be configured right now.",
		},
		"save": {
			ZH: "保存",
			EN: "Save",
		},
		"saving": {
			ZH: "保存中…",
			EN: "Saving…",
		},
		"saveFailed": {
			ZH: "本部署没有接受这些值，已保留供你修改。",
			EN: "The deployment did not accept these values; they were left for you to correct.",
		},
		"invalidNumber": {
			ZH: "请填数字；留空表示使用默认值。",
			EN: "Enter a number, or leave blank to use the default.",
		},
	},
	"dsh-client-ui-settings-general": {
		"trigger": {
			ZH: "设置",
			EN: "Settings",
		},
		"shortcut.open": {
			ZH: "打开设置",
			EN: "Open settings",
		},
		"desktop.update.available": {
			ZH: "新版本",
			EN: "Update",
		},
		"desktop.update.checking": {
			ZH: "正在检查更新…",
			EN: "Checking for updates…",
		},
		"desktop.update.progress": {
			ZH: "{percent}%…",
			EN: "{percent}%…",
		},
		"desktop.update.verifying": {
			ZH: "正在校验更新文件…",
			EN: "Verifying update files…",
		},
		"desktop.update.installing": {
			ZH: "正在准备重启…",
			EN: "Preparing to restart…",
		},
		"desktop.update.ready": {
			ZH: "安装并重启",
			EN: "Install and Restart",
		},
		"desktop.update.retry": {
			ZH: "重试更新",
			EN: "Retry update",
		},
		"desktop.update.versionDetail": {
			ZH: "{label}：{version}",
			EN: "{label}: {version}",
		},
		"desktop.update.downloadDetail": {
			ZH: `正在下载更新：{percent}%
目标版本：{version}`,
			EN: `Downloading update: {percent}%
Target version: {version}`,
		},
		"desktop.update.checkFailed": {
			ZH: "检查更新失败，请稍后重试。",
			EN: "Could not check for updates. Please try again later.",
		},
		"desktop.update.downloadFailed": {
			ZH: "下载更新失败，请重试。",
			EN: "Could not download the update. Please try again.",
		},
		"desktop.update.installFailed": {
			ZH: "安装更新失败，请稍后重试。",
			EN: "Could not install the update. Please try again later.",
		},
		"desktop.update.checkNetworkFailed": {
			ZH: "检查更新失败，请检查网络连接后重试。",
			EN: "Could not check for updates. Check your connection and try again.",
		},
		"desktop.update.downloadNetworkFailed": {
			ZH: "下载更新失败，请检查网络连接后重试。",
			EN: "Could not download the update. Check your connection and try again.",
		},
		"desktop.update.installNetworkFailed": {
			ZH: "安装更新失败，请检查网络连接后重试。",
			EN: "Could not install the update. Check your connection and try again.",
		},
		"desktop.update.stopFailed": {
			ZH: "未能安全停止任务，更新尚未安装，请稍后重试。",
			EN: "Could not safely stop the tasks. The update has not been installed. Please try again later.",
		},
		"desktop.update.tasksChanged": {
			ZH: "有新任务开始运行，请重新确认是否停止任务并更新。",
			EN: "New tasks have started. Confirm again to stop the tasks and update.",
		},
		"desktop.update.tasksUnavailable": {
			ZH: "无法确认任务状态，请在工作区就绪后重试更新。",
			EN: "Task status is unavailable. Try updating again when the workspace is ready.",
		},
		"title": {
			ZH: "设置",
			EN: "Settings",
		},
		"close": {
			ZH: "关闭",
			EN: "Close",
		},
		"openDocument": {
			ZH: "打开配置文件",
			EN: "Open configuration file",
		},
		"openDocument.error": {
			ZH: "无法打开配置文件",
			EN: "Could not open configuration file",
		},
		"feedback.notSaved": {
			ZH: "设置未保存，请重试",
			EN: "Settings were not saved. Please try again.",
		},
		"feedback.restartCancelled": {
			ZH: "已取消重启，设置未更改。",
			EN: "Restart cancelled; the setting was not changed.",
		},
		"feedback.saveError": {
			ZH: "设置保存失败：{message}",
			EN: "Could not save settings: {message}",
		},
		"feedback.readError": {
			ZH: "设置读取失败：{message}",
			EN: "Could not read settings: {message}",
		},
		"general.nav": {
			ZH: "通用设置",
			EN: "General",
		},
		"general.currentVersion": {
			ZH: "Harness 版本：{version}",
			EN: "Harness version: {version}",
		},
		"developerTools.title": {
			ZH: "显示代码工作视图",
			EN: "Show coding view",
		},
		"developerTools.error": {
			ZH: "保存失败，请重试",
			EN: "Could not save. Please try again.",
		},
		"developerTools.description": {
			ZH: "开启后显示轨迹、本轮代码差异，新对话中的 Agent 预设切换",
			EN: "Shows trajectory, code diffs, and Agent preset switching in new chats",
		},
		"interface.nav": {
			ZH: "界面设置",
			EN: "Interface",
		},
		"closeBehavior.title": {
			ZH: "关闭窗口时",
			EN: "When closing the window",
		},
		"closeBehavior.description": {
			ZH: "点标题栏关闭时：最小化到托盘，或退出并停止本机 Harness 服务。",
			EN: "The title-bar close button can hide the app in the tray, or quit and stop the local Harness service.",
		},
		"closeBehavior.tray": {
			ZH: "最小化到托盘",
			EN: "Minimize to tray",
		},
		"closeBehavior.quit": {
			ZH: "直接退出",
			EN: "Quit",
		},
		"autoStartDesktop.title": {
			ZH: "启动时",
			EN: "At startup",
		},
		"autoStartDesktop.description": {
			ZH: "是：直接打开桌面端，启动器可从托盘右键「打开启动器」进入。否：先打开启动器，再手动启动桌面端。",
			EN: "Yes opens the desktop directly; open the launcher from the tray context menu when needed. No opens the launcher first so you can start the desktop manually.",
		},
		"autoStartDesktop.direct": {
			ZH: "直接启动桌面端",
			EN: "Start desktop directly",
		},
		"autoStartDesktop.launcher": {
			ZH: "先打开启动器",
			EN: "Open launcher first",
		},
		"harnessRestart.title": {
			ZH: "Harness 自动恢复",
			EN: "Harness auto-restart",
		},
		"harnessRestart.description": {
			ZH: "Harness 进程意外退出后自动重启，最多重试有限次数，重试之间按基础延迟退避等待。",
			EN: "Restart Harness automatically after an unexpected exit — up to a bounded number of attempts, backing off by the base delay between retries.",
		},
		"harnessRestart.enable": {
			ZH: "启用自动恢复",
			EN: "Enable auto-restart",
		},
		"harnessRestart.maxAttempts": {
			ZH: "最大重启次数",
			EN: "Max restart attempts",
		},
		"harnessRestart.baseDelay": {
			ZH: "基础重启延迟",
			EN: "Base restart delay",
		},
		"harnessRestart.delay": {
			ZH: "{count} 秒",
			EN: "{count}s",
		},
		"harnessRestart.loading": {
			ZH: "正在读取设置…",
			EN: "Loading settings…",
		},
		"harnessRestart.error": {
			ZH: "设置读取或保存失败：{message}",
			EN: "Failed to read or save settings: {message}",
		},
		"project.title": {
			ZH: "Project",
			EN: "Project",
		},
		"project.description": {
			ZH: "让一个主对话带领智能体团队，按需求清单持续完成一个目录里的开发。",
			EN: "Let one lead conversation guide an agent team through a running list of requirements in a directory.",
		},
		"project.helpLabel": {
			ZH: "Project 是什么",
			EN: "What is Project?",
		},
		"project.help": {
			ZH: "Project 把一个本地目录变成长期项目：它的主对话负责协调，你可以连续提需求，需求会排成清单，交给团队成员（子智能体）逐个完成；顶部的「需求」「团队」面板显示进度和交付结果。开启后，可在新会话首屏点「Project」，或在工作区目录菜单选「在此目录新建 Project」。关闭只是隐藏入口，已有 Project 的数据会保留。",
			EN: "Project turns a local directory into a long-running project: its lead conversation coordinates, you can keep adding requirements, and they queue up as a list that team members (subagents) complete one by one. The Requests and Team panels in the header show progress and deliveries. Once enabled, start one with Project on the New Session row, or New Project here in a Workspace row menu. Turning it off only hides the entries; existing Project data is kept.",
		},
		"project.restartHint": {
			ZH: "切换后会自动重启 Harness。",
			EN: "Toggling restarts the Harness automatically.",
		},
		"dshbot.title": {
			ZH: "机器人（Bots）",
			EN: "Bots",
		},
		"dshbot.beta": {
			ZH: "测试中",
			EN: "Beta",
		},
		"dshbot.description": {
			ZH: "在侧栏启用机器人联系人与群聊。该功能仍在测试，可能不稳定。",
			EN: "Enable bot contacts and group rooms in the sidebar. This feature is still in testing and may be unstable.",
		},
		"dshbot.restartHint": {
			ZH: "切换后会自动重启 Harness。",
			EN: "Toggling restarts the Harness automatically.",
		},
		"remoteWorkspace.title": {
			ZH: "远程工作区（SSH）",
			EN: "Remote workspaces (SSH)",
		},
		"remoteWorkspace.description": {
			ZH: "通过 SSH 把远程机器挂为工作区：在「添加工作区」选择远程目录，本地镜像与远端双向同步。",
			EN: "Mount SSH machines as workspaces: pick a remote directory under Add workspace, and a local mirror syncs both ways.",
		},
		"remoteWorkspace.restartHint": {
			ZH: "切换后会自动重启 Harness。",
			EN: "Toggling restarts the Harness automatically.",
		},
		"pet.nav": {
			ZH: "鲸鱼娘",
			EN: "Whale",
		},
		"pet.intro": {
			ZH: "你的常驻助理与桌面伙伴。桌面和 IM 共用她的同一条对话。",
			EN: "Your resident assistant and desktop companion. Desktop and IM share one conversation.",
		},
		"pet.tab.assistant": {
			ZH: "聊天与能力",
			EN: "Chat & capabilities",
		},
		"pet.tab.desktop": {
			ZH: "桌面形象与行为",
			EN: "Desktop appearance & behavior",
		},
		"pet.enabled": {
			ZH: "在桌面显示鲸鱼娘",
			EN: "Show Whale on desktop",
		},
		"pet.enabledDesc": {
			ZH: "关闭后隐藏桌面上的她，随时可以再开。",
			EN: "Hide the on-screen companion; you can bring her back anytime.",
		},
		"pet.group.look": {
			ZH: "外观",
			EN: "Appearance",
		},
		"pet.scale": {
			ZH: "体型",
			EN: "Size",
		},
		"pet.opacity": {
			ZH: "不透明度",
			EN: "Opacity",
		},
		"pet.scaleOpacity": {
			ZH: "体型与不透明度",
			EN: "Size & opacity",
		},
		"pet.personality": {
			ZH: "性格",
			EN: "Personality",
		},
		"pet.personalityDesc": {
			ZH: "影响她的台词与「聊聊」语气；助理人格跟随同一设置。",
			EN: "Shapes her lines and chat tone; the assistant persona follows the same setting.",
		},
		"pet.personality.natural": {
			ZH: "软萌",
			EN: "Soft",
		},
		"pet.personality.genki": {
			ZH: "元气",
			EN: "Lively",
		},
		"pet.personality.tsundere": {
			ZH: "傲娇",
			EN: "Tsundere",
		},
		"pet.personality.poison": {
			ZH: "毒舌",
			EN: "Sharp-tongued",
		},
		"pet.group.behavior": {
			ZH: "行为",
			EN: "Behavior",
		},
		"pet.activity": {
			ZH: "活跃频率",
			EN: "Activity",
		},
		"pet.activityDesc": {
			ZH: "驱动自言自语与乱逛的间隔。",
			EN: "Drives self-talk and wander intervals.",
		},
		"pet.activity.quiet": {
			ZH: "安静",
			EN: "Quiet",
		},
		"pet.activity.balanced": {
			ZH: "均衡",
			EN: "Balanced",
		},
		"pet.activity.active": {
			ZH: "活泼",
			EN: "Lively",
		},
		"pet.selfTalk": {
			ZH: "自言自语",
			EN: "Self-talk",
		},
		"pet.wander": {
			ZH: "自主乱逛",
			EN: "Wander",
		},
		"pet.autonomy": {
			ZH: "自主行为",
			EN: "Autonomy",
		},
		"pet.autonomyDesc": {
			ZH: "她自己开口说话、到处乱逛。",
			EN: "She talks to herself and wanders on her own.",
		},
		"pet.dragMode": {
			ZH: "拖拽方式",
			EN: "Dragging",
		},
		"pet.dragModeDesc": {
			ZH: "锁定后不能拖动她；Shift 模式需按住 Shift 拖。",
			EN: "Locked pins her in place; Shift mode requires holding Shift.",
		},
		"pet.dragFree": {
			ZH: "自由拖拽",
			EN: "Free drag",
		},
		"pet.dragShift": {
			ZH: "按住 Shift 拖拽",
			EN: "Hold Shift to drag",
		},
		"pet.dragLocked": {
			ZH: "锁定位置",
			EN: "Locked",
		},
		"pet.powerSave": {
			ZH: "省电模式",
			EN: "Power save",
		},
		"pet.powerSaveDesc": {
			ZH: "无交互约 5 分钟后降低帧率。",
			EN: "Lowers the frame rate after ~5 idle minutes.",
		},
		"pet.clickSound": {
			ZH: "点击音效",
			EN: "Click sound",
		},
		"pet.group.link": {
			ZH: "联动",
			EN: "Integrations",
		},
		"pet.chat": {
			ZH: "快捷对话",
			EN: "Quick chat",
		},
		"pet.chatDesc": {
			ZH: "右键面板的「聊聊」对话卡。",
			EN: "The 「chat」card on her panel.",
		},
		"pet.lookModel": {
			ZH: "看看模型",
			EN: "Look model",
		},
		"pet.lookModelDesc": {
			ZH: "「看看屏幕」用的视觉模型，从 DSH 已配置的供应商里挑（模型需声明支持图片输入）；请求走所选供应商自己的通道。关闭则面板不显示「看看」。",
			EN: "Vision route for 「看看」, picked from the configured DSH providers (models must declare image input); the request is dispatched through the selected provider itself. Off hides the look cell.",
		},
		"pet.lookModelOff": {
			ZH: "关闭",
			EN: "Off",
		},
		"pet.lookModelLoadFailed": {
			ZH: "模型目录加载失败：{message}",
			EN: "Could not load the model catalog: {message}",
		},
		"pet.group.assistant": {
			ZH: "助理",
			EN: "Assistant",
		},
		"pet.whaleAssistant": {
			ZH: "鲸鱼娘助理",
			EN: "Whale assistant",
		},
		"pet.whaleAssistantDesc": {
			ZH: "开启她的常驻对话与侧栏入口；切换后会重启 Harness。",
			EN: "Mounts her persistent conversation and sidebar entry; toggling restarts Harness.",
		},
		"pet.reset": {
			ZH: "恢复桌面形象与行为默认",
			EN: "Reset desktop appearance & behavior",
		},
		"pet.saveError": {
			ZH: "设置保存失败：{message}",
			EN: "Could not save pet settings: {message}",
		},
		"about.nav": {
			ZH: "关于",
			EN: "About",
		},
		"about.intro": {
			ZH: "本桌面应用与官方 Harness。",
			EN: "This desktop app and official Harness.",
		},
		"about.app": {
			ZH: "鲸屿 MyGo",
			EN: "Whale Isle MyGo",
		},
		"about.appMeta": {
			ZH: "Based on DeepSeek Harness",
			EN: "Based on DeepSeek Harness",
		},
		"about.version": {
			ZH: "桌面版本 {version}",
			EN: "Desktop version {version}",
		},
		"about.route": {
			ZH: "下载线路",
			EN: "Download route",
		},
		"about.routeGithub": {
			ZH: "GitHub · 国外线路",
			EN: "GitHub · international",
		},
		"about.routeCnb": {
			ZH: "CNB · 国内镜像",
			EN: "CNB · China mirror",
		},
		"about.checkUpdate": {
			ZH: "检测更新",
			EN: "Check for updates",
		},
		"about.installUpdate": {
			ZH: "在线安装",
			EN: "Install online",
		},
		"about.homePath": {
			ZH: "运行目录 {path}",
			EN: "Data folder {path}",
		},
		"about.openHome": {
			ZH: "打开运行目录",
			EN: "Open data folder",
		},
		"about.openHomeError": {
			ZH: "无法打开运行目录：{message}",
			EN: "Could not open the data folder: {message}",
		},
		"about.credEncrypted": {
			ZH: "凭据存储：系统钥匙串加密（safeStorage）",
			EN: "Credential storage: encrypted by the OS keychain (safeStorage)",
		},
		"about.credPlaintext": {
			ZH: "凭据存储：明文回退——本系统没有可用的钥匙串（常见于未解锁 keyring 的 Linux），API Key 等凭据以明文存放在应用数据目录。",
			EN: "Credential storage: plaintext fallback — no OS keychain is available (common on Linux without an unlocked keyring), so API keys and tokens sit unencrypted in the app data folder.",
		},
		"about.releases": {
			ZH: "发行版",
			EN: "Releases",
		},
		"about.desktopOnly": {
			ZH: "版本检测仅在桌面应用中可用。",
			EN: "Update checks are available in the desktop app.",
		},
		"about.updateChecking": {
			ZH: "正在检测更新…",
			EN: "Checking for updates…",
		},
		"about.updateNone": {
			ZH: "暂无发行版。",
			EN: "No release yet.",
		},
		"about.updateCurrent": {
			ZH: "已是最新版本 {latest}",
			EN: "You are on the latest version {latest}",
		},
		"about.updateAvailable": {
			ZH: "发现新版本 {latest}",
			EN: "New version {latest} is available",
		},
		"about.updateDownloading": {
			ZH: "正在下载 {percent}%",
			EN: "Downloading {percent}%",
		},
		"about.updateInstalling": {
			ZH: "已启动安装程序，请按向导完成安装。",
			EN: "Installer launched. Follow the wizard to finish.",
		},
		"about.updateOpenedPage": {
			ZH: "未找到安装包，已打开发行页。",
			EN: "No installer asset found. Opened the releases page.",
		},
		"about.updateManualInstall": {
			ZH: "已打开安装映像，请将 Whale Isle MyGo 拖入 Applications 完成更新。",
			EN: "Disk image opened. Drag Whale Isle MyGo into Applications to finish updating.",
		},
		"about.updateError": {
			ZH: "检测或安装失败：{message}",
			EN: "Update failed: {message}",
		},
		"about.updateCancelled": {
			ZH: "已取消更新",
			EN: "Update cancelled",
		},
		"about.harness": {
			ZH: "官方 Harness",
			EN: "Official Harness",
		},
		"update.badge": {
			ZH: "有新版本",
			EN: "Update available",
		},
		"update.tooltip": {
			ZH: "有新版本 {latest}，点击更新",
			EN: "Version {latest} is available — click to update",
		},
		"update.title": {
			ZH: "发现新版本 {latest}",
			EN: "New version {latest}",
		},
		"update.body": {
			ZH: "当前版本 {current}，可在线更新到 {latest}。下载完成后将启动安装程序。",
			EN: "You are on {current}. Update online to {latest}; the installer launches after the download.",
		},
		"update.install": {
			ZH: "立即更新",
			EN: "Update now",
		},
		"update.later": {
			ZH: "稍后再说",
			EN: "Later",
		},
		"update.retry": {
			ZH: "重试",
			EN: "Retry",
		},
		"connection.error": {
			ZH: "连接异常，刷新重试",
			EN: "Disconnected",
		},
		"connection.connecting": {
			ZH: "重新连接中",
			EN: "Reconnecting",
		},
		"connection.connected": {
			ZH: "连接成功",
			EN: "Connected",
		},
		"connection.reconnect": {
			ZH: "连接异常，点击立即重连",
			EN: "Disconnected, reconnect now",
		},
		"connection.restart": {
			ZH: "连接中断，正在重试，点击立即重连",
			EN: "Reconnecting, reconnect now",
		},
		"hostRelease.title": {
			ZH: "服务版本变化",
			EN: "Service version changed",
		},
		"hostRelease.description": {
			ZH: "服务版本已变化，本页面已暂停热更新，刷新后加载新版本。",
			EN: "The service version changed. This page paused live updates; reload to load the new version.",
		},
		"hostRelease.service": {
			ZH: "服务",
			EN: "Service",
		},
		"hostRelease.page": {
			ZH: "页面",
			EN: "Page",
		},
		"hostRelease.unknown": {
			ZH: "未解析",
			EN: "Unknown",
		},
		"hostRelease.reload": {
			ZH: "刷新以加载新版本",
			EN: "Reload to load the new version",
		},
		"hostRelease.later": {
			ZH: "暂不刷新",
			EN: "Not now",
		},
		"nav.group.agent": {
			ZH: "智能体",
			EN: "Agent",
		},
		"search.results": {
			ZH: "搜索结果",
			EN: "Search results",
		},
		"about.center.open": {
			ZH: "打开更新",
			EN: "Open updates",
		},
		"about.center.idle": {
			ZH: "当前版本 {current}",
			EN: "Current version {current}",
		},
		"about.center.current": {
			ZH: "已是最新版本（{current}）",
			EN: "Up to date ({current})",
		},
		"about.center.available": {
			ZH: "发现新版本 {version}",
			EN: "Version {version} is available",
		},
		"about.center.downloading": {
			ZH: "正在下载 {version} · {percent}%",
			EN: "Downloading {version} · {percent}%",
		},
		"about.center.paused": {
			ZH: "{version} 下载已暂停 · {percent}%",
			EN: "{version} download paused · {percent}%",
		},
		"about.center.verifying": {
			ZH: "正在校验 {version}",
			EN: "Verifying {version}",
		},
		"about.center.ready": {
			ZH: "{version} 已准备好，可以安装",
			EN: "{version} is ready to install",
		},
		"about.center.installing": {
			ZH: "正在安装 {version}",
			EN: "Installing {version}",
		},
		"about.center.error": {
			ZH: "更新未完成：{message}",
			EN: "Update not finished: {message}",
		},
	},
	"dsh-client-ui-settings-market": {
		"close": {
			ZH: "关闭",
			EN: "Close",
		},
		"buildApprovalMissing": {
			ZH: "缺少有效的 allowBuilds 授权项",
			EN: "No valid allowBuilds approval keys",
		},
		"publishedVersion": {
			ZH: "已发布版本",
			EN: "Published version",
		},
		"declaredRequirements": {
			ZH: "作者声明的依赖要求",
			EN: "Author-declared requirements",
		},
		"detailsFailed": {
			ZH: "部分详情获取失败；可重试或访问项目主页。",
			EN: "Some details could not be loaded. Retry or visit the project homepage.",
		},
		"detailsLoading": {
			ZH: "正在获取项目文档…",
			EN: "Loading project documentation…",
		},
		"repositoryReadme": {
			ZH: "仓库 README",
			EN: "Repository README",
		},
		"copy": {
			ZH: "复制",
			EN: "Copy",
		},
		"copied": {
			ZH: "已复制",
			EN: "Copied",
		},
		"footnotes": {
			ZH: "脚注",
			EN: "Footnotes",
		},
		"tabFavorites": {
			ZH: "收藏",
			EN: "Favorites",
		},
		"tabHistory": {
			ZH: "操作记录",
			EN: "Operations",
		},
		"favorite": {
			ZH: "收藏插件",
			EN: "Favorite plugin",
		},
		"unfavorite": {
			ZH: "取消收藏",
			EN: "Remove favorite",
		},
		"details": {
			ZH: "插件详情",
			EN: "Plugin details",
		},
		"author": {
			ZH: "作者",
			EN: "Author",
		},
		"source": {
			ZH: "安装来源",
			EN: "Install source",
		},
		"added": {
			ZH: "收录时间",
			EN: "Date added",
		},
		"unavailable": {
			ZH: "不可安装",
			EN: "Not installable",
		},
		"unknown": {
			ZH: "未知",
			EN: "Unknown",
		},
		"compatibility": {
			ZH: "宿主兼容性",
			EN: "Host compatibility",
		},
		"compatibilityUnknown": {
			ZH: "未验证；作者的版本声明不等于运行兼容性保证",
			EN: "Unverified; declared requirements do not guarantee runtime compatibility",
		},
		"screenshot": {
			ZH: "插件截图 {number}",
			EN: "Plugin screenshot {number}",
		},
		"imageFailed": {
			ZH: "截图加载失败",
			EN: "Screenshot could not be loaded",
		},
		"imageCount": {
			ZH: "{current} / {total}",
			EN: "{current} / {total}",
		},
		"previousImage": {
			ZH: "上一张",
			EN: "Previous image",
		},
		"nextImage": {
			ZH: "下一张",
			EN: "Next image",
		},
		"openImage": {
			ZH: "查看原图",
			EN: "Open original image",
		},
		"noScreenshots": {
			ZH: "暂无截图",
			EN: "No screenshots",
		},
		"readmeHomepage": {
			ZH: "README / 项目主页",
			EN: "README / project homepage",
		},
		"sort": {
			ZH: "排序",
			EN: "Sort",
		},
		"sortCatalog": {
			ZH: "目录顺序",
			EN: "Catalog order",
		},
		"sortStars": {
			ZH: "最多星标",
			EN: "Most stars",
		},
		"sortNewest": {
			ZH: "最新收录",
			EN: "Recently added",
		},
		"period": {
			ZH: "收录时间范围",
			EN: "Date added range",
		},
		"periodAll": {
			ZH: "全部时间",
			EN: "Any time",
		},
		"periodWeek": {
			ZH: "最近 7 天",
			EN: "Last 7 days",
		},
		"periodMonth": {
			ZH: "最近 30 天",
			EN: "Last 30 days",
		},
		"installedFilter": {
			ZH: "已安装筛选",
			EN: "Installed filter",
		},
		"filterAll": {
			ZH: "全部插件",
			EN: "All plugins",
		},
		"filterUpdates": {
			ZH: "可更新",
			EN: "Updates available",
		},
		"filterFailed": {
			ZH: "检查失败",
			EN: "Check failed",
		},
		"updateAll": {
			ZH: "更新全部",
			EN: "Update all",
		},
		"confirmInstall": {
			ZH: "确认安装插件",
			EN: "Confirm plugin installation",
		},
		"confirmUninstall": {
			ZH: "确认卸载插件",
			EN: "Confirm plugin removal",
		},
		"confirmBatch": {
			ZH: "确认批量更新",
			EN: "Confirm batch update",
		},
		"confirmAction": {
			ZH: "确认执行",
			EN: "Confirm",
		},
		"installWarning": {
			ZH: "第三方插件将在本机执行代码。确认来源可信后安装；完成后会重启 Harness，请先结束正在进行的任务。",
			EN: "Third-party plugins execute code on this computer. Verify the source before installing. Harness will restart; finish any running tasks first.",
		},
		"uninstallWarning": {
			ZH: "将从桌面 web profile 移除此插件并重启 Harness。请先结束正在进行的任务；插件自行保存的数据不会在此删除。",
			EN: "This removes the plugin from the desktop web profile and restarts Harness. Finish running tasks first. Data saved by the plugin is not deleted here.",
		},
		"batchWarning": {
			ZH: "以下插件将依次更新，结束后统一重启 Harness。请先结束正在进行的任务；需要构建授权的项保留失败结果，稍后逐项确认。",
			EN: "These plugins will update sequentially, then Harness will restart once. Finish running tasks first. Items requiring build approval remain failed for individual approval later.",
		},
		"stateFailed": {
			ZH: "无法读取或保存市场收藏与操作记录。",
			EN: "Marketplace favorites or operation history could not be read or saved.",
		},
		"installedReadFailed": {
			ZH: "无法读取已安装插件；当前列表可能不是最新状态。",
			EN: "Installed plugins could not be read; the current list may be out of date.",
		},
		"opOutcomeUnknown": {
			ZH: "插件操作已返回，但 Harness 重启结果尚未确认。请核对运行状态与操作记录，不要重复安装或卸载。",
			EN: "The plugin operation returned, but the Harness restart result is unconfirmed. Check the runtime and operation history before repeating the operation.",
		},
		"exportLog": {
			ZH: "导出脱敏日志",
			EN: "Export redacted logs",
		},
		"historyEmpty": {
			ZH: "暂无操作记录",
			EN: "No operations yet",
		},
		"noLog": {
			ZH: "没有日志输出",
			EN: "No log output",
		},
		"operationRunning": {
			ZH: "进行中",
			EN: "Running",
		},
		"operationSucceeded": {
			ZH: "已完成",
			EN: "Completed",
		},
		"operationFailed": {
			ZH: "失败",
			EN: "Failed",
		},
		"operationInterrupted": {
			ZH: "已中断，请核对插件状态",
			EN: "Interrupted; verify plugin state",
		},
		"nav": {
			ZH: "市场",
			EN: "Marketplace",
		},
		"heading": {
			ZH: "插件市场",
			EN: "Plugin marketplace",
		},
		"intro": {
			ZH: "从精选目录浏览、安装并更新社区插件；变更后 Harness 会自动重启。",
			EN: "Browse, install, and update community plugins from the curated catalog; Harness restarts after changes.",
		},
		"tabDiscover": {
			ZH: "发现",
			EN: "Discover",
		},
		"tabInstalled": {
			ZH: "已安装",
			EN: "Installed",
		},
		"search": {
			ZH: "搜索插件",
			EN: "Search plugins",
		},
		"categories": {
			ZH: "分类",
			EN: "Categories",
		},
		"expandCategories": {
			ZH: "全部分类",
			EN: "All categories",
		},
		"collapseCategories": {
			ZH: "收起分类",
			EN: "Fewer categories",
		},
		"refresh": {
			ZH: "刷新",
			EN: "Refresh",
		},
		"refreshing": {
			ZH: "刷新中…",
			EN: "Refreshing…",
		},
		"loading": {
			ZH: "正在加载插件目录…",
			EN: "Loading the plugin catalog…",
		},
		"loadError": {
			ZH: "无法加载插件目录。",
			EN: "The plugin catalog is unavailable.",
		},
		"retry": {
			ZH: "重试",
			EN: "Retry",
		},
		"empty": {
			ZH: "目录里还没有插件。",
			EN: "No plugins in the catalog yet.",
		},
		"emptySearch": {
			ZH: "没有匹配的插件。",
			EN: "No plugins match your search.",
		},
		"count": {
			ZH: "{count} 个插件",
			EN: "{count} plugins",
		},
		"showMore": {
			ZH: "加载更多（剩余 {count}）",
			EN: "Show more ({count} left)",
		},
		"stars": {
			ZH: "{count} 星",
			EN: "{count} stars",
		},
		"installed": {
			ZH: "已安装",
			EN: "Installed",
		},
		"installedEmpty": {
			ZH: "还没有装过社区插件，去「发现」页逛逛吧。",
			EN: "No community plugins yet — browse the Discover tab.",
		},
		"checkingUpdates": {
			ZH: "正在检查更新…",
			EN: "Checking for updates…",
		},
		"updatesAvailable": {
			ZH: "{count} 个插件可更新",
			EN: "{count} plugins can be updated",
		},
		"updatesCurrent": {
			ZH: "未发现可用更新",
			EN: "No updates found",
		},
		"updateCheckFailed": {
			ZH: "无法检查插件更新。",
			EN: "Plugin updates could not be checked.",
		},
		"updateFromTo": {
			ZH: "{current} → {latest}",
			EN: "{current} → {latest}",
		},
		"currentVersion": {
			ZH: "当前 {current}",
			EN: "Current {current}",
		},
		"ungrouped": {
			ZH: "未分组",
			EN: "Ungrouped",
		},
		"dropped": {
			ZH: "已退役",
			EN: "Retired",
		},
		"deprecated": {
			ZH: "已弃用",
			EN: "Deprecated",
		},
		"install": {
			ZH: "安装",
			EN: "Install",
		},
		"installing": {
			ZH: "安装中…",
			EN: "Installing…",
		},
		"update": {
			ZH: "更新",
			EN: "Update",
		},
		"updating": {
			ZH: "更新中…",
			EN: "Updating…",
		},
		"uninstall": {
			ZH: "卸载",
			EN: "Uninstall",
		},
		"uninstalling": {
			ZH: "卸载中…",
			EN: "Uninstalling…",
		},
		"homepage": {
			ZH: "主页",
			EN: "Homepage",
		},
		"progressHeading": {
			ZH: "插件操作进度",
			EN: "Plugin operation progress",
		},
		"restarting": {
			ZH: "正在重启 Harness…",
			EN: "Restarting Harness…",
		},
		"installDone": {
			ZH: "安装完成，Harness 已重启。",
			EN: "Installed; Harness restarted.",
		},
		"updateDone": {
			ZH: "更新完成，Harness 已重启。",
			EN: "Updated; Harness restarted.",
		},
		"uninstallDone": {
			ZH: "卸载完成，Harness 已重启。",
			EN: "Uninstalled; Harness restarted.",
		},
		"opFailed": {
			ZH: "操作失败：{message}",
			EN: "Operation failed: {message}",
		},
		"allowBuildsAsk": {
			ZH: "「{name}」需要在本机执行构建脚本才能完成操作。",
			EN: "\"{name}\" needs to run build scripts on this machine to finish the operation.",
		},
		"allowBuildsKeys": {
			ZH: "将允许：{keys}",
			EN: "Will allow: {keys}",
		},
		"allowBuildsConfirm": {
			ZH: "允许并重试",
			EN: "Allow and retry",
		},
		"allowBuildsCancel": {
			ZH: "取消",
			EN: "Cancel",
		},
		"harnessDown": {
			ZH: "插件已写入，但 Harness 没有起来。请从现有入口重启，不要再安装一次。",
			EN: "The plugin was written, but Harness did not come back. Restart from an existing entry; do not install again.",
		},
	},
	"dsh-client-ui-settings-mcp": {
		"nav": {
			ZH: "MCP",
			EN: "MCP",
		},
		"title": {
			ZH: "MCP 服务器",
			EN: "MCP servers",
		},
		"intro": {
			ZH: "管理本机 MCP 服务器。受管条目写入 $DSH_HOME/mcp-servers.yaml；HTTP 服务器可在本页登录后在对话里使用工具。组成配置条目只读。",
			EN: "Manage local MCP servers. Managed rows write $DSH_HOME/mcp-servers.yaml. Sign in on an HTTP row so its tools are available in chat. Composition rows are read-only.",
		},
		"add": {
			ZH: "添加服务器",
			EN: "Add server",
		},
		"refresh": {
			ZH: "刷新",
			EN: "Refresh",
		},
		"empty": {
			ZH: "还没有 MCP 服务器。添加一个服务器以连接本地工具或远程服务。",
			EN: "No MCP servers yet. Add one to connect local tools or a remote service.",
		},
		"emptySearch": {
			ZH: "没有符合当前搜索或筛选条件的服务器。",
			EN: "No servers match the current search and filters.",
		},
		"clearFilters": {
			ZH: "清除搜索和筛选",
			EN: "Clear search and filters",
		},
		"loading": {
			ZH: "正在读取 MCP 服务器…",
			EN: "Reading MCP servers…",
		},
		"error": {
			ZH: "暂时无法读取 MCP 服务器。",
			EN: "MCP servers are temporarily unavailable.",
		},
		"retry": {
			ZH: "重试",
			EN: "Retry",
		},
		"refreshFailed": {
			ZH: "服务器列表刷新失败；当前显示的是上次读取的状态。",
			EN: "The server list could not be refreshed; the last loaded state is still shown.",
		},
		"search": {
			ZH: "搜索名称、ID、命令或 URL",
			EN: "Search name, ID, command, or URL",
		},
		"filterEnabled": {
			ZH: "启用状态",
			EN: "Enabled state",
		},
		"filterAll": {
			ZH: "全部",
			EN: "All",
		},
		"filterEnabledOnly": {
			ZH: "已启用",
			EN: "Enabled",
		},
		"filterDisabledOnly": {
			ZH: "已停用",
			EN: "Disabled",
		},
		"configuredCount": {
			ZH: "已配置 {count} 项",
			EN: "{count} configured",
		},
		"compositionSection": {
			ZH: "组合内 MCP",
			EN: "Composition MCP",
		},
		"compositionNote": {
			ZH: "由组合注册，只读。",
			EN: "Registered by the composition, read-only.",
		},
		"edit": {
			ZH: "编辑",
			EN: "Edit",
		},
		"remove": {
			ZH: "删除",
			EN: "Delete",
		},
		"editFor": {
			ZH: "编辑 {name}",
			EN: "Edit {name}",
		},
		"removeFor": {
			ZH: "删除 {name}",
			EN: "Delete {name}",
		},
		"enableAria": {
			ZH: "启用 {name}",
			EN: "Enable {name}",
		},
		"enabled": {
			ZH: "已启用",
			EN: "Enabled",
		},
		"disabled": {
			ZH: "已停用",
			EN: "Disabled",
		},
		"managed": {
			ZH: "已配置",
			EN: "Configured",
		},
		"composition": {
			ZH: "组成配置",
			EN: "Composition",
		},
		"readOnly": {
			ZH: "只读",
			EN: "Read-only",
		},
		"stdio": {
			ZH: "stdio",
			EN: "stdio",
		},
		"http": {
			ZH: "HTTP",
			EN: "HTTP",
		},
		"unobserved": {
			ZH: "未挂载",
			EN: "Not mounted",
		},
		"pending": {
			ZH: "等待依赖",
			EN: "Waiting for dependencies",
		},
		"loadingPhase": {
			ZH: "加载中",
			EN: "Loading",
		},
		"active": {
			ZH: "已连接",
			EN: "Connected",
		},
		"failed": {
			ZH: "失败",
			EN: "Failed",
		},
		"unloading": {
			ZH: "卸载中",
			EN: "Unloading",
		},
		"healthConnecting": {
			ZH: "连接中",
			EN: "Connecting",
		},
		"healthConnected": {
			ZH: "已连接",
			EN: "Connected",
		},
		"healthReconnecting": {
			ZH: "重连中",
			EN: "Reconnecting",
		},
		"healthFailed": {
			ZH: "连接失败",
			EN: "Connection failed",
		},
		"toolCount": {
			ZH: "{count} 个工具",
			EN: "{count} tools",
		},
		"toolsTitle": {
			ZH: "{name} 可用工具",
			EN: "{name} tools",
		},
		"test": {
			ZH: "测试",
			EN: "Test",
		},
		"testing": {
			ZH: "测试中…",
			EN: "Testing…",
		},
		"testFor": {
			ZH: "测试 {name}",
			EN: "Test {name}",
		},
		"testSuccess": {
			ZH: "测试成功：{count} 个工具（{tools}）",
			EN: "Test passed: {count} tools ({tools})",
		},
		"testNoTools": {
			ZH: "没有工具",
			EN: "no tools",
		},
		"testMoreTools": {
			ZH: "{names}，另有 {count} 个",
			EN: "{names}, {count} more",
		},
		"testFailed": {
			ZH: "测试失败",
			EN: "Test failed",
		},
		"signIn": {
			ZH: "登录",
			EN: "Sign in",
		},
		"signingIn": {
			ZH: "正在登录…",
			EN: "Signing in…",
		},
		"signInFor": {
			ZH: "登录 {name}",
			EN: "Sign in to {name}",
		},
		"signInFailed": {
			ZH: "无法完成登录",
			EN: "Could not complete sign-in",
		},
		"toggleFailed": {
			ZH: "无法更新启用状态",
			EN: "Could not update enabled state",
		},
		"editorTitleAdd": {
			ZH: "添加 MCP 服务器",
			EN: "Add MCP server",
		},
		"editorTitleEdit": {
			ZH: "编辑 MCP 服务器",
			EN: "Edit MCP server",
		},
		"editorDescription": {
			ZH: "设置服务器标识与连接方式。保存后，受管配置会立即更新。",
			EN: "Set the server identity and connection. Saving immediately updates the managed configuration.",
		},
		"editorMode": {
			ZH: "编辑方式",
			EN: "Editor mode",
		},
		"editorModeForm": {
			ZH: "表单",
			EN: "Form",
		},
		"editorModeJson": {
			ZH: "JSON",
			EN: "JSON",
		},
		"jsonLabel": {
			ZH: "服务器 JSON",
			EN: "Server JSON",
		},
		"jsonInvalid": {
			ZH: "JSON 无法解析为 MCP 服务器。",
			EN: "JSON could not be parsed as an MCP server.",
		},
		"jsonEmpty": {
			ZH: "请提供一个服务器对象。",
			EN: "Provide a server object.",
		},
		"generalGroup": {
			ZH: "基本信息",
			EN: "General",
		},
		"connectionGroup": {
			ZH: "连接配置",
			EN: "Connection",
		},
		"id": {
			ZH: "ID",
			EN: "ID",
		},
		"idHint": {
			ZH: "1–32 个字母、数字、下划线或连字符。创建后不可修改。",
			EN: "1–32 letters, numbers, underscores, or hyphens. It cannot be changed after creation.",
		},
		"serverName": {
			ZH: "名称",
			EN: "Name",
		},
		"serverNameHint": {
			ZH: "用于生成 mcp__<serverName>__… 工具名称。",
			EN: "Used to create mcp__<serverName>__… tool names.",
		},
		"transport": {
			ZH: "类型",
			EN: "Type",
		},
		"timeout": {
			ZH: "超时（毫秒）",
			EN: "Timeout (ms)",
		},
		"timeoutHint": {
			ZH: "可选。工具调用超时，留空则使用默认值。",
			EN: "Optional. Tool-call timeout; leave empty for the default.",
		},
		"timeoutInvalid": {
			ZH: "超时必须是正整数。",
			EN: "Timeout must be a positive integer.",
		},
		"command": {
			ZH: "命令",
			EN: "Command",
		},
		"args": {
			ZH: "参数（空格分隔）",
			EN: "Arguments (space-separated)",
		},
		"env": {
			ZH: "环境变量（KEY=value）",
			EN: "Environment (KEY=value)",
		},
		"envToggle": {
			ZH: "环境变量",
			EN: "Environment variables",
		},
		"cwd": {
			ZH: "工作目录",
			EN: "Working directory",
		},
		"url": {
			ZH: "URL",
			EN: "URL",
		},
		"headers": {
			ZH: "请求头（KEY=value）",
			EN: "Headers (KEY=value)",
		},
		"stdioRiskTitle": {
			ZH: "仅运行你信任的命令",
			EN: "Only run commands you trust",
		},
		"stdioRiskBody": {
			ZH: "stdio 服务器命令在智能体沙箱之外以当前用户权限运行，可读取或修改本机文件。",
			EN: "stdio server commands run outside the agent sandbox with your user permissions and can read or modify local files.",
		},
		"save": {
			ZH: "保存",
			EN: "Save",
		},
		"saving": {
			ZH: "正在保存…",
			EN: "Saving…",
		},
		"cancel": {
			ZH: "取消",
			EN: "Cancel",
		},
		"close": {
			ZH: "关闭",
			EN: "Close",
		},
		"deleteTitle": {
			ZH: "删除 {name}？",
			EN: "Delete {name}?",
		},
		"deleteBody": {
			ZH: "将从受管文档中移除 {name}，并卸载对应的 mcp-client。",
			EN: "This removes {name} from the managed document and unmounts its mcp-client.",
		},
		"deleteConfirm": {
			ZH: "确认删除",
			EN: "Confirm delete",
		},
		"deleting": {
			ZH: "正在删除…",
			EN: "Deleting…",
		},
		"saveFailed": {
			ZH: "保存失败",
			EN: "Save failed",
		},
		"deleteFailed": {
			ZH: "删除失败",
			EN: "Delete failed",
		},
		"idInvalid": {
			ZH: "ID 必须匹配 [A-Za-z0-9_-]{1,32}。",
			EN: "ID must match [A-Za-z0-9_-]{1,32}.",
		},
		"serverNameInvalid": {
			ZH: "名称必须匹配 [A-Za-z0-9_-]{1,32}。",
			EN: "Name must match [A-Za-z0-9_-]{1,32}.",
		},
		"commandRequired": {
			ZH: "stdio 服务器需要命令。",
			EN: "A stdio server needs a command.",
		},
		"urlRequired": {
			ZH: "HTTP 服务器需要 URL。",
			EN: "An HTTP server needs a URL.",
		},
		"urlInvalid": {
			ZH: "请输入有效的 HTTP 或 HTTPS URL。",
			EN: "Enter a valid HTTP or HTTPS URL.",
		},
		"envLineInvalid": {
			ZH: "环境变量第 {line} 行必须是有效的 KEY=value。",
			EN: "Environment line {line} must be a valid KEY=value pair.",
		},
		"headerLineInvalid": {
			ZH: "请求头第 {line} 行必须是有效的 KEY=value。",
			EN: "Header line {line} must be a valid KEY=value pair.",
		},
	},
	"dsh-client-ui-settings-models": {
		"nav": {
			ZH: "模型",
			EN: "Models",
		},
		"deepSeekAccount": {
			ZH: "DeepSeek 账号",
			EN: "DeepSeek Account",
		},
		"title": {
			ZH: "模型",
			EN: "Models",
		},
		"intro": {
			ZH: "填入各提供商的 API 密钥即可使用其模型。",
			EN: "Enter your API keys to use models from the following providers.",
		},
		"edit": {
			ZH: "编辑",
			EN: "Edit",
		},
		"editProvider": {
			ZH: "编辑 {provider}",
			EN: "Edit {provider}",
		},
		"remove": {
			ZH: "删除",
			EN: "Delete",
		},
		"removeProvider": {
			ZH: "删除 {provider}",
			EN: "Delete {provider}",
		},
		"deleteTitle": {
			ZH: "删除 {provider}？",
			EN: "Delete {provider}?",
		},
		"deleteDescription": {
			ZH: "删除 {provider} 会移除其配置；其使用的凭证（如有）由其他位置管理，将会保留。",
			EN: "Deleting {provider} removes its configuration. Any credential it uses is managed elsewhere and will be kept.",
		},
		"deleteDescriptionWithCredential": {
			ZH: "删除 {provider} 会移除其配置和存储的 API 密钥。",
			EN: "Deleting {provider} removes its configuration and stored API key.",
		},
		"deleteConfirm": {
			ZH: "删除 {provider}",
			EN: "Delete {provider}",
		},
		"deleting": {
			ZH: "正在删除 {provider}…",
			EN: "Deleting {provider}…",
		},
		"add": {
			ZH: "添加模型提供商",
			EN: "Add model provider",
		},
		"addMode": {
			ZH: "添加方式",
			EN: "How to add",
		},
		"addCatalog": {
			ZH: "第三方模型提供商",
			EN: "Third-party model provider",
		},
		"addCustom": {
			ZH: "自定义模型 API",
			EN: "Custom model API",
		},
		"addCatalogHint": {
			ZH: "从内置目录中选择 OpenAI、Anthropic、Kimi 等提供商，填入其 API 密钥即可使用。",
			EN: "Pick OpenAI, Anthropic, Kimi, or another provider from the built-in catalog and enter its API key.",
		},
		"addCustomHint": {
			ZH: "连接中转站、自部署服务或其他兼容 OpenAI / Anthropic 协议的接口，需填写 API 地址、协议和模型。",
			EN: "Connect a relay, a self-hosted server, or any other OpenAI- or Anthropic-compatible endpoint by its base URL, protocol, and models.",
		},
		"addCatalogExhausted": {
			ZH: "目录中的提供商都已添加。",
			EN: "Every catalog provider is already configured.",
		},
		"addCustomUnavailable": {
			ZH: "没有可用的 API 协议。",
			EN: "No API protocol is available to declare.",
		},
		"provider": {
			ZH: "提供商",
			EN: "Provider",
		},
		"close": {
			ZH: "关闭",
			EN: "Close",
		},
		"cancel": {
			ZH: "取消",
			EN: "Cancel",
		},
		"apply": {
			ZH: "保存",
			EN: "Apply",
		},
		"applying": {
			ZH: "保存中…",
			EN: "Applying…",
		},
		"savedProvider": {
			ZH: "已保存 {provider}。",
			EN: "Saved {provider}.",
		},
		"credentialConfigured": {
			ZH: "API 密钥已配置",
			EN: "API key configured",
		},
		"credentialMissing": {
			ZH: "API 密钥缺失",
			EN: "API key missing",
		},
		"accountSignedIn": {
			ZH: "已登录",
			EN: "Signed in",
		},
		"readOnly": {
			ZH: "当前部署的设置文档为只读。",
			EN: "The settings document is read-only in this deployment.",
		},
		"loadFailed": {
			ZH: "加载提供商目录失败",
			EN: "Loading the provider directory failed",
		},
		"conflict": {
			ZH: "这张卡片打开期间，这些设置已被其他地方改动。请关闭后重新打开，在当前值上编辑。",
			EN: "Someone else changed these settings while this card was open. Close it and reopen to edit the current values.",
		},
		"retry": {
			ZH: "重试",
			EN: "Retry",
		},
		"keyInput": {
			ZH: "API 密钥",
			EN: "API key",
		},
		"keyReveal": {
			ZH: "显示密钥",
			EN: "Show secret",
		},
		"keyHide": {
			ZH: "隐藏密钥",
			EN: "Hide secret",
		},
		"keyPlaceholder": {
			ZH: "输入 API 密钥",
			EN: "Enter your API key",
		},
		"keyPlaceholderFallback": {
			ZH: "留空使用环境认证（{hint}）",
			EN: "Leave blank to use env as auth ({hint})",
		},
		"keyPlaceholderSaved": {
			ZH: "留空使用已保存凭据",
			EN: "Leave blank to use saved credentials",
		},
		"keyFallbackMissing": {
			ZH: "未检测到环境变量。请填写 API 密钥，或设置环境变量 {hint}。",
			EN: "Env var not detected. Enter an API key, or configure {hint} in your env.",
		},
		"keyFallbackDetected": {
			ZH: "已检测到环境变量（{hint}）",
			EN: "Env var detected. ({hint})",
		},
		"keyFallbackDetectedHidden": {
			ZH: "已检测到环境变量",
			EN: "Env var detected.",
		},
		"keyFallbackUnchecked": {
			ZH: "环境凭据状态尚未确认",
			EN: "Environment credential status has not been confirmed.",
		},
		"keyHelp": {
			ZH: "优先使用已保存的凭据。此处显示的环境后备按以下顺序排列：{env}。来源依次为启动环境、$cwd/.env、$DSH_HOME/.env。",
			EN: "Saved credentials are used first. Environment fallbacks shown here, in order: {env}. Sources are launch environment, $cwd/.env, then $DSH_HOME/.env.",
		},
		"authHelp": {
			ZH: "认证说明",
			EN: "Authentication details",
		},
		"customized": {
			ZH: "自定义设置",
			EN: "Customized settings",
		},
		"baseUrl": {
			ZH: "API 地址",
			EN: "Base URL",
		},
		"baseUrlDefault": {
			ZH: "没有默认地址",
			EN: "No default endpoint",
		},
		"baseUrlEnvironment": {
			ZH: "提供方地址配置：{env}",
			EN: "Provider endpoint configuration: {env}",
		},
		"baseUrlConfigured": {
			ZH: "由提供方配置地址",
			EN: "Provider-configured endpoint",
		},
		"baseUrlPlaceholder": {
			ZH: "{url}",
			EN: "{url}",
		},
		"deepSeekBaseUrl": {
			ZH: "https://api.deepseek.com/anthropic（默认使用环境 DEEPSEEK_BASE_URL）",
			EN: "https://api.deepseek.com/anthropic (Default to use DEEPSEEK_BASE_URL)",
		},
		"deepSeekEndpointHint": {
			ZH: "请填写兼容 Anthropic Messages 协议的 API 地址。",
			EN: "Use an API endpoint compatible with Anthropic Messages.",
		},
		"models": {
			ZH: "模型目录",
			EN: "Models",
		},
		"modelsInherited": {
			ZH: "正在使用适配器默认模型",
			EN: "Using the adapter defaults",
		},
		"modelsCustomized": {
			ZH: "已自定义模型目录",
			EN: "Customized model catalog",
		},
		"resetModels": {
			ZH: "恢复默认模型",
			EN: "Restore defaults",
		},
		"model": {
			ZH: "模型",
			EN: "Model",
		},
		"modelId": {
			ZH: "模型 ID",
			EN: "Model ID",
		},
		"modelName": {
			ZH: "显示名称",
			EN: "Display name",
		},
		"modelNamePlaceholder": {
			ZH: "留空时使用模型 ID",
			EN: "Uses the model ID when empty",
		},
		"contextWindow": {
			ZH: "上下文窗口",
			EN: "Context window",
		},
		"contextWindowPlaceholder": {
			ZH: "使用提供商默认值",
			EN: "Uses the provider default",
		},
		"maxTokens": {
			ZH: "最大输出 token 数",
			EN: "Max output tokens",
		},
		"maxTokensPlaceholder": {
			ZH: "使用提供商默认值",
			EN: "Uses the provider default",
		},
		"modelAdvanced": {
			ZH: "模型选项",
			EN: "Model options",
		},
		"effortTitle": {
			ZH: "Supported thinking intensity",
			EN: "Supported thinking intensity",
		},
		"effort.off": {
			ZH: "Off",
			EN: "Off",
		},
		"effort.minimal": {
			ZH: "Minimal",
			EN: "Minimal",
		},
		"effort.low": {
			ZH: "Low",
			EN: "Low",
		},
		"effort.medium": {
			ZH: "Medium",
			EN: "Medium",
		},
		"effort.high": {
			ZH: "High",
			EN: "High",
		},
		"effort.xhigh": {
			ZH: "Extra High",
			EN: "Extra High",
		},
		"effort.max": {
			ZH: "Max",
			EN: "Max",
		},
		"effortOffAlone": {
			ZH: "Off cannot be the only thinking intensity.",
			EN: "Off cannot be the only thinking intensity.",
		},
		"modelInputTypes": {
			ZH: "输入类型",
			EN: "Input types",
		},
		"modelInputText": {
			ZH: "文本",
			EN: "Text",
		},
		"modelInputImage": {
			ZH: "图片",
			EN: "Image",
		},
		"modelReasoning": {
			ZH: "推理等级",
			EN: "Reasoning levels",
		},
		"modelReasoningLevel": {
			ZH: "档位",
			EN: "Level",
		},
		"modelReasoningValue": {
			ZH: "API 参数值",
			EN: "API value",
		},
		"modelReasoningHelp": {
			ZH: "推理等级说明",
			EN: "About reasoning levels",
		},
		"modelReasoningHelpText": {
			ZH: "API 参数值会写入请求的推理强度字段（如 OpenAI 的 reasoning_effort），默认与档位名相同；如有不同，以模型提供商的 API 文档为准。Off 表示关闭思考，不需要填写参数值；配置中已写入的值会保留并发送。",
			EN: "The API value is sent in the request's reasoning-effort field, such as OpenAI's reasoning_effort. It defaults to the level name; where the model provider's API documentation specifies a different value, follow it. Off turns reasoning off and needs no value; a value already written in the configuration is kept and sent.",
		},
		"modelReasoningNeedsLevel": {
			ZH: "至少添加一个 Off 以外的档位；不需要思考请关闭推理等级。",
			EN: "Add at least one level besides Off, or turn reasoning levels off.",
		},
		"modelReasoningValueRequired": {
			ZH: "Off 以外的档位都需要填写 API 参数值，且参数值不能为空字符串。",
			EN: "Every level besides Off needs an API value, and no value can be empty.",
		},
		"modelReasoningAdd": {
			ZH: "添加档位",
			EN: "Add level",
		},
		"modelReasoningRemove": {
			ZH: "删除档位",
			EN: "Delete level",
		},
		"modelReasoningOffValuePlaceholder": {
			ZH: "关闭思考",
			EN: "Turns reasoning off",
		},
		"addModel": {
			ZH: "添加模型",
			EN: "Add model",
		},
		"removeModel": {
			ZH: "删除模型",
			EN: "Delete model",
		},
		"modelsEmpty": {
			ZH: "模型选择器中将不显示任何模型；目录外 ID 仍可直接发送。",
			EN: "No models will be shown in the selector. Unlisted IDs can still be sent directly.",
		},
		"keyBlank": {
			ZH: "请输入 API 密钥；留空则保持已存储的密钥。",
			EN: "Enter the API key, or leave the field empty to keep the stored one.",
		},
		"keyBlankNew": {
			ZH: "请输入 API 密钥；若该提供商以其他方式鉴权，可以留空。",
			EN: "Enter the API key, or leave the field empty if this provider authenticates another way.",
		},
		"keyIllegalCharacters": {
			ZH: "该 API 密钥格式错误，请检查。",
			EN: "This API key is not in a valid format. Please check it.",
		},
		"modelIdRequired": {
			ZH: "模型 ID 不能为空。",
			EN: "Model ID is required.",
		},
		"modelIdDuplicate": {
			ZH: "模型 ID 不能重复。",
			EN: "Model ID must be unique.",
		},
		"modelNameInvalid": {
			ZH: "显示名称不能为空。",
			EN: "Display name cannot be empty.",
		},
		"modelContextInvalid": {
			ZH: "上下文窗口必须是正数，例如 131072、256K 或 1M。",
			EN: "Context window must be a positive count, like 131072, 256K, or 1M.",
		},
		"modelMaxTokensInvalid": {
			ZH: "最大输出 token 数必须是正数，例如 8192、64K 或 1M。",
			EN: "Max output tokens must be a positive count, like 8192, 64K, or 1M.",
		},
		"advancedHint": {
			ZH: "其余字段在 cordis.patch.yml 中，请直接编辑对应段。",
			EN: "Other fields live in cordis.patch.yml; edit that section directly.",
		},
		"modelCapacityInvalid": {
			ZH: "容量需为数字，可加 K 或 M 后缀。",
			EN: "A capacity must be a number, optionally suffixed K or M.",
		},
		"modelDuplicate": {
			ZH: "每个模型 ID 只能出现一次。",
			EN: "Each model ID may appear once.",
		},
		"fetchModels": {
			ZH: "获取可用模型",
			EN: "Fetch available models",
		},
		"fetching": {
			ZH: "正在询问提供商…",
			EN: "Asking the provider…",
		},
		"fetchNeedsBaseUrl": {
			ZH: "请先填写 API 地址，再获取。",
			EN: "Enter the base URL first, then fetch.",
		},
		"fetchEmpty": {
			ZH: "该提供商没有列出任何模型，请手动添加。",
			EN: "The provider listed no models. Add them by hand.",
		},
		"fetchTitle": {
			ZH: "选择要添加的模型",
			EN: "Choose models to add",
		},
		"fetchDescription": {
			ZH: "以下是模型提供商的可用模型，勾选要添加的模型。",
			EN: "These are the models this provider has available. Choose the ones to add.",
		},
		"fetchSearch": {
			ZH: "搜索模型",
			EN: "Search models",
		},
		"fetchNoMatches": {
			ZH: "没有匹配的模型。",
			EN: "No matching models.",
		},
		"fetchSelectAll": {
			ZH: "全选",
			EN: "Select all",
		},
		"fetchDeselectAll": {
			ZH: "取消全选",
			EN: "Deselect all",
		},
		"fetchAdopt": {
			ZH: "添加所选",
			EN: "Add selected",
		},
		"customTag": {
			ZH: "自定义",
			EN: "Custom",
		},
		"customRoute": {
			ZH: "Provider ID",
			EN: "Provider ID",
		},
		"customRouteHint": {
			ZH: "以小写字母开头的标识，在请求中唯一标识该提供商，并用于派生凭据名。",
			EN: "Lowercase identifier, starting with a letter, that uniquely names this provider in requests and as its credential name.",
		},
		"customRouteInvalid": {
			ZH: "需以小写字母开头，之后可用小写字母、数字和短横线。",
			EN: "Start with a lowercase letter; then lowercase letters, digits, and dashes.",
		},
		"customRouteTaken": {
			ZH: "已有提供商使用了这个 ID。",
			EN: "A provider already uses this ID.",
		},
		"customDisplayName": {
			ZH: "显示名称",
			EN: "Display name",
		},
		"customApi": {
			ZH: "API 协议",
			EN: "API protocol",
		},
		"customApiUnset": {
			ZH: "未选择",
			EN: "Not selected",
		},
		"protocolOpenAiCompletions": {
			ZH: "OpenAI Chat Completions",
			EN: "OpenAI Chat Completions",
		},
		"protocolOpenAiResponses": {
			ZH: "OpenAI Responses",
			EN: "OpenAI Responses",
		},
		"protocolAnthropicMessages": {
			ZH: "Anthropic Messages",
			EN: "Anthropic Messages",
		},
		"customNeedsBaseUrl": {
			ZH: "自定义模型 API 需要填写 API 地址。",
			EN: "A custom provider needs a base URL.",
		},
		"customBaseUrlInvalid": {
			ZH: "请输入有效的 HTTP 或 HTTPS 地址。",
			EN: "Enter a valid HTTP or HTTPS URL.",
		},
		"customNeedsModels": {
			ZH: "自定义模型 API 至少需要一个模型。",
			EN: "A custom provider needs at least one model.",
		},
		"customBaseUrlPlaceholder": {
			ZH: "https://gateway.example/v1",
			EN: "https://gateway.example/v1",
		},
		"customAnthropicBaseUrlPlaceholder": {
			ZH: "https://gateway.example",
			EN: "https://gateway.example",
		},
		"settingsPathUnresolvable": {
			ZH: "无法解析设置路径",
			EN: "unresolvable settings path",
		},
		"create": {
			ZH: "创建提供商",
			EN: "Create provider",
		},
		"creating": {
			ZH: "创建中…",
			EN: "Creating…",
		},
		"welcomeTitle": {
			ZH: "预览版说明",
			EN: "Preview Notice",
		},
		"welcomeBody": {
			ZH: `DeepSeek Harness 目前的 0.2 版本仍处于预览阶段，还有许多地方需要持续改进和打磨，希望听取广大开发者和用户的反馈建议。现在，新的桌面端面向广泛用户，开发者相关的进阶功能可在配置中开启使用。预计 DeepSeek Harness 的产品功能以及插件 API 都会继续快速迭代、持续演化，并逐渐趋于稳定。

我们期待与全球用户和开发者一起，在开源、可复用、可组合的基础设施之上，共同探索智能上限。欢迎大家用 DeepSeek Harness 将想法变成现实，与社区一起丰富插件生态。`,
			EN: `DeepSeek Harness 0.2 is still in preview, and many areas need continued improvement and refinement. We welcome feedback and suggestions from all developers and users. The new desktop app now targets a broad range of users, while developer-related advanced features can be enabled in the settings. DeepSeek Harness’s product features and plugin APIs are expected to continue rapid iteration and evolution, and will gradually stabilize over time.

We look forward to exploring the limits of intelligence together with users and developers around the world, building on open-source, reusable, and composable infrastructure. We welcome everyone to bring their ideas to life with DeepSeek Harness and participate in the community to enrich the plugin ecosystem.`,
		},
		"welcomeContinue": {
			ZH: "继续",
			EN: "Continue",
		},
		"welcomeError": {
			ZH: "暂时无法保存确认状态，请重试。",
			EN: "The acknowledgement could not be saved. Please try again.",
		},
		"onboardingTitle": {
			ZH: "添加一个 API Key 开始使用",
			EN: "Add an API key to get started",
		},
		"onboardingDescription": {
			ZH: "配置 DeepSeek 官方模型，即可开始使用。",
			EN: "Configure the official DeepSeek provider to start building.",
		},
		"onboardingLater": {
			ZH: "稍后配置",
			EN: "Configure later",
		},
		"onboardingSave": {
			ZH: "保存并继续",
			EN: "Save and continue",
		},
		"onboardingSaving": {
			ZH: "保存中…",
			EN: "Saving…",
		},
		"keyRequired": {
			ZH: "请输入 API 密钥后继续。",
			EN: "Enter an API key to continue.",
		},
		"visionModel": {
			ZH: "识图模型",
			EN: "Vision model",
		},
		"visionModelHint": {
			ZH: "主模型不支持图片时，先由该模型识别图片内容，再把描述交给主模型处理。请选择一个支持图片输入的模型。",
			EN: "When the main model cannot read images, this model describes them first and the description is handed to the main model. Pick a model that accepts image input.",
		},
		"visionModelOff": {
			ZH: "不启用",
			EN: "Off",
		},
		"visionModelSaveFailed": {
			ZH: "保存识图模型失败",
			EN: "Saving the vision model failed",
		},
		"visionModelLoadFailed": {
			ZH: "加载模型列表失败",
			EN: "Loading the model list failed",
		},
		"inputTitle": {
			ZH: "输入类型",
			EN: "Input types",
		},
		"inputText": {
			ZH: "文本",
			EN: "Text",
		},
		"inputImage": {
			ZH: "图像",
			EN: "Image",
		},
		"inputInherited": {
			ZH: "未设置——继承提供方默认值。",
			EN: "Not set — inherits the provider defaults.",
		},
		"defaultInputTitle": {
			ZH: "默认输入类型",
			EN: "Default input types",
		},
		"defaultInputHint": {
			ZH: "应用于本路由下未声明的模型。任何受支持协议都必然接受文本。",
			EN: "Applies to models on this route that declare none. Text is always accepted by any supported protocol.",
		},
		"defaultInputEmpty": {
			ZH: "请至少选择一种输入类型。",
			EN: "Pick at least one input type.",
		},
	},
	"dsh-client-ui-settings-plugin-inventory": {
		"tab": {
			ZH: "插件列表",
			EN: "Plugin list",
		},
		"loading": {
			ZH: "正在读取插件…",
			EN: "Reading plugins…",
		},
		"clientSyncing": {
			ZH: "正在同步本页面的插件…",
			EN: "Syncing plugins on this page…",
		},
		"clientSyncFailed": {
			ZH: "本页面的插件未能完成同步；服务端的启用状态保持不变。",
			EN: "Some plugins could not sync on this page. Host enablement is unchanged.",
		},
		"clientSyncRetry": {
			ZH: "重试本页面同步",
			EN: "Retry this page",
		},
		"dismissFeedback": {
			ZH: "关闭提示",
			EN: "Dismiss notice",
		},
		"error": {
			ZH: "暂时无法读取插件。",
			EN: "Plugins are temporarily unavailable.",
		},
		"metadataError": {
			ZH: "包元信息错误：{error}",
			EN: "Package metadata error: {error}",
		},
		"retry": {
			ZH: "重试",
			EN: "Retry",
		},
		"search": {
			ZH: "搜索插件",
			EN: "Search plugins",
		},
		"empty": {
			ZH: "暂无插件。",
			EN: "No plugins are available.",
		},
		"emptySearch": {
			ZH: "没有匹配的插件。",
			EN: "No matching plugins.",
		},
		"presetTitle": {
			ZH: "会话插件",
			EN: "Session plugins",
		},
		"presetSubtitle": {
			ZH: "由 Agent 预设按会话组成",
			EN: "Composed per session by agent presets",
		},
		"switcherLabel": {
			ZH: "选择要查看的 Agent 预设",
			EN: "Choose the agent preset to inspect",
		},
		"presetOptionDefault": {
			ZH: "{name}（默认）",
			EN: "{name} (default)",
		},
		"presetOptionBroken": {
			ZH: "{name}（加载失败）",
			EN: "{name} (failed to load)",
		},
		"globalTitle": {
			ZH: "全局插件",
			EN: "Global plugins",
		},
		"globalSubtitle": {
			ZH: "系统与所有会话共用",
			EN: "Shared by the system and every session",
		},
		"presetProvidedDetail": {
			ZH: "全局已停用，由 Agent 预设按会话提供",
			EN: "Disabled globally; agent presets provide it per session",
		},
		"enabledIn": {
			ZH: "启用于",
			EN: "Enabled in",
		},
		"matchesInOtherPresets": {
			ZH: "其他预设中还有 {count} 个匹配：",
			EN: "{count} more matches in other presets: ",
		},
		"enabledTag": {
			ZH: "已启用",
			EN: "Enabled",
		},
		"disabledTag": {
			ZH: "已停用",
			EN: "Disabled",
		},
		"conditionalTag": {
			ZH: "条件启用",
			EN: "Conditional",
		},
		"presetEnabledTag": {
			ZH: "预设中启用",
			EN: "Via presets",
		},
		"failedTag": {
			ZH: "启动失败",
			EN: "Failed",
		},
		"moduleLabel": {
			ZH: "完整名称",
			EN: "Module",
		},
		"fromPreset": {
			ZH: "来自",
			EN: "From",
		},
		"condition": {
			ZH: "禁用条件",
			EN: "Disabled when",
		},
		"configuration": {
			ZH: "配置状态",
			EN: "Configuration",
		},
		"runtime": {
			ZH: "运行状态",
			EN: "Status",
		},
		"unobserved": {
			ZH: "未运行",
			EN: "Not running",
		},
		"pending": {
			ZH: "等待依赖",
			EN: "Waiting for dependencies",
		},
		"loadingPhase": {
			ZH: "加载中",
			EN: "Loading",
		},
		"active": {
			ZH: "运行中",
			EN: "Running",
		},
		"failed": {
			ZH: "启动失败",
			EN: "Failed to start",
		},
		"unloading": {
			ZH: "卸载中",
			EN: "Unloading",
		},
	},
	"dsh-client-ui-settings-plugins": {
		"nav": {
			ZH: "内置插件",
			EN: "Built-in plugins",
		},
		"title": {
			ZH: "内置插件",
			EN: "Built-in plugins",
		},
		"intro": {
			ZH: "查看内置部署的插件列表",
			EN: "Inspect the plugins this deployment ships.",
		},
		"tabs": {
			ZH: "插件视图",
			EN: "Plugin views",
		},
		"empty": {
			ZH: "本部署没有开放任何插件视图。",
			EN: "This deployment exposes no plugin views.",
		},
	},
	"dsh-client-ui-settings-remote": {
		"trigger": {
			ZH: "远程",
			EN: "Remote",
		},
		"heading": {
			ZH: "远程",
			EN: "Remote",
		},
		"nav": {
			ZH: "远程",
			EN: "Remote",
		},
		"title": {
			ZH: "远程",
			EN: "Remote",
		},
		"intro": {
			ZH: "手机在侧栏扫码配对：Android App 内扫码＝链接设备，相机 / 浏览器扫码＝打开 web 端。此处只改连接方式与中继。",
			EN: "Pair from the sidebar: scanning inside the Android app links the device; a camera / browser scan opens the web client. Set LAN / Server and the relay here.",
		},
		"tabs": {
			ZH: "远程分区标签",
			EN: "Remote section tabs",
		},
		"empty": {
			ZH: "还没有可显示的标签页。",
			EN: "No tabs to show yet.",
		},
		"gatewayTab": {
			ZH: "网关",
			EN: "Gateway",
		},
		"channelsTab": {
			ZH: "消息渠道",
			EN: "Channels",
		},
		"enable": {
			ZH: "开启配对",
			EN: "Start pairing",
		},
		"enabledOn": {
			ZH: "开启",
			EN: "On",
		},
		"enabledOff": {
			ZH: "关闭",
			EN: "Off",
		},
		"mode": {
			ZH: "连接方式",
			EN: "Connection",
		},
		"modeLan": {
			ZH: "局域网",
			EN: "LAN",
		},
		"modeRelay": {
			ZH: "服务器",
			EN: "Server",
		},
		"modeHint": {
			ZH: "局域网：扫码打开本机 :3180 网页。服务器：扫码打开公网网页；数据走中继。不要用微信转发链接（会丢掉 #offer=）。",
			EN: "LAN: scan to open this machine's :3180 page. Server: scan opens the public web page; traffic uses the relay. Do not forward links in WeChat — it strips #offer=.",
		},
		"relayNeedsBoth": {
			ZH: "请先填写并保存自建中继主机（host:443），再选择服务器。",
			EN: "Save your self-hosted relay host (host:443) before choosing Server.",
		},
		"relayNeedsUrl": {
			ZH: "请填写并保存自建中继主机后再选择服务器。",
			EN: "Save your self-hosted relay host before choosing Server.",
		},
		"relayNeedsToken": {
			ZH: "请填写并保存自建中继主机后再选择服务器。",
			EN: "Save your self-hosted relay host before choosing Server.",
		},
		"relayNeedsEndpoint": {
			ZH: "桌面已内置默认中继，可直接选择服务器。",
			EN: "Built-in relay is ready — Server works out of the box.",
		},
		"bindScope": {
			ZH: "监听范围",
			EN: "Listen scope",
		},
		"bindAll": {
			ZH: "整个局域网",
			EN: "Whole LAN",
		},
		"bindLoopback": {
			ZH: "仅本机",
			EN: "This machine only",
		},
		"bindLoopbackHint": {
			ZH: "仅本机监听：局域网设备无法直连。",
			EN: "Loopback only: other LAN devices cannot connect.",
		},
		"bindUnavailable": {
			ZH: "已失效",
			EN: "unavailable",
		},
		"lanTransport": {
			ZH: "传输加密",
			EN: "Transport",
		},
		"transportPlain": {
			ZH: "明文",
			EN: "Plain",
		},
		"transportTls": {
			ZH: "TLS",
			EN: "TLS",
		},
		"lanPlaintextWarning": {
			ZH: "局域网直连仅限可信网络。",
			EN: "LAN direct connect is for trusted networks only.",
		},
		"lanTlsHint": {
			ZH: "手机会话使用端到端加密，中继 TLS 由中继主机配置决定；局域网配对页使用 HTTP，只提供应用文件。",
			EN: "Phone sessions use end-to-end encryption. Relay TLS follows the relay host configuration; the LAN pairing page uses HTTP and only serves app files.",
		},
		"port": {
			ZH: "监听端口",
			EN: "Listen port",
		},
		"portHint": {
			ZH: "本机 daemon 默认 6767；局域网配对页另用 3180。",
			EN: "The local daemon defaults to 6767; the LAN pairing page uses 3180 separately.",
		},
		"portInvalid": {
			ZH: "端口必须是 1024 到 65535 之间的整数。",
			EN: "Port must be an integer from 1024 to 65535.",
		},
		"relayUrl": {
			ZH: "中继主机",
			EN: "Relay host",
		},
		"relayUrlHint": {
			ZH: "默认 ayase.cn:443（TLS），一般不用改。",
			EN: "Defaults to ayase.cn:443 (TLS) — usually leave it.",
		},
		"relayUrlPlaceholder": {
			ZH: "ayase.cn:443",
			EN: "ayase.cn:443",
		},
		"relayUseDefault": {
			ZH: "恢复默认",
			EN: "Reset default",
		},
		"relayToken": {
			ZH: "（已退役）宿主令牌",
			EN: "(Retired) host token",
		},
		"relayTokenHint": {
			ZH: "dshd 配对不使用宿主令牌；设备密钥在扫码后自动绑定，直至解除。",
			EN: "dshd pairing does not use a host token; device secrets bind after scan until you unbind.",
		},
		"relayTokenPlaceholder": {
			ZH: "",
			EN: "",
		},
		"relayTokenSet": {
			ZH: "中继已配置",
			EN: "Relay configured",
		},
		"relayTokenClear": {
			ZH: "清除",
			EN: "Clear",
		},
		"legacyConfiguration": {
			ZH: "旧版配置清理",
			EN: "Legacy configuration cleanup",
		},
		"legacyEmpty": {
			ZH: "没有遗留的宿主令牌。",
			EN: "No retired host token is stored.",
		},
		"rotateToken": {
			ZH: "刷新配对码",
			EN: "Refresh pairing code",
		},
		"rotateTokenHint": {
			ZH: "只换二维码；已配对设备仍有效，除非解除配对。",
			EN: "New QR only; paired devices stay until you unbind.",
		},
		"rotateTokenConfirm": {
			ZH: "刷新",
			EN: "Refresh",
		},
		"save": {
			ZH: "保存",
			EN: "Save",
		},
		"saved": {
			ZH: "已保存",
			EN: "Saved",
		},
		"savedServiceError": {
			ZH: "配置已保存，但远程服务异常：{message}",
			EN: "Settings saved, but the remote service has an error: {message}",
		},
		"dismiss": {
			ZH: "关闭提示",
			EN: "Dismiss message",
		},
		"copyFailed": {
			ZH: "无法复制链接，请重试。",
			EN: "Could not copy the link. Please try again.",
		},
		"qr": {
			ZH: "配对二维码",
			EN: "Pairing QR code",
		},
		"pairingUrl": {
			ZH: "配对链接",
			EN: "Pairing link",
		},
		"scanSplitHint": {
			ZH: "同一张码：Android App 内扫码＝链接设备；手机相机 / 浏览器扫码＝打开 web 端。",
			EN: "One code: scan inside the Android app to link the device; a camera / browser scan opens the web client.",
		},
		"relayDown": {
			ZH: "中继未连接，正在重试",
			EN: "Relay offline — retrying",
		},
		"relayDownWithError": {
			ZH: "中继未连接：{message}",
			EN: "Relay offline: {message}",
		},
		"relayDownDisconnected": {
			ZH: "中继已断开，正在重试",
			EN: "Relay disconnected — retrying",
		},
		"relayUnavailable": {
			ZH: "默认中继暂时不可用",
			EN: "Default relay is unavailable",
		},
		"startingHint": {
			ZH: "正在启动远程服务…",
			EN: "Starting remote…",
		},
		"mintingHint": {
			ZH: "正在生成配对码…",
			EN: "Creating pairing code…",
		},
		"copyLink": {
			ZH: "复制链接",
			EN: "Copy link",
		},
		"copiedLink": {
			ZH: "已复制",
			EN: "Copied",
		},
		"statusErrorGeneric": {
			ZH: "远程暂时不可用，请关闭后再开启重试",
			EN: "Remote unavailable — turn off then on to retry",
		},
		"errorPortInUse": {
			ZH: "配对端口被占用，请关闭占用程序后点「开启」重试",
			EN: "Pairing port is in use — free it, then press On",
		},
		"errorBindGone": {
			ZH: "监听网卡已失效——到 设置 → 远程 改选「监听范围」后重试",
			EN: "Saved listen address is gone — pick a new Listen scope under Settings → Remote, then retry",
		},
		"offHint": {
			ZH: "开启后显示扫码配对二维码",
			EN: "Turn on to show the pairing QR code",
		},
		"noQr": {
			ZH: "还没有可扫描的配对二维码",
			EN: "No pairing QR code yet",
		},
		"notListening": {
			ZH: "远程服务未在运行——点「开启」重试",
			EN: "Remote service is not running — press On to retry",
		},
		"devices": {
			ZH: "已配对设备",
			EN: "Paired devices",
		},
		"devicesManage": {
			ZH: "设备管理",
			EN: "Manage devices",
		},
		"devicesEmpty": {
			ZH: "还没有已配对的设备。扫码配对后会出现在这里。",
			EN: "No paired devices yet. They appear after a successful scan.",
		},
		"devicesOnline": {
			ZH: "在线",
			EN: "Online",
		},
		"devicesSeen": {
			ZH: "最近访问 {time}",
			EN: "Last seen {time}",
		},
		"devicesSeenUnknown": {
			ZH: "尚未访问",
			EN: "Not seen yet",
		},
		"devicesBound": {
			ZH: "配对于 {time}",
			EN: "Paired {time}",
		},
		"devicesId": {
			ZH: "编号 {id}",
			EN: "Id {id}",
		},
		"unbind": {
			ZH: "解除配对",
			EN: "Unbind",
		},
		"rename": {
			ZH: "重命名",
			EN: "Rename",
		},
		"renameLabel": {
			ZH: "设备名称",
			EN: "Device name",
		},
		"renameCancel": {
			ZH: "取消",
			EN: "Cancel",
		},
		"loading": {
			ZH: "正在读取…",
			EN: "Loading…",
		},
		"error": {
			ZH: "暂时无法读取远程状态。",
			EN: "Could not read remote status.",
		},
		"retry": {
			ZH: "重试",
			EN: "Retry",
		},
		"statusError": {
			ZH: "远程出错：{message}",
			EN: "Remote error: {message}",
		},
		"relayConnectingHint": {
			ZH: "正在连接中继…",
			EN: "Connecting to relay…",
		},
		"pairingLifetimeHint": {
			ZH: "二维码过期或使用后自动刷新；复制的链接长期有效。",
			EN: "The QR refreshes after expiry or use. Copied links stay valid until revoked.",
		},
		"persistentLinkHint": {
			ZH: "长期链接可重复配对，在这里撤销。已配对设备不受影响。",
			EN: "Reusable links can pair new devices until revoked here. Paired devices stay connected.",
		},
		"revokeLink": {
			ZH: "撤销长期链接",
			EN: "Revoke persistent link",
		},
		"revokeLinkConfirm": {
			ZH: "撤销后，旧链接不能再配对；已配对设备保持连接。",
			EN: "The old link will stop pairing new devices. Paired devices stay connected.",
		},
		"revokeLinkAction": {
			ZH: "撤销并生成新链接",
			EN: "Revoke and replace link",
		},
		"linkRevoked": {
			ZH: "旧链接已撤销，可重新复制新链接。",
			EN: "Old link revoked. You can copy the replacement link.",
		},
	},
	"dsh-client-ui-settings-session-log": {
		"title": {
			ZH: "在使用官方模型 API 时上传 Session Log",
			EN: "Upload Session Log when using the official model API",
		},
		"description": {
			ZH: "帮助改进 DeepSeek 模型与产品",
			EN: "Help improve DeepSeek models and products.",
		},
		"saved": {
			ZH: "设置已保存",
			EN: "Preference saved",
		},
		"failed": {
			ZH: "无法保存设置",
			EN: "Could not save preference",
		},
	},
	"dsh-client-ui-settings-shell": {
		"title": {
			ZH: "终端",
			EN: "Shell",
		},
		"description": {
			ZH: "限制每条命令最多能跑多久、最多输出多少内容。",
			EN: "Limit how long each command may run and how much it may output.",
		},
		"timeoutMs": {
			ZH: "命令超时（毫秒）",
			EN: "Command timeout (ms)",
		},
		"timeoutMsHint": {
			ZH: "单条命令允许运行多久，超时即终止。",
			EN: "How long one command may run before it is terminated.",
		},
		"maxOutputBytes": {
			ZH: "单流输出上限（字节）",
			EN: "Output cap per stream (bytes)",
		},
		"maxOutputBytesHint": {
			ZH: "超出部分会转存到临时文件，而不是被丢弃。",
			EN: "Output beyond this spills to a temporary file rather than being lost.",
		},
		"overridden": {
			ZH: "已覆盖",
			EN: "Overridden",
		},
		"reset": {
			ZH: "恢复默认",
			EN: "Reset to default",
		},
		"readOnly": {
			ZH: "本部署的设置为只读。",
			EN: "This deployment stores settings read-only.",
		},
		"unavailable": {
			ZH: "该插件当前未加载，暂时无法配置。",
			EN: "This plugin is not loaded, so it cannot be configured right now.",
		},
		"save": {
			ZH: "保存",
			EN: "Save",
		},
		"saving": {
			ZH: "保存中…",
			EN: "Saving…",
		},
		"saveFailed": {
			ZH: "本部署没有接受这些值，已保留供你修改。",
			EN: "The deployment did not accept these values; they were left for you to correct.",
		},
		"invalidNumber": {
			ZH: "请填数字；留空表示使用默认值。",
			EN: "Enter a number, or leave blank to use the default.",
		},
	},
	"dsh-client-ui-settings-skills": {
		"nav": {
			ZH: "技能",
			EN: "Skills",
		},
		"title": {
			ZH: "技能",
			EN: "Skills",
		},
		"intro": {
			ZH: "查找并管理用户、当前项目与捆绑技能。",
			EN: "Find and manage user, current-project, and bundled skills.",
		},
		"projectCatalogUnavailable": {
			ZH: "当前没有活动项目，因此未加载项目技能。",
			EN: "No project is active, so project skills are not loaded.",
		},
		"sessionCatalogUnavailable": {
			ZH: "当前会话暂不可用，已改用全局技能目录。",
			EN: "The current session is unavailable, so the global skill catalog is shown.",
		},
		"add": {
			ZH: "添加技能",
			EN: "Add skill",
		},
		"refresh": {
			ZH: "刷新",
			EN: "Refresh",
		},
		"hubOpen": {
			ZH: "浏览 GitHub 技能",
			EN: "Browse GitHub skills",
		},
		"hubTitle": {
			ZH: "GitHub 技能库",
			EN: "GitHub Skill Hub",
		},
		"hubIntro": {
			ZH: "搜索公开 GitHub 仓库中的 Agent Skills，并安装到真实的 DSH 技能目录。",
			EN: "Search public GitHub repositories for Agent Skills and install them into a real DSH skills directory.",
		},
		"hubScope": {
			ZH: "安装位置",
			EN: "Install location",
		},
		"hubUserScope": {
			ZH: "将安装到当前用户的 DSH 技能目录。",
			EN: "Installs into the current user DSH skills directory.",
		},
		"hubProjectScope": {
			ZH: "将安装到当前项目：{cwd}",
			EN: "Installs into the current project: {cwd}",
		},
		"hubSearchLabel": {
			ZH: "搜索 GitHub 技能",
			EN: "Search GitHub skills",
		},
		"hubSearchPlaceholder": {
			ZH: "例如 browser、review、terraform",
			EN: "For example: browser, review, terraform",
		},
		"hubSearch": {
			ZH: "搜索",
			EN: "Search",
		},
		"hubSearching": {
			ZH: "正在搜索…",
			EN: "Searching…",
		},
		"hubSearchHint": {
			ZH: "输入关键词后搜索公开技能。",
			EN: "Enter a keyword to search public skills.",
		},
		"hubNoResults": {
			ZH: "没有找到匹配的 GitHub 技能。",
			EN: "No matching GitHub skills were found.",
		},
		"hubSearchFailed": {
			ZH: "GitHub 技能搜索失败。",
			EN: "GitHub skill search failed.",
		},
		"hubInstall": {
			ZH: "安装",
			EN: "Install",
		},
		"hubInstalled": {
			ZH: "已安装",
			EN: "Installed",
		},
		"hubConflict": {
			ZH: "同名冲突",
			EN: "Name conflict",
		},
		"hubConfirmTitle": {
			ZH: "确认安装技能",
			EN: "Confirm skill installation",
		},
		"hubConfirmBody": {
			ZH: "请核对技能的公开来源和安装范围。",
			EN: "Review the public source and installation scope before continuing.",
		},
		"hubSkill": {
			ZH: "技能",
			EN: "Skill",
		},
		"hubRepository": {
			ZH: "仓库",
			EN: "Repository",
		},
		"hubPath": {
			ZH: "路径",
			EN: "Path",
		},
		"hubNoOverwrite": {
			ZH: "安装不会静默覆盖已有技能；若名称冲突，操作会失败并保留现有文件。",
			EN: "Installation never silently overwrites an existing skill. A conflict fails and preserves the current files.",
		},
		"hubInstallConfirm": {
			ZH: "确认安装",
			EN: "Install skill",
		},
		"hubInstalling": {
			ZH: "正在安装…",
			EN: "Installing…",
		},
		"hubInstallFailed": {
			ZH: "GitHub 技能安装失败。",
			EN: "GitHub skill installation failed.",
		},
		"empty": {
			ZH: "还没有技能。",
			EN: "No skills yet.",
		},
		"noResults": {
			ZH: "没有符合当前搜索和筛选条件的技能。",
			EN: "No skills match the current search and filters.",
		},
		"loading": {
			ZH: "正在读取技能…",
			EN: "Reading skills…",
		},
		"error": {
			ZH: "暂时无法读取技能。",
			EN: "Skills are temporarily unavailable.",
		},
		"retry": {
			ZH: "重试",
			EN: "Retry",
		},
		"refreshFailed": {
			ZH: "更改已保存，但技能列表刷新失败。",
			EN: "The change was saved, but the skill list could not be refreshed.",
		},
		"searchLabel": {
			ZH: "搜索技能",
			EN: "Search skills",
		},
		"searchPlaceholder": {
			ZH: "搜索名称、描述、分组或何时使用",
			EN: "Search name, description, group, or when to use",
		},
		"sourceFilter": {
			ZH: "来源",
			EN: "Source",
		},
		"filterAll": {
			ZH: "全部",
			EN: "All",
		},
		"clearFilters": {
			ZH: "清除筛选",
			EN: "Clear filters",
		},
		"resultCount": {
			ZH: "工作区与个人技能 {count} 项",
			EN: "{count} workspace and personal skills",
		},
		"group": {
			ZH: "分组",
			EN: "Group",
		},
		"groupPlaceholder": {
			ZH: "输入分组后回车添加，可多个",
			EN: "Type a group and press Enter to add; groups combine",
		},
		"groupOptionsLabel": {
			ZH: "选择已有分组",
			EN: "Choose groups",
		},
		"groupClearOption": {
			ZH: "清空分组（未分组）",
			EN: "Clear groups (ungrouped)",
		},
		"groupRemoveFor": {
			ZH: "移除分组 {group}",
			EN: "Remove group {group}",
		},
		"ungrouped": {
			ZH: "未分组",
			EN: "Ungrouped",
		},
		"groupCount": {
			ZH: "{count} 项",
			EN: "{count} skills",
		},
		"groupToggleFor": {
			ZH: "切换 {group} 组的模型调用",
			EN: "Toggle model invocation for group {group}",
		},
		"openDirectoryFor": {
			ZH: "打开 {name} 所在目录",
			EN: "Open the directory containing {name}",
		},
		"openDirectoryFailed": {
			ZH: "无法打开目录。",
			EN: "Could not open the directory.",
		},
		"sourceUser": {
			ZH: "个人",
			EN: "Personal",
		},
		"sourceProject": {
			ZH: "当前项目",
			EN: "Current project",
		},
		"sourceBundled": {
			ZH: "捆绑",
			EN: "Bundled",
		},
		"sourceOther": {
			ZH: "其他",
			EN: "Other",
		},
		"model": {
			ZH: "模型调用",
			EN: "Model invocation",
		},
		"user": {
			ZH: "用户调用",
			EN: "User invocation",
		},
		"invocationTitle": {
			ZH: "调用方式",
			EN: "Invocation",
		},
		"modelHint": {
			ZH: "允许模型在合适时自动调用此技能。",
			EN: "Allow the model to invoke this skill when appropriate.",
		},
		"userHint": {
			ZH: "允许用户通过技能名称直接调用。",
			EN: "Allow users to invoke this skill directly by name.",
		},
		"modelFor": {
			ZH: "{name} 的模型调用",
			EN: "Model invocation for {name}",
		},
		"userFor": {
			ZH: "{name} 的用户调用",
			EN: "User invocation for {name}",
		},
		"remove": {
			ZH: "删除",
			EN: "Delete",
		},
		"removeFor": {
			ZH: "删除 {name}",
			EN: "Delete {name}",
		},
		"loadingDetail": {
			ZH: "正在读取技能详情…",
			EN: "Reading skill details…",
		},
		"detailFailed": {
			ZH: "无法读取技能详情。",
			EN: "Could not read skill details.",
		},
		"invocationFailed": {
			ZH: "无法更新调用方式。",
			EN: "Could not update invocation settings.",
		},
		"whenToUse": {
			ZH: "何时使用",
			EN: "When to use",
		},
		"editorTitleAdd": {
			ZH: "添加技能",
			EN: "Add skill",
		},
		"editorTitleEdit": {
			ZH: "编辑技能",
			EN: "Edit skill",
		},
		"scope": {
			ZH: "保存位置",
			EN: "Save location",
		},
		"scopeUser": {
			ZH: "用户",
			EN: "User",
		},
		"scopeProject": {
			ZH: "当前项目",
			EN: "Current project",
		},
		"projectUnavailable": {
			ZH: "打开一个项目或会话后，才能添加项目技能。",
			EN: "Open a project or session to add a project skill.",
		},
		"projectPath": {
			ZH: "将保存到当前项目：{cwd}",
			EN: "This will be saved in the current project: {cwd}",
		},
		"name": {
			ZH: "名称",
			EN: "Name",
		},
		"description": {
			ZH: "描述",
			EN: "Description",
		},
		"content": {
			ZH: "正文",
			EN: "Body",
		},
		"save": {
			ZH: "保存",
			EN: "Save",
		},
		"saving": {
			ZH: "正在保存…",
			EN: "Saving…",
		},
		"cancel": {
			ZH: "取消",
			EN: "Cancel",
		},
		"close": {
			ZH: "关闭",
			EN: "Close",
		},
		"deleteTitle": {
			ZH: "删除 {name}？",
			EN: "Delete {name}?",
		},
		"deleteBody": {
			ZH: "将永久删除 {name} 的整个技能目录，包括 SKILL.md 和目录内所有附带文件。此操作无法撤销。",
			EN: "This permanently deletes the entire skill directory for {name}, including SKILL.md and every accompanying file in that directory. This cannot be undone.",
		},
		"deleteConfirm": {
			ZH: "删除技能目录",
			EN: "Delete skill directory",
		},
		"deleting": {
			ZH: "正在删除…",
			EN: "Deleting…",
		},
		"saveFailed": {
			ZH: "保存失败。",
			EN: "Save failed.",
		},
		"deleteFailed": {
			ZH: "删除失败。",
			EN: "Delete failed.",
		},
		"nameRequired": {
			ZH: "名称必须使用小写 kebab-case，例如 review-code。",
			EN: "Use a lowercase kebab-case name, such as review-code.",
		},
		"descriptionRequired": {
			ZH: "请输入技能描述。",
			EN: "Enter a skill description.",
		},
		"contentRequired": {
			ZH: "请输入技能正文。",
			EN: "Enter the skill body.",
		},
	},
	"dsh-client-ui-settings-subagent": {
		"overridden": {
			ZH: "已覆盖",
			EN: "Overridden",
		},
		"reset": {
			ZH: "恢复默认",
			EN: "Reset to default",
		},
		"readOnly": {
			ZH: "本部署的设置为只读。",
			EN: "This deployment stores settings read-only.",
		},
		"unavailable": {
			ZH: "该插件当前未加载，暂时无法配置。",
			EN: "This plugin is not loaded, so it cannot be configured right now.",
		},
		"save": {
			ZH: "保存",
			EN: "Save",
		},
		"saving": {
			ZH: "保存中…",
			EN: "Saving…",
		},
		"saveFailed": {
			ZH: "本部署没有接受这些值，已保留供你修改。",
			EN: "The deployment did not accept these values; they were left for you to correct.",
		},
		"dismissFeedback": {
			ZH: "关闭提示",
			EN: "Dismiss",
		},
		"discardDraft": {
			ZH: "放弃修改",
			EN: "Discard changes",
		},
		"subagentTitle": {
			ZH: "子智能体",
			EN: "Subagent",
		},
		"subagentDescription": {
			ZH: "设置子智能体的递归层级、数量和模型。",
			EN: "Set Subagent recursion depth, count, and models.",
		},
		"subagentLimitsTitle": {
			ZH: "运行限制",
			EN: "Limits",
		},
		"subagentMaxDepth": {
			ZH: "最大递归深度",
			EN: "Maximum recursion depth",
		},
		"subagentDepthHelpLabel": {
			ZH: "最大递归深度说明",
			EN: "About maximum recursion depth",
		},
		"subagentDepthHelp": {
			ZH: "限制 Agent 创建子智能体的递归层级。",
			EN: "Limits how many levels of Subagents an Agent can create.",
		},
		"subagentDepthZero": {
			ZH: "禁用子智能体",
			EN: "Disable Subagents",
		},
		"subagentDepthOne": {
			ZH: "仅允许主 Agent 创建子智能体",
			EN: "Only the main Agent can create Subagents",
		},
		"subagentDepthOverride": {
			ZH: "如果某个工具单独设置了最大递归深度，以该工具的设置为准。",
			EN: "If a tool defines its own maximum recursion depth, that setting takes precedence.",
		},
		"subagentMaxActive": {
			ZH: "子智能体并行数量上限",
			EN: "Subagent parallelism limit",
		},
		"subagentCapacityHelpLabel": {
			ZH: "子智能体并行数量上限说明",
			EN: "About the Subagent parallelism limit",
		},
		"subagentCapacityHelp": {
			ZH: "同一主 Agent 下，所有递归层级同时存活的子智能体总数，主 Agent 不计入。达到上限时，新的启动请求会被拒绝。",
			EN: "Total live Subagents under the same main Agent, across all recursion levels. The main Agent is excluded. New start requests are rejected when the limit is reached.",
		},
		"subagentDepthInvalid": {
			ZH: "请输入不小于 0 的整数。",
			EN: "Enter a whole number of 0 or more.",
		},
		"subagentCapacityInvalid": {
			ZH: "请输入不小于 1 的整数。",
			EN: "Enter a whole number of 1 or more.",
		},
		"subagentModelSelectionTitle": {
			ZH: "模型选择",
			EN: "Model selection",
		},
		"subagentModelSelectionToggle": {
			ZH: "允许 Agent 为子智能体选择模型",
			EN: "Allow agents to choose models for Subagents",
		},
		"subagentModelSelectionChoose": {
			ZH: "开启后，Agent 可以从下方授权模型中，为每个子智能体选择提供方、模型和推理强度。仅影响新会话。",
			EN: "When enabled, agents can choose a provider, model, and reasoning effort for each Subagent from the authorized models below. Applies only to new sessions.",
		},
		"subagentModelSelectionAllowed": {
			ZH: "Agent 可选择的模型",
			EN: "Models agents may choose",
		},
		"subagentModelSelectionLoading": {
			ZH: "正在加载模型…",
			EN: "Loading models…",
		},
		"subagentModelSelectionLoadFailed": {
			ZH: "无法加载模型。",
			EN: "Models could not be loaded.",
		},
		"subagentModelSelectionRetry": {
			ZH: "重试",
			EN: "Retry",
		},
		"subagentModelSelectionPartial": {
			ZH: "部分模型提供方暂时无法加载；已保存的选择仍可移除。",
			EN: "Some model providers could not be loaded; saved choices remain removable.",
		},
		"subagentModelSelectionUnavailable": {
			ZH: "当前不可用",
			EN: "Currently unavailable",
		},
		"subagentModelSelectionUnavailableGroup": {
			ZH: "已保存但当前不可用",
			EN: "Saved but currently unavailable",
		},
		"subagentModelSelectionEmpty": {
			ZH: "当前没有模型提供方公布模型。",
			EN: "No model provider currently advertises a model.",
		},
		"subagentModelSelectionRequired": {
			ZH: "保存前请至少选择一个模型。",
			EN: "Select at least one model before saving.",
		},
		"subagentModelSelectionConflict": {
			ZH: "设置已在其他位置更新。请放弃修改后重试。",
			EN: "Settings changed elsewhere. Discard your draft and try again.",
		},
		"subagentModelSelectionOff": {
			ZH: "关闭后，子智能体使用配置的默认模型或继承父 Agent 的模型；已选模型会保留。",
			EN: "Subagents use configured defaults or inherit the parent agent's model. Saved model choices are retained.",
		},
	},
	"dsh-client-ui-settings-web-search": {
		"title": {
			ZH: "网页搜索",
			EN: "Web search",
		},
		"description": {
			ZH: "设置 DeepSeek 的搜索提供方。",
			EN: "Set up the DeepSeek search provider.",
		},
		"useSearchApi": {
			ZH: "启用新版 Web 搜索 API",
			EN: "Use the new Web Search API",
		},
		"useSearchApiHint": {
			ZH: "默认关闭，搜索通过 DeepSeek Messages 完成。新版 Web 搜索 API 功能与计费详见",
			EN: "Off by default: search runs through DeepSeek Messages. For the new Web Search API's features and pricing, see the ",
		},
		"searchGuide": {
			ZH: "网页搜索与抓取指南",
			EN: "Search & Fetch guide",
		},
		"searchGuideUrl": {
			ZH: "https://api-docs.deepseek.com/zh-cn/guides/search",
			EN: "https://api-docs.deepseek.com/guides/search",
		},
		"apiKey": {
			ZH: "API Key",
			EN: "API key",
		},
		"apiKeyHint": {
			ZH: "不写入设置文件。留空表示保持当前密钥。",
			EN: "Stored outside the settings file. Leave blank to keep the current key.",
		},
		"apiKeySet": {
			ZH: "已配置密钥。",
			EN: "A key is configured.",
		},
		"apiKeyUnset": {
			ZH: "未配置密钥；仅使用 DeepSeek 账号模型的对话可以通过默认接口地址搜索。",
			EN: "No key is configured; only conversations using a DeepSeek Account model can search, through the default endpoint.",
		},
		"baseUrl": {
			ZH: "Messages API 基础地址",
			EN: "Messages API base URL",
		},
		"baseUrlHint": {
			ZH: "填写含 API 前缀的地址，例如 https://api.deepseek.com/anthropic/v1；程序追加 /messages，留空恢复默认。",
			EN: "Include the API prefix, such as https://api.deepseek.com/anthropic/v1. The provider appends /messages. Leave blank to restore the default.",
		},
		"searchApiBaseUrl": {
			ZH: "Search / Fetch API 基础地址",
			EN: "Search / Fetch API base URL",
		},
		"searchApiBaseUrlHint": {
			ZH: "填写含 API 前缀的地址，例如 https://api.deepseek.com/v1/tools；程序追加 /search 或 /fetch，留空恢复默认。",
			EN: "Include the API prefix, such as https://api.deepseek.com/v1/tools. The provider appends /search or /fetch. Leave blank to restore the default.",
		},
		"maxUses": {
			ZH: "单次请求最多搜索次数",
			EN: "Max searches per request",
		},
		"maxUsesHint": {
			ZH: "辅助模型在一次 Messages 请求内最多可执行的搜索次数，仅适用于 Messages API。",
			EN: "How many searches the auxiliary model may run within one Messages request. Applies only to the Messages API.",
		},
		"count": {
			ZH: "每次搜索结果数",
			EN: "Results per search",
		},
		"countHint": {
			ZH: "每个查询返回的结果条数，须为 1–50 的整数，默认 10；不控制搜索调用次数。web_search 保留的来源数不超过其自身的结果上限。",
			EN: "Results returned per query: an integer from 1 to 50, default 10. This controls result count, not search calls. web_search keeps no more sources than its own result limit.",
		},
		"overridden": {
			ZH: "已覆盖",
			EN: "Overridden",
		},
		"reset": {
			ZH: "恢复默认",
			EN: "Reset to default",
		},
		"readOnly": {
			ZH: "本部署的设置为只读。",
			EN: "This deployment stores settings read-only.",
		},
		"unavailable": {
			ZH: "该插件当前未加载，暂时无法配置。",
			EN: "This plugin is not loaded, so it cannot be configured right now.",
		},
		"save": {
			ZH: "保存",
			EN: "Save",
		},
		"saving": {
			ZH: "保存中…",
			EN: "Saving…",
		},
		"saveFailed": {
			ZH: "本部署没有接受这些值，已保留供你修改。",
			EN: "The deployment did not accept these values; they were left for you to correct.",
		},
		"invalidNumber": {
			ZH: "请填数字；留空表示使用默认值。",
			EN: "Enter a number, or leave blank to use the default.",
		},
		"invalidCount": {
			ZH: "请填 1–50 的整数；留空表示使用默认值。",
			EN: "Enter an integer from 1 to 50, or leave blank to use the default.",
		},
	},
	"dsh-client-ui-shortcuts": {
		"edit-label": {
			ZH: "修改{command}快捷键",
			EN: "Edit shortcut for {command}",
		},
		"record": {
			ZH: "按下快捷键",
			EN: "Press a shortcut",
		},
		"record-help": {
			ZH: "松开组合键即保存。Tab 切换操作，Esc 退出录键。",
			EN: "Release the keys to save. Tab moves between actions; Esc cancels.",
		},
		"web-help": {
			ZH: "浏览器可用组合：Mod+/、Mod+Shift+,、Mod+Shift+.。Mod 在 Mac 上为 Command，其他系统为 Ctrl。",
			EN: "Browser combinations: Mod+/, Mod+Shift+,, Mod+Shift+.. Mod is Command on Mac and Ctrl elsewhere.",
		},
		"unsupported-key": {
			ZH: "暂不支持这个按键。",
			EN: "This key is not supported.",
		},
		"reserved": {
			ZH: "此组合由系统或文本编辑操作保留。",
			EN: "This combination is reserved for system or text editing actions.",
		},
		"modifier-required": {
			ZH: "请同时按下 Command、Ctrl 或 Alt 修饰键。",
			EN: "Include Command, Ctrl, or Alt in the combination.",
		},
		"too-many-keys": {
			ZH: "最多同时按下两个非修饰键。请松开后重试。",
			EN: "Hold at most two non-modifier keys. Release the keys to try again.",
		},
		"macos-web-help": {
			ZH: "可设置 Command+/、Command+,、Command+反斜杠、Control+反引号、Command+Option+按键或 Command+Shift+按键。也支持三个或四个不同修饰键的组合。浏览器或系统占用的组合可能无法送达页面。",
			EN: "Use Command+/, Command+,, Command+Backslash, Control+Backquote, Command+Option+key, or Command+Shift+key. Combinations with three or four distinct modifiers are also supported. Browser or system shortcuts may not reach the page.",
		},
		"windows-web-help": {
			ZH: "可设置 Ctrl+/、Ctrl+,、Ctrl+Alt+按键或 Ctrl+Shift+按键。也支持三个或四个不同修饰键的组合。浏览器或系统占用的组合可能无法送达页面。",
			EN: "Use Ctrl+/, Ctrl+,, Ctrl+Alt+key, or Ctrl+Shift+key. Combinations with three or four distinct modifiers are also supported. Browser or system shortcuts may not reach the page.",
		},
		"unsupported-browser": {
			ZH: "此浏览器暂不支持该组合。",
			EN: "This browser does not support this combination yet.",
		},
		"conflict": {
			ZH: "已被「{commands}」占用",
			EN: "Already used by “{commands}”",
		},
		"saved": {
			ZH: "已修改",
			EN: "Modified",
		},
		"clear": {
			ZH: "移除",
			EN: "Remove",
		},
		"retry-save": {
			ZH: "重试保存",
			EN: "Retry save",
		},
		"reset": {
			ZH: "恢复默认",
			EN: "Restore default",
		},
		"reset-all": {
			ZH: "恢复全部默认",
			EN: "Restore all defaults",
		},
		"modified-count": {
			ZH: "{count} 项已自定义",
			EN: "{count} customized",
		},
		"reset-title": {
			ZH: "恢复全部默认快捷键？",
			EN: "Restore all default shortcuts?",
		},
		"reset-description": {
			ZH: "将恢复当前平台的默认快捷键。你修改或移除的快捷键都会恢复，其他平台不受影响。",
			EN: "Restore the default shortcuts for this platform. All modified or removed shortcuts will be restored. Other platforms are unaffected.",
		},
		"cancel": {
			ZH: "取消",
			EN: "Cancel",
		},
		"reset-saved": {
			ZH: "已恢复默认快捷键",
			EN: "Default shortcuts restored",
		},
		"close-confirmation": {
			ZH: "关闭确认",
			EN: "Close confirmation",
		},
		"reset-failed": {
			ZH: "恢复默认失败。原快捷键已保留，请重试。",
			EN: "Could not restore defaults. Your shortcuts are unchanged. Please retry.",
		},
		"review": {
			ZH: "已核对最新配置",
			EN: "I have reviewed the latest configuration",
		},
		"stale": {
			ZH: "快捷键配置或可用命令已更新。请核对最新键位后再保存。",
			EN: "Shortcut configuration or available commands changed. Review the latest bindings before saving.",
		},
		"write-failed": {
			ZH: "保存失败。原快捷键和当前草稿已保留，请重试。",
			EN: "Could not save. Your previous shortcuts and current draft are preserved. Please retry.",
		},
		"not-ready": {
			ZH: "快捷键尚未准备好，请稍后重试。",
			EN: "Shortcuts are not ready. Please try again.",
		},
		"read": {
			ZH: "无法读取 {location}。请检查访问权限后{reload}。",
			EN: "Could not read {location}. Check access permissions, then {reload}.",
		},
		"invalid": {
			ZH: "{location} 的快捷键配置内容损坏。请先备份并修复该配置，然后{reload}。",
			EN: "Shortcut configuration in {location} is damaged. Back up and repair this configuration, then {reload}.",
		},
		"future": {
			ZH: "{location} 的快捷键配置由更新版本创建。请升级 Harness 后重试。",
			EN: "Shortcut configuration in {location} was created by a newer version. Upgrade Harness and try again.",
		},
		"web-document": {
			ZH: "当前网站本地存储中的 dsh.keybindings.v1",
			EN: "this site’s localStorage entry dsh.keybindings.v1",
		},
		"desktop-document": {
			ZH: "userData/keybindings.json",
			EN: "userData/keybindings.json",
		},
		"web-reload": {
			ZH: "刷新页面",
			EN: "reload the page",
		},
		"desktop-reload": {
			ZH: "重新启动 Harness",
			EN: "restart Harness",
		},
		"using-defaults": {
			ZH: "当前使用默认键位。",
			EN: "Default bindings are active.",
		},
		"using-accepted": {
			ZH: "当前继续使用上次读取成功的键位。",
			EN: "The last successfully read bindings remain active.",
		},
		"native-failed": {
			ZH: "无法启用桌面录键保护，请关闭后重试。",
			EN: "Could not protect desktop key recording. Exit recording and try again.",
		},
		"global-hint": {
			ZH: "全局唤起查看",
			EN: "Open from anywhere",
		},
		"clear-search": {
			ZH: "清空搜索",
			EN: "Clear search",
		},
		"title": {
			ZH: "快捷键",
			EN: "Keyboard shortcuts",
		},
		"open": {
			ZH: "快捷键速查",
			EN: "Open keyboard shortcuts",
		},
		"settings": {
			ZH: "快捷键",
			EN: "Keyboard shortcuts",
		},
		"view": {
			ZH: "编辑快捷键",
			EN: "Edit shortcuts",
		},
		"description": {
			ZH: "查看和编辑当前可用的快捷键和输入操作",
			EN: "View and edit available shortcuts and input actions",
		},
		"search": {
			ZH: "搜索快捷键",
			EN: "Search shortcuts",
		},
		"close": {
			ZH: "关闭快捷键",
			EN: "Close keyboard shortcuts",
		},
		"application": {
			ZH: "应用操作",
			EN: "Application",
		},
		"input": {
			ZH: "消息输入",
			EN: "Message input",
		},
		"menus": {
			ZH: "菜单与弹层",
			EN: "Menus and dialogs",
		},
		"approval": {
			ZH: "审批区域",
			EN: "Approval area",
		},
		"unbound": {
			ZH: "暂无快捷键",
			EN: "No shortcut",
		},
		"empty": {
			ZH: "没有匹配的快捷键",
			EN: "No matching shortcuts",
		},
		"move": {
			ZH: "移动菜单选择",
			EN: "Move menu selection",
		},
		"select": {
			ZH: "选择菜单项",
			EN: "Select menu item",
		},
		"dismiss": {
			ZH: "关闭菜单或顶层弹窗",
			EN: "Close menu or top dialog",
		},
	},
	"dsh-client-ui-theme": {
		"fontFamily.text.title": {
			ZH: "正文字体",
			EN: "Text font",
		},
		"fontFamily.text.description": {
			ZH: "用于界面与会话正文。多个字体用逗号分隔，留空使用默认字体",
			EN: "Used for the interface and conversation text. Separate fonts with commas; leave empty for the default",
		},
		"fontFamily.code.title": {
			ZH: "代码字体",
			EN: "Code font",
		},
		"fontFamily.code.description": {
			ZH: "用于代码块、行内代码与工具输出。多个字体用逗号分隔，留空使用默认字体",
			EN: "Used for code blocks, inline code, and tool output. Separate fonts with commas; leave empty for the default",
		},
		"fontFamily.terminal.title": {
			ZH: "终端字体",
			EN: "Terminal font",
		},
		"fontFamily.terminal.description": {
			ZH: "用于侧栏终端。多个字体用逗号分隔，留空使用默认字体",
			EN: "Used for the sidebar terminal. Separate fonts with commas; leave empty for the default",
		},
		"fontFamily.placeholder": {
			ZH: "默认",
			EN: "Default",
		},
		"fontSize.more": {
			ZH: "更多字体设置",
			EN: "More font settings",
		},
		"nav": {
			ZH: "外观",
			EN: "Appearance",
		},
		"appearance.title": {
			ZH: "外观",
			EN: "Appearance",
		},
		"appearance.light": {
			ZH: "浅色",
			EN: "Light",
		},
		"appearance.dark": {
			ZH: "深色",
			EN: "Dark",
		},
		"appearance.system": {
			ZH: "跟随系统",
			EN: "System",
		},
		"fontSize.text.title": {
			ZH: "字号大小",
			EN: "Font size",
		},
		"fontSize.text.description": {
			ZH: "仅影响会话内容的字号",
			EN: "Only affects conversation content",
		},
		"fontSize.text.increase": {
			ZH: "增大字号",
			EN: "Increase font size",
		},
		"fontSize.text.decrease": {
			ZH: "减小字号",
			EN: "Decrease font size",
		},
		"fontSize.code.title": {
			ZH: "代码字号",
			EN: "Code font size",
		},
		"fontSize.code.description": {
			ZH: "用于代码块、行内代码与工具输出",
			EN: "Used for code blocks, inline code, and tool output",
		},
		"fontSize.code.increase": {
			ZH: "增大代码字号",
			EN: "Increase code font size",
		},
		"fontSize.code.decrease": {
			ZH: "减小代码字号",
			EN: "Decrease code font size",
		},
		"fontSize.terminal.title": {
			ZH: "终端字号",
			EN: "Terminal font size",
		},
		"fontSize.terminal.description": {
			ZH: "用于侧栏终端",
			EN: "Used for the sidebar terminal",
		},
		"fontSize.terminal.increase": {
			ZH: "增大终端字号",
			EN: "Increase terminal font size",
		},
		"fontSize.terminal.decrease": {
			ZH: "减小终端字号",
			EN: "Decrease terminal font size",
		},
		"fontSize.unit": {
			ZH: "px",
			EN: "px",
		},
		"fontSize.increase": {
			ZH: "增大字号",
			EN: "Increase font size",
		},
		"fontSize.decrease": {
			ZH: "减小字号",
			EN: "Decrease font size",
		},
		"scheme.title": {
			ZH: "颜色模式",
			EN: "Color scheme",
		},
		"scheme.description": {
			ZH: "选择浅色、深色或跟随系统。",
			EN: "Choose whether the light half, dark half, or the system setting is active.",
		},
		"library.title": {
			ZH: "主题",
			EN: "Themes",
		},
		"library.create": {
			ZH: "创建主题",
			EN: "Create theme",
		},
		"library.import": {
			ZH: "导入主题",
			EN: "Import theme",
		},
		"library.export": {
			ZH: "导出",
			EN: "Export",
		},
		"library.duplicate": {
			ZH: "复制",
			EN: "Duplicate",
		},
		"library.delete": {
			ZH: "删除",
			EN: "Delete",
		},
		"library.edit": {
			ZH: "编辑",
			EN: "Edit",
		},
		"library.advanced": {
			ZH: "高级 token",
			EN: "Advanced tokens",
		},
		"library.currentMode": {
			ZH: "当前模式",
			EN: "Current mode",
		},
		"editor.previewHintDark": {
			ZH: "修改会实时预览。界面正处于深色模式，只有「深色主题」的颜色立即可见；「浅色主题」在切换到浅色模式后生效。",
			EN: "Edits preview live. The UI is in dark mode, so only the dark half is visible right now; the light half applies after switching to light mode.",
		},
		"editor.previewHintLight": {
			ZH: "修改会实时预览。界面正处于浅色模式，只有「浅色主题」的颜色立即可见；「深色主题」在切换到深色模式后生效。",
			EN: "Edits preview live. The UI is in light mode, so only the light half is visible right now; the dark half applies after switching to dark mode.",
		},
		"editor.name": {
			ZH: "名称",
			EN: "Name",
		},
		"editor.accent": {
			ZH: "强调色",
			EN: "Accent",
		},
		"editor.accentHint": {
			ZH: "强调色会铺到发送按钮、用户气泡和侧栏选中项。",
			EN: "The accent paints the send button, user bubbles, and the selected sidebar item.",
		},
		"editor.background": {
			ZH: "背景",
			EN: "Background",
		},
		"editor.foreground": {
			ZH: "前景",
			EN: "Foreground",
		},
		"editor.contrast": {
			ZH: "对比度",
			EN: "Contrast",
		},
		"editor.light": {
			ZH: "浅色主题",
			EN: "Light half",
		},
		"editor.dark": {
			ZH: "深色主题",
			EN: "Dark half",
		},
		"editor.save": {
			ZH: "保存",
			EN: "Save",
		},
		"editor.cancel": {
			ZH: "取消",
			EN: "Cancel",
		},
		"glass.title": {
			ZH: "玻璃透明度",
			EN: "Glass opacity",
		},
		"glass.description": {
			ZH: "控制界面表层的实心程度：菜单、对话框、输入框，以及设置背景图后的侧栏和聊天区域。数值越低越通透。有背景图时画布不会完全实心，图始终能透出一层。",
			EN: "Controls how solid UI surfaces are: menus, dialogs, the composer, and — with a wallpaper set — the sidebar and chat area. Lower values are more see-through. With a wallpaper, the canvas never goes fully opaque, so the image always shows through a little.",
		},
		"glass.transparent": {
			ZH: "透明主题",
			EN: "Transparent theme",
		},
		"glass.transparentHint": {
			ZH: "开启后界面表层完全透明：侧栏、输入框、菜单和对话框都不再有遮罩，壁纸色彩全量透出。文字可读性可以用「毛玻璃程度」调节；生效时若毛玻璃低于 20% 会自动提到 20%。",
			EN: "Makes every UI surface fully transparent: the sidebar, composer, menus, and dialogs drop their masks so the wallpaper colors show through everywhere. Use the frosted-glass slider to keep text readable; taking effect raises a frosted glass below 20% to 20% automatically.",
		},
		"glass.transparentBlurHint": {
			ZH: "毛玻璃程度低于 20%，文字会直接贴在清晰壁纸上，可能难以阅读。建议把「毛玻璃程度」调高。",
			EN: "Frosted glass is below 20%, so text sits directly on the sharp wallpaper and may be hard to read. Raise the frosted-glass slider.",
		},
		"glass.transparentNeedsWallpaper": {
			ZH: "透明主题需要先设置背景图才会生效。",
			EN: "The transparent theme takes effect after a wallpaper is set.",
		},
		"glass.sidebarMask": {
			ZH: "隐藏侧栏遮罩",
			EN: "Hide sidebar mask",
		},
		"glass.sidebarMaskHint": {
			ZH: "开启后左侧栏不再叠加遮罩，与工作区底色保持一致，只保留中间的分割线。",
			EN: "The left sidebar drops its extra surface fill and shares the workspace canvas color; only the divider between them stays.",
		},
		"glass.terminal": {
			ZH: "终端透明度",
			EN: "Terminal opacity",
		},
		"glass.terminalHint": {
			ZH: "终端面板自己的实心程度，不跟随上面的玻璃滑杆。有背景图或流动渐变时，按这个比例透出背景。",
			EN: "The terminal pane’s own solidity, independent of the glass slider above. With a wallpaper or the flowing gradient set, the backdrop shows through at this rate.",
		},
		"glass.terminalDeepHint": {
			ZH: "低于 75% 时，只用加粗和颜色标选中行的终端程序可能看不清选中了哪一行。",
			EN: "Below 75%, terminal apps that mark the selected row with only bold text and a color may make it hard to tell which row is selected.",
		},
		"glass.terminalNeedsBackdrop": {
			ZH: "设置背景图或开启「流动渐变」背景特效后生效。",
			EN: "Takes effect once a wallpaper or the flowing gradient backdrop effect is set.",
		},
		"wallpaper.title": {
			ZH: "背景图",
			EN: "Wallpaper",
		},
		"wallpaper.description": {
			ZH: "选一张图铺在整个界面后面。图要透过界面才能看见：玻璃透明度越高，图越被盖住。设好之后可以拉毛玻璃和像素化。",
			EN: "Choose an image to sit behind the whole UI. It shows through the chrome: higher glass opacity hides more of it. Frost and pixelate sliders appear after it is set.",
		},
		"wallpaper.skin": {
			ZH: "使用主题壁纸",
			EN: "Theme wallpaper",
		},
		"wallpaper.skinHint": {
			ZH: "当前主题自带一张默认背景图；关闭后回到你自己的背景图或背景特效。",
			EN: "Use the active theme’s default backdrop. Turn it off to fall back to your own wallpaper or the ambient effect.",
		},
		"wallpaper.choose": {
			ZH: "选择图片",
			EN: "Choose image",
		},
		"wallpaper.clear": {
			ZH: "清除",
			EN: "Clear",
		},
		"wallpaper.blur": {
			ZH: "毛玻璃程度",
			EN: "Frosted glass",
		},
		"wallpaper.pixelate": {
			ZH: "像素化程度",
			EN: "Pixelation",
		},
		"wallpaper.glassHint": {
			ZH: "玻璃透明度偏高时背景图会被盖住。把玻璃透明度调低，图会更清楚。",
			EN: "High glass opacity covers the wallpaper. Lower Glass opacity to let more of the image show through.",
		},
		"wallpaper.browse": {
			ZH: "浏览图库",
			EN: "Browse gallery",
		},
		"wallpaper.close": {
			ZH: "关闭",
			EN: "Close",
		},
		"wallpaper.sources": {
			ZH: "图源",
			EN: "Sources",
		},
		"wallpaper.backToGallery": {
			ZH: "返回图库",
			EN: "Back to gallery",
		},
		"wallpaper.addSource": {
			ZH: "新增图源",
			EN: "Add source",
		},
		"wallpaper.editSource": {
			ZH: "编辑图源",
			EN: "Edit source",
		},
		"wallpaper.sourceKind": {
			ZH: "类型",
			EN: "Type",
		},
		"wallpaper.sourceName": {
			ZH: "显示名",
			EN: "Display name",
		},
		"wallpaper.sourceUrl": {
			ZH: "HTTPS 目录地址",
			EN: "HTTPS catalog URL",
		},
		"wallpaper.sourcesHint": {
			ZH: "分类来自这里。必应和 Wallhaven 各只能有一条。",
			EN: "Categories come from here. Bing and Wallhaven may each appear only once.",
		},
		"wallpaper.sourceKindBing": {
			ZH: "必应",
			EN: "Bing",
		},
		"wallpaper.sourceKindWallhaven": {
			ZH: "Wallhaven",
			EN: "Wallhaven",
		},
		"wallpaper.sourceKindCatalog": {
			ZH: "目录",
			EN: "Catalog",
		},
		"wallpaper.sourceExists": {
			ZH: "已经添加过这个图源。",
			EN: "That source is already added.",
		},
		"wallpaper.bing": {
			ZH: "Bing 每日壁纸",
			EN: "Bing daily wallpapers",
		},
		"wallpaper.bingHint": {
			ZH: "从 Bing 获取最近的壁纸。需要联网。",
			EN: "Fetch recent wallpapers from Bing. An internet connection is required.",
		},
		"wallpaper.catalogHint": {
			ZH: "也可以添加 HTTPS JSON 壁纸目录。",
			EN: "You can also add HTTPS JSON wallpaper catalogs.",
		},
		"wallpaper.catalogUrls": {
			ZH: "壁纸目录地址",
			EN: "Wallpaper catalog URLs",
		},
		"wallpaper.catalogPlaceholder": {
			ZH: "https://example.com/wallpapers.json",
			EN: "https://example.com/wallpapers.json",
		},
		"wallpaper.addCatalog": {
			ZH: "添加目录",
			EN: "Add catalog",
		},
		"wallpaper.removeCatalog": {
			ZH: "移除",
			EN: "Remove",
		},
		"wallpaper.catalogRejected": {
			ZH: "地址无效、重复，或已达到目录数量上限。",
			EN: "The URL is invalid, duplicated, or the catalog limit is reached.",
		},
		"wallpaper.downloading": {
			ZH: "正在下载壁纸…",
			EN: "Downloading wallpaper…",
		},
		"wallpaper.galleryEmpty": {
			ZH: "没有可用壁纸。",
			EN: "No wallpapers available.",
		},
		"wallpaper.galleryFailed": {
			ZH: "图库读取失败。",
			EN: "The gallery could not be loaded.",
		},
		"wallpaper.downloadFailed": {
			ZH: "壁纸下载失败。",
			EN: "The wallpaper download failed.",
		},
		"wallpaper.invalidImage": {
			ZH: "无法读取这张图片。",
			EN: "This image could not be read.",
		},
		"wallpaper.fileTooLarge": {
			ZH: "图片不能超过 12 MB。",
			EN: "Images must be 12 MB or smaller.",
		},
		"wallpaper.favorites": {
			ZH: "收藏",
			EN: "Favorites",
		},
		"wallpaper.favorite": {
			ZH: "收藏这张",
			EN: "Favorite this image",
		},
		"wallpaper.today": {
			ZH: "今日",
			EN: "Today",
		},
		"wallpaper.wallhavenGeneral": {
			ZH: "常规",
			EN: "General",
		},
		"wallpaper.wallhavenAnime": {
			ZH: "动漫",
			EN: "Anime",
		},
		"wallpaper.wallhavenPeople": {
			ZH: "人物",
			EN: "People",
		},
		"wallpaper.search": {
			ZH: "搜索",
			EN: "Search",
		},
		"wallpaper.loadMore": {
			ZH: "加载更多",
			EN: "Load more",
		},
		"wallpaper.loading": {
			ZH: "加载中…",
			EN: "Loading…",
		},
		"wallpaper.confirmSet": {
			ZH: "将这张图设为背景？",
			EN: "Set this image as the wallpaper?",
		},
		"wallpaper.confirmSetYes": {
			ZH: "设为壁纸",
			EN: "Set wallpaper",
		},
		"wallpaper.confirmSetNo": {
			ZH: "取消",
			EN: "Cancel",
		},
		"wallpaper.crop": {
			ZH: "调整背景图",
			EN: "Adjust wallpaper",
		},
		"wallpaper.cropHint": {
			ZH: "拖动图片或使用滚轮调整取景。",
			EN: "Drag the image or use the wheel to adjust the crop.",
		},
		"wallpaper.use": {
			ZH: "使用此图片",
			EN: "Use this image",
		},
		"wallpaper.cropFailed": {
			ZH: "图片处理失败，请重新选择。",
			EN: "Image processing failed. Choose another image.",
		},
		"wallpaper.zoom": {
			ZH: "缩放",
			EN: "Zoom",
		},
		"effect.title": {
			ZH: "背景特效",
			EN: "Background effect",
		},
		"effect.description": {
			ZH: "不设置背景图时，在整个界面后面铺一层缓慢流动的渐变光斑。",
			EN: "Without a wallpaper, a slowly flowing gradient blooms behind the whole UI.",
		},
		"effect.gradient": {
			ZH: "流动渐变",
			EN: "Flowing gradient",
		},
		"effect.pausedByWallpaper": {
			ZH: "当前已设置背景图，特效暂不生效；清除背景图后自动恢复。",
			EN: "A wallpaper is set, so the effect is paused; it returns when the wallpaper is cleared.",
		},
		"effect.colors": {
			ZH: "配色",
			EN: "Colors",
		},
		"effect.colorStart": {
			ZH: "底色·起",
			EN: "Base · start",
		},
		"effect.colorEnd": {
			ZH: "底色·止",
			EN: "Base · end",
		},
		"effect.colorBloom": {
			ZH: "光斑",
			EN: "Bloom",
		},
		"effect.speed": {
			ZH: "流动速度",
			EN: "Flow speed",
		},
		"effect.count": {
			ZH: "光斑数量",
			EN: "Bloom count",
		},
		"effect.configure": {
			ZH: "特效设置",
			EN: "Effect settings",
		},
		"effect.dialogDesc": {
			ZH: "选一个预设配色，或直接逐项调整；保存后生效。",
			EN: "Pick a preset palette or tune each value directly; changes apply on save.",
		},
		"effect.scheme": {
			ZH: "方案",
			EN: "Scheme",
		},
		"effect.preset.default": {
			ZH: "跟随主题",
			EN: "Follow theme",
		},
		"effect.preset.aurora": {
			ZH: "极光",
			EN: "Aurora",
		},
		"effect.preset.sunset": {
			ZH: "晚霞",
			EN: "Sunset",
		},
		"effect.preset.ocean": {
			ZH: "海洋",
			EN: "Ocean",
		},
		"effect.preset.sakura": {
			ZH: "樱花",
			EN: "Sakura",
		},
		"effect.preset.custom": {
			ZH: "自定义",
			EN: "Custom",
		},
		"effect.variant": {
			ZH: "光斑形态",
			EN: "Bloom shape",
		},
		"effect.variant.orbs": {
			ZH: "光球",
			EN: "Orbs",
		},
		"effect.variant.aurora": {
			ZH: "极光带",
			EN: "Aurora ribbons",
		},
		"effect.variant.chaos": {
			ZH: "混沌",
			EN: "Chaos",
		},
		"effect.variant.rays": {
			ZH: "光束",
			EN: "Rays",
		},
		"effect.cancel": {
			ZH: "取消",
			EN: "Cancel",
		},
		"effect.save": {
			ZH: "保存",
			EN: "Save",
		},
		"effect.close": {
			ZH: "关闭",
			EN: "Close",
		},
		"effect.lowPowerGpu": {
			ZH: "检测到集成显卡，背景特效默认关闭；开启后可能影响流畅度。",
			EN: "An integrated GPU was detected, so the background effect starts off; turning it on may make the app less smooth.",
		},
		"preset.defaultLight.label": {
			ZH: "DeepSeek 浅色",
			EN: "DeepSeek Light",
		},
		"preset.defaultDark.label": {
			ZH: "DeepSeek 深色",
			EN: "DeepSeek Dark",
		},
		"preset.midnight.label": {
			ZH: "午夜",
			EN: "Midnight",
		},
		"preset.celadon.label": {
			ZH: "青瓷",
			EN: "Celadon",
		},
		"preset.violet.label": {
			ZH: "暮紫",
			EN: "Violet",
		},
		"preset.amber.label": {
			ZH: "琥珀",
			EN: "Amber",
		},
		"preset.paper.label": {
			ZH: "宣纸",
			EN: "Paper",
		},
		"preset.contrast.label": {
			ZH: "对比",
			EN: "Contrast",
		},
		"cursorFx.title": {
			ZH: "指针特效",
			EN: "Cursor effects",
		},
		"cursorFx.description": {
			ZH: "在指针划过处绘制装饰特效，可选像素拖尾或流体飞溅。",
			EN: "Paint a decorative effect where the pointer moves: a fading pixel trail or a fluid splash.",
		},
		"cursorFx.dialogDesc": {
			ZH: "选择一种指针特效和方案，或切到自定义方案逐项调整；保存后生效。",
			EN: "Pick an effect and a preset scheme, or switch to the custom scheme and tune each value; changes apply on save.",
		},
		"cursorFx.kind": {
			ZH: "特效类型",
			EN: "Effect",
		},
		"cursorFx.kind.trail": {
			ZH: "像素拖尾",
			EN: "Pixel trail",
		},
		"cursorFx.kind.splash": {
			ZH: "流体飞溅",
			EN: "Fluid splash",
		},
		"cursorFx.previewHint": {
			ZH: "在预览区移动指针查看效果",
			EN: "Move the pointer over the preview to try the effect",
		},
		"cursorFx.preset.default": {
			ZH: "跟随主题",
			EN: "Follow theme",
		},
		"cursorFx.preset.rainbow": {
			ZH: "彩虹",
			EN: "Rainbow",
		},
		"cursorFx.preset.aurora": {
			ZH: "极光",
			EN: "Aurora",
		},
		"cursorFx.preset.sunset": {
			ZH: "晚霞",
			EN: "Sunset",
		},
		"cursorFx.preset.ocean": {
			ZH: "海洋",
			EN: "Ocean",
		},
		"cursorFx.preset.sakura": {
			ZH: "樱花",
			EN: "Sakura",
		},
		"cursorFx.preset.custom": {
			ZH: "自定义",
			EN: "Custom",
		},
		"cursorFx.color": {
			ZH: "颜色",
			EN: "Color",
		},
		"cursorFx.speed": {
			ZH: "速度",
			EN: "Speed",
		},
		"cursorFx.size": {
			ZH: "大小",
			EN: "Size",
		},
		"metallicPaint.title": {
			ZH: "按钮悬停光泽",
			EN: "Button sheen",
		},
		"metallicPaint.description": {
			ZH: "鼠标悬停在按钮上时叠加一层金属扫光；关闭后只保留原本的悬停效果。",
			EN: "Overlay a metallic sheen sweep on hovered buttons; off keeps only the plain hover fill.",
		},
		"type.title": {
			ZH: "字体",
			EN: "Typography",
		},
		"type.advanced": {
			ZH: "高级",
			EN: "Advanced",
		},
		"type.interface": {
			ZH: "界面字体",
			EN: "Interface font",
		},
		"type.interfaceHint": {
			ZH: "填写本机已安装的字体名（例如 Microsoft YaHei）。留空则用系统默认。用于代码块和终端以外的文字。",
			EN: "Type a font name installed on this computer (for example Segoe UI). Leave empty for the system default. Used for text outside code blocks and the terminal.",
		},
		"type.code": {
			ZH: "等宽字体",
			EN: "Monospace font",
		},
		"type.codeHint": {
			ZH: "填写本机已安装的等宽字体名。留空则用系统默认。用于代码块、diff 和文件预览。",
			EN: "Type a monospace font name installed on this computer. Leave empty for the system default. Used for code blocks, diffs, and file previews.",
		},
		"type.size": {
			ZH: "字号",
			EN: "Size",
		},
		"type.sizeCode": {
			ZH: "代码字号",
			EN: "Code size",
		},
		"type.composer": {
			ZH: "输入框字体",
			EN: "Composer font",
		},
		"type.composerHint": {
			ZH: "填写本机已安装的字体名。留空则跟随界面字体。",
			EN: "Type a font name installed on this computer. Leave empty to follow the interface font.",
		},
		"type.terminal": {
			ZH: "终端字体",
			EN: "Terminal font",
		},
		"type.terminalHint": {
			ZH: "填写本机已安装的等宽字体名。留空则跟随等宽字体。",
			EN: "Type a monospace font name installed on this computer. Leave empty to follow the monospace font.",
		},
		"type.placeholder": {
			ZH: "系统默认",
			EN: "System default",
		},
		"reset": {
			ZH: "重置",
			EN: "Reset",
		},
	},
	"dsh-experimental-client-ui-cot-translation": {
		"provider": {
			ZH: "翻译服务",
			EN: "Translation service",
		},
		"google": {
			ZH: "Google",
			EN: "Google",
		},
		"bing": {
			ZH: "Bing",
			EN: "Bing",
		},
		"deepseekAccount": {
			ZH: "DeepSeek Flash · 账号（付费）",
			EN: "DeepSeek Flash — Account (paid)",
		},
		"deepseekOfficial": {
			ZH: "DeepSeek Flash · 官方 API（付费）",
			EN: "DeepSeek Flash — Official API (paid)",
		},
		"providerUnavailableOption": {
			ZH: "{provider} — 不可用",
			EN: "{provider} — unavailable",
		},
		"providerUnavailable": {
			ZH: "此服务暂不可用，请配置 DeepSeek 账号或官方 API 服务，或选择可用的翻译服务",
			EN: "This provider is unavailable. Configure your DeepSeek account or Official API provider, or choose an available service.",
		},
		"paidNotice": {
			ZH: "每个未缓存片段都会通过所选账号或官方 API 凭据单独发送一次付费的 DeepSeek Flash 请求；重新展开会复用已保存的译文，更改服务、语言、原文或请求设置可能发送新的付费请求",
			EN: "Each uncached fragment sends a separate paid DeepSeek Flash request through the selected account or Official API credentials. Reopening reasoning reuses saved results; changing the provider, language, source text, or request settings may send a new paid request.",
		},
		"anonymousNotice": {
			ZH: "Bing 和 Google 翻译无需额外登录或 API key，但匿名接口可能拒绝或限制请求",
			EN: "Bing and Google need no translation login or API key. Their anonymous endpoints may reject or limit requests.",
		},
		"targetLanguage": {
			ZH: "目标语言",
			EN: "Target language",
		},
		"targetLanguageHint": {
			ZH: "auto 使用界面语言；也可填写语言代码，例如 zh、en、ja",
			EN: "auto follows the UI language; language codes such as zh, en, and ja also work.",
		},
		"privacy": {
			ZH: "展开的思考内容会发送给所选翻译服务，可能包含私有代码或对话细节。",
			EN: "Expanded reasoning is sent to the selected service. It may contain private code or conversation details.",
		},
		"translation": {
			ZH: "查看译文",
			EN: "View translation",
		},
		"original": {
			ZH: "查看原文",
			EN: "View original",
		},
		"retry": {
			ZH: "重试",
			EN: "Retry",
		},
		"failed": {
			ZH: "翻译暂不可用，已显示原文",
			EN: "Translation failed. Showing the original.",
		},
		"translating": {
			ZH: "正在翻译思考内容",
			EN: "Translating reasoning",
		},
		"invalidProvider": {
			ZH: "请选择翻译服务",
			EN: "Choose a translation service.",
		},
		"invalidLanguage": {
			ZH: "请填写 auto 或语言代码，例如 zh、en、ja",
			EN: "Enter auto or a language code, such as zh, en, or ja.",
		},
		"overridden": {
			ZH: "已覆盖",
			EN: "Overridden",
		},
		"reset": {
			ZH: "恢复默认",
			EN: "Reset to default",
		},
		"unavailable": {
			ZH: "翻译设置暂不可用",
			EN: "Translation preferences are unavailable.",
		},
		"readOnly": {
			ZH: "本部署的设置为只读",
			EN: "This deployment stores settings read-only.",
		},
		"saveFailed": {
			ZH: "本部署没有接受这些值，已保留供你修改",
			EN: "The deployment did not accept these values; they were left for you to correct.",
		},
		"save": {
			ZH: "保存",
			EN: "Save",
		},
		"saving": {
			ZH: "保存中…",
			EN: "Saving…",
		},
		"copy": {
			ZH: "复制",
			EN: "Copy",
		},
		"copied": {
			ZH: "已复制",
			EN: "Copied",
		},
		"codeTitle": {
			ZH: "代码块",
			EN: "Code block",
		},
		"wrap": {
			ZH: "自动换行",
			EN: "Wrap lines",
		},
		"unwrap": {
			ZH: "不换行",
			EN: "Unwrap lines",
		},
		"footnotes": {
			ZH: "脚注",
			EN: "Footnotes",
		},
	},
	"dsh-experimental-client-ui-voice-input": {
		"dictate": {
			ZH: "听写",
			EN: "Dictate",
		},
		"dismissFeedback": {
			ZH: "关闭提示",
			EN: "Dismiss notice",
		},
		"providerFeedback": {
			ZH: "{name}：{message}",
			EN: "{name}: {message}",
		},
		"saving": {
			ZH: "正在保存语音设置…",
			EN: "Saving voice preferences…",
		},
		"setupPrompt.title": {
			ZH: "使用语音输入前需要安装",
			EN: "Set up voice input before recording",
		},
		"setupPrompt.body": {
			ZH: "语音输入插件已开启。首次使用需要在本机下载并准备识别模型，请前往插件详情页查看空间、内存和时间说明，再开始安装。",
			EN: "Voice input is enabled. First use requires downloading and preparing local recognition models. Open the plugin details to review disk space, memory and time estimates before starting setup.",
		},
		"setupPrompt.later": {
			ZH: "稍后",
			EN: "Later",
		},
		"setupPrompt.open": {
			ZH: "前往安装",
			EN: "Go to setup",
		},
		"setupPrompt.trigger": {
			ZH: "打开语音输入引导",
			EN: "Open voice input setup",
		},
		"setupPrompt.details": {
			ZH: "前往语音插件设置",
			EN: "Open voice plugin settings",
		},
		"setupPrompt.unavailableTitle": {
			ZH: "语音识别尚未就绪",
			EN: "Speech recognition is not ready",
		},
		"setupPrompt.unavailableBody": {
			ZH: "请前往语音插件详情页查看识别服务状态、准备进度或错误信息",
			EN: "Open the voice plugin details to check recognition status, preparation progress, or errors.",
		},
		"setup.local": {
			ZH: "将在运行 DSH 的机器上下载本地模型，无需安装 Python 或编译工具。",
			EN: "Local models will be downloaded to the machine running DSH. No Python or compiler is required.",
		},
		"setup.disk": {
			ZH: "硬盘空间",
			EN: "Disk space",
		},
		"setup.diskValue": {
			ZH: "建议预留约 {gb} GB，包含模型、运行时和下载缓存",
			EN: "Allow about {gb} GB for models, runtime, and download caches",
		},
		"setup.memory": {
			ZH: "运行内存",
			EN: "Memory",
		},
		"setup.memoryValue": {
			ZH: "模型加载后约 {gb} GB，识别时可能更高",
			EN: "About {gb} GB with the model loaded; recognition may use more",
		},
		"setup.time": {
			ZH: "首次准备",
			EN: "First setup",
		},
		"setup.timeValue": {
			ZH: "参考 {min}–{max} 分钟，取决于网络和机器性能",
			EN: "Allow roughly {min}–{max} minutes, depending on the network and machine",
		},
		"setup.estimateNote": {
			ZH: "以上为估算；网络较慢时可能更久。已下载的资源会复用，默认在闲置后释放模型内存。",
			EN: "These are estimates; slow networks may take longer. Downloads are reused; idle models release memory by default.",
		},
		"step.check": {
			ZH: "检查本地资源",
			EN: "Check local resources",
		},
		"step.model": {
			ZH: "准备语音识别模型",
			EN: "Prepare recognition model",
		},
		"step.vad": {
			ZH: "准备语音检测模型",
			EN: "Prepare voice detection model",
		},
		"step.verify": {
			ZH: "校验模型文件",
			EN: "Verify model files",
		},
		"step.load": {
			ZH: "加载 {name}",
			EN: "Load {name}",
		},
		"stepStatus.pending": {
			ZH: "未开始",
			EN: "Not started",
		},
		"stepStatus.running": {
			ZH: "进行中",
			EN: "In progress",
		},
		"stepStatus.complete": {
			ZH: "已完成",
			EN: "Completed",
		},
		"stepStatus.failed": {
			ZH: "失败",
			EN: "Failed",
		},
		"stepStatus.cancelled": {
			ZH: "已取消",
			EN: "Cancelled",
		},
		"short.downloading": {
			ZH: "下载中",
			EN: "Downloading",
		},
		"short.failed": {
			ZH: "准备失败",
			EN: "Setup failed",
		},
		"retryRecording": {
			ZH: "重新录音",
			EN: "Record again",
		},
		"preparationSteps": {
			ZH: "准备步骤",
			EN: "Preparation steps",
		},
		"prepare": {
			ZH: "下载并准备",
			EN: "Download and prepare",
		},
		"retryPrepare": {
			ZH: "重试准备",
			EN: "Retry preparation",
		},
		"cancelPrepare": {
			ZH: "取消准备",
			EN: "Cancel preparation",
		},
		"downloadProgress": {
			ZH: "下载进度",
			EN: "Download progress",
		},
		"downloadBytes": {
			ZH: "下载：{completed} / {total} MB，{percent}%",
			EN: "Download: {completed} / {total} MB, {percent}%",
		},
		"downloadUnknown": {
			ZH: "已下载 {completed} MB",
			EN: "Downloaded {completed} MB",
		},
		"elapsed": {
			ZH: "已等待 {seconds} 秒",
			EN: "Waiting {seconds} seconds",
		},
		"preparationFailed": {
			ZH: "准备失败：{message}",
			EN: "Preparation failed: {message}",
		},
		"download.network": {
			ZH: "无法下载 {resource}：连接下载服务失败。",
			EN: "Cannot download {resource}: connection to the download service failed.",
		},
		"download.dns": {
			ZH: "无法下载 {resource}：无法解析下载地址。",
			EN: "Cannot download {resource}: the download address could not be resolved.",
		},
		"download.timeout": {
			ZH: "下载 {resource} 超时。",
			EN: "The download of {resource} timed out.",
		},
		"download.certificate": {
			ZH: "无法下载 {resource}：安全连接的证书校验失败。",
			EN: "Cannot download {resource}: certificate verification failed.",
		},
		"download.http": {
			ZH: "无法下载 {resource}：下载服务返回 HTTP {status}。",
			EN: "Cannot download {resource}: the download service returned HTTP {status}.",
		},
		"download.integrity": {
			ZH: "{resource} 下载不完整或文件校验失败。",
			EN: "The download of {resource} is incomplete or failed verification.",
		},
		"download.storage": {
			ZH: "无法保存 {resource}：磁盘空间不足或没有写入权限。",
			EN: "Cannot save {resource}: insufficient disk space or write permission.",
		},
		"download.unknown": {
			ZH: "准备 {resource} 失败。",
			EN: "Could not prepare {resource}.",
		},
		"downloadAdvice.network": {
			ZH: "请检查运行 DSH 的机器能否访问下载来源及其模型下载服务；如需代理，请在该机器上配置后重试。",
			EN: "Check that the machine running DSH can reach the download source and its model download services. Configure a proxy on that machine if needed, then retry.",
		},
		"downloadAdvice.dns": {
			ZH: "请检查运行 DSH 的机器的 DNS 和代理设置，确认可以解析下载来源的域名后重试。",
			EN: "Check DNS and proxy settings on the machine running DSH. Confirm that it can resolve the download source domain, then retry.",
		},
		"downloadAdvice.timeout": {
			ZH: "请检查运行 DSH 的机器的网络或代理连接，稍后重试。已下载并通过校验的文件会保留。",
			EN: "Check the network or proxy connection on the machine running DSH, then retry later. Verified downloaded files are retained.",
		},
		"downloadAdvice.certificate": {
			ZH: "请检查运行 DSH 的机器的系统时间、受信任证书和代理设置后重试。",
			EN: "Check the system clock, trusted certificates and proxy settings on the machine running DSH, then retry.",
		},
		"downloadAdvice.http": {
			ZH: "请确认下载地址可用、代理可正常连接下载服务，或稍后重试。",
			EN: "Confirm that the download address is available and the proxy can reach the download service, or retry later.",
		},
		"downloadAdvice.integrity": {
			ZH: "请重试下载。已完成并通过校验的其他文件会保留。",
			EN: "Retry the download. Other completed and verified files are retained.",
		},
		"downloadAdvice.storage": {
			ZH: "请检查运行 DSH 的机器的剩余磁盘空间，以及模型目录的写入权限后重试。",
			EN: "Check available disk space and write permission for the model directory on the machine running DSH, then retry.",
		},
		"downloadAdvice.unknown": {
			ZH: "请检查运行 DSH 的机器的网络、磁盘空间和模型目录权限后重试。",
			EN: "Check the network, disk space and model directory permissions on the machine running DSH, then retry.",
		},
		"downloadSource": {
			ZH: "下载来源：{source}",
			EN: "Download source: {source}",
		},
		"sourceChoice": {
			ZH: "模型下载源",
			EN: "Model download source",
		},
		"sourceAuto": {
			ZH: "自动选择（推荐）",
			EN: "Automatic (recommended)",
		},
		"sourceHuggingFace": {
			ZH: "Hugging Face",
			EN: "Hugging Face",
		},
		"sourceMirror": {
			ZH: "HF-Mirror（国内镜像）",
			EN: "HF-Mirror (China mirror)",
		},
		"sourceAutoHelp": {
			ZH: "优先使用响应较快的源，下载失败时自动尝试其他源。",
			EN: "Prefer the first responding source and try another if a download fails.",
		},
		"sourceManualHelp": {
			ZH: "仅从所选源下载；已下载并通过校验的文件会复用。",
			EN: "Download only from the selected source. Verified downloaded files are reused.",
		},
		"downloadCode": {
			ZH: "错误码：{code}",
			EN: "Error code: {code}",
		},
		"reconnecting": {
			ZH: "正在连接语音服务…",
			EN: "Connecting to speech recognition…",
		},
		"preparation.unprepared": {
			ZH: "首次使用需要下载本地识别模型。",
			EN: "First use downloads the local recognition models.",
		},
		"preparation.checking": {
			ZH: "正在检查本地资源…",
			EN: "Checking local resources…",
		},
		"preparation.loading": {
			ZH: "正在加载 {name}…",
			EN: "Loading {name}…",
		},
		"preparation.waking": {
			ZH: "正在唤醒本地语音…",
			EN: "Waking local speech recognition…",
		},
		"preparation.ready": {
			ZH: "本地语音已就绪。",
			EN: "Local speech recognition is ready.",
		},
		"preparation.standby": {
			ZH: "本地资源已准备，录音时自动唤醒。",
			EN: "Local resources are prepared. Recording wakes the recognizer.",
		},
		"preparation.cancelled": {
			ZH: "准备已取消，已完成的下载会保留。",
			EN: "Preparation cancelled. Completed downloads are retained.",
		},
		"preparation.cancelling": {
			ZH: "正在取消准备…",
			EN: "Cancelling preparation…",
		},
		"cloudReady": {
			ZH: "云端语音识别已就绪。",
			EN: "Cloud speech recognition is ready.",
		},
		"discard": {
			ZH: "丢弃识别文字",
			EN: "Discard transcript",
		},
		"transcribingShort": {
			ZH: "识别中…",
			EN: "Transcribing…",
		},
		"wakingShort": {
			ZH: "唤醒中…",
			EN: "Waking…",
		},
		"provider": {
			ZH: "识别服务",
			EN: "Recognition service",
		},
		"language": {
			ZH: "识别语言",
			EN: "Recognition language",
		},
		"inputDevice": {
			ZH: "输入设备",
			EN: "Input device",
		},
		"systemMicrophone": {
			ZH: "系统默认",
			EN: "System default",
		},
		"systemMicrophoneNamed": {
			ZH: "系统默认（{name}）",
			EN: "System default ({name})",
		},
		"unnamedSelectedMicrophone": {
			ZH: "麦克风",
			EN: "Microphone",
		},
		"unnamedMicrophone": {
			ZH: "麦克风 {number}",
			EN: "Microphone {number}",
		},
		"inputLevel": {
			ZH: "麦克风输入电平",
			EN: "Microphone input level",
		},
		"deviceUnavailable": {
			ZH: "不可用",
			EN: "Unavailable",
		},
		"deviceMissing": {
			ZH: "麦克风不可用，请选择其他输入设备",
			EN: "Microphone unavailable. Choose another input device.",
		},
		"auto": {
			ZH: "自动识别",
			EN: "Detect automatically",
		},
		"zh": {
			ZH: "中文",
			EN: "Chinese",
		},
		"en": {
			ZH: "英语",
			EN: "English",
		},
		"yue": {
			ZH: "粤语",
			EN: "Cantonese",
		},
		"ja": {
			ZH: "日语",
			EN: "Japanese",
		},
		"ko": {
			ZH: "韩语",
			EN: "Korean",
		},
		"start": {
			ZH: "开始录音",
			EN: "Start recording",
		},
		"stop": {
			ZH: "停止并识别",
			EN: "Stop and transcribe",
		},
		"cancel": {
			ZH: "取消",
			EN: "Cancel",
		},
		"insert": {
			ZH: "插入文字",
			EN: "Insert text",
		},
		"loading": {
			ZH: "正在读取识别服务…",
			EN: "Loading recognition services…",
		},
		"requesting": {
			ZH: "请允许使用麦克风…",
			EN: "Allow microphone access to continue…",
		},
		"recording": {
			ZH: "正在录音…",
			EN: "Recording…",
		},
		"interrupted": {
			ZH: "录音中断，请重试。",
			EN: "Recording was interrupted. Please try again.",
		},
		"local": {
			ZH: "音频在运行 DSH 的机器上识别。需要下载模型时，请确保该机器能访问所选下载源及其文件服务。如需代理，请在该机器上配置。",
			EN: "Audio is recognized on the machine running DSH. If models need downloading, that machine must be able to reach the selected source and its file services. Configure a proxy on that machine if needed.",
		},
		"cloud": {
			ZH: "音频将发送至所选云端服务。",
			EN: "Audio will be sent to the selected cloud service.",
		},
		"empty": {
			ZH: "未识别到语音",
			EN: "No speech recognized",
		},
		"cancelled": {
			ZH: "已取消语音输入。",
			EN: "Voice input cancelled.",
		},
		"conflict": {
			ZH: "草稿已被修改。识别文字已保留，可在当前光标位置插入。",
			EN: "Your draft changed. The transcript is preserved and can be inserted at the current cursor.",
		},
		"failed": {
			ZH: "语音识别失败：{message}",
			EN: "Speech recognition failed: {message}",
		},
		"unavailable": {
			ZH: "当前浏览器不支持录音，请使用支持麦克风的浏览器。",
			EN: "This browser cannot record audio. Use a browser with microphone support.",
		},
		"permission": {
			ZH: "麦克风权限未开启，请在浏览器和系统设置中允许访问。",
			EN: "Microphone access is disabled. Allow it in browser and system settings.",
		},
		"tooLarge": {
			ZH: "录音超过服务限制，请缩短录音后重试。",
			EN: "The recording exceeds the service limit. Try a shorter recording.",
		},
	},
	"dsh-client-ui-conversation": {
		"settings.enter.title": {
			ZH: "繁忙时的发送行为",
			EN: "Send behavior while busy",
		},
		"settings.enter.description": {
			ZH: "智能体运行时 Enter 键和发送按钮的行为；Cmd/Ctrl+Enter 使用另一行为",
			EN: "What Enter and the Send button do while the agent is running; Cmd/Ctrl+Enter uses the other behavior",
		},
		"settings.enter.queue": {
			ZH: "排队发送",
			EN: "Queue",
		},
		"settings.enter.steer": {
			ZH: "插话发送",
			EN: "Steer",
		},
		"settings.customInstructions.title": {
			ZH: "自定义指令",
			EN: "Custom instructions",
		},
		"settings.customInstructions.description": {
			ZH: "给智能体定几条规则，后续对所有任务都生效",
			EN: "Set a few standing rules for the agent; they apply to every later task",
		},
		"settings.customInstructions.placeholder": {
			ZH: "输入规则，例如：回复始终使用中文",
			EN: "Write your rules, e.g. always reply in Chinese",
		},
		"settings.customInstructions.counter": {
			ZH: "{count} / {max}",
			EN: "{count} / {max}",
		},
		"settings.beam.title": {
			ZH: "发送消息时的思考炫光",
			EN: "Thinking glow when sending",
		},
		"settings.beam.description": {
			ZH: "智能体发送或思考时，在输入条周围播放彩色边光。",
			EN: "Play a colored border beam on the composer while a turn is sending or thinking.",
		},
		"settings.beam.configure": {
			ZH: "设置思考炫光",
			EN: "Configure thinking glow",
		},
		"settings.beam.panel.title": {
			ZH: "思考炫光",
			EN: "Thinking glow",
		},
		"settings.beam.panel.description": {
			ZH: "调整运行态边光；保存后立即用于所有会话。",
			EN: "Tune the running border beam; saved changes apply to every conversation.",
		},
		"settings.beam.panel.close": {
			ZH: "关闭思考炫光设置",
			EN: "Close thinking glow settings",
		},
		"settings.beam.panel.preview": {
			ZH: "思考炫光预览",
			EN: "Thinking glow preview",
		},
		"settings.beam.panel.previewStatus": {
			ZH: "智能体正在思考",
			EN: "Agent is thinking",
		},
		"settings.beam.panel.direction": {
			ZH: "流动方向",
			EN: "Direction",
		},
		"settings.beam.panel.clockwise": {
			ZH: "顺时针",
			EN: "Clockwise",
		},
		"settings.beam.panel.counterclockwise": {
			ZH: "逆时针",
			EN: "Counterclockwise",
		},
		"settings.beam.panel.pingPong": {
			ZH: "往返",
			EN: "Ping-pong",
		},
		"settings.beam.panel.period": {
			ZH: "单圈周期",
			EN: "Rotation period",
		},
		"settings.beam.panel.motion": {
			ZH: "运动",
			EN: "Motion",
		},
		"settings.beam.panel.mode": {
			ZH: "模式",
			EN: "Mode",
		},
		"settings.beam.panel.mode.legacy": {
			ZH: "Legacy",
			EN: "Legacy",
		},
		"settings.beam.panel.mode.lounge": {
			ZH: "Lounge",
			EN: "Lounge",
		},
		"settings.beam.panel.mode.aurora": {
			ZH: "Aurora",
			EN: "Aurora",
		},
		"settings.beam.panel.mode.reactive": {
			ZH: "Reactive",
			EN: "Reactive",
		},
		"settings.beam.panel.mode.custom": {
			ZH: "自定义",
			EN: "Custom",
		},
		"settings.beam.panel.easing": {
			ZH: "缓动",
			EN: "Easing",
		},
		"settings.beam.panel.easing.linear": {
			ZH: "线性",
			EN: "Linear",
		},
		"settings.beam.panel.easing.ease": {
			ZH: "缓入缓出",
			EN: "Ease",
		},
		"settings.beam.panel.easing.ease-in-out": {
			ZH: "平滑缓入缓出",
			EN: "Ease in/out",
		},
		"settings.beam.panel.easing.cubic-bezier": {
			ZH: "平滑曲线",
			EN: "Smooth curve",
		},
		"settings.beam.panel.appearance": {
			ZH: "外观",
			EN: "Appearance",
		},
		"settings.beam.panel.intensity": {
			ZH: "整体强度",
			EN: "Overall intensity",
		},
		"settings.beam.panel.bloom": {
			ZH: "光晕强度",
			EN: "Bloom intensity",
		},
		"settings.beam.panel.hue": {
			ZH: "色相偏移",
			EN: "Hue offset",
		},
		"settings.beam.panel.trackWidth": {
			ZH: "轨道宽度",
			EN: "Track width",
		},
		"settings.beam.panel.glowBlur": {
			ZH: "光晕模糊",
			EN: "Glow blur",
		},
		"settings.beam.panel.palette": {
			ZH: "色板",
			EN: "Palette",
		},
		"settings.beam.panel.palette.legacy": {
			ZH: "Legacy",
			EN: "Legacy",
		},
		"settings.beam.panel.palette.warm": {
			ZH: "暖色",
			EN: "Warm",
		},
		"settings.beam.panel.palette.aurora": {
			ZH: "极光",
			EN: "Aurora",
		},
		"settings.beam.panel.palette.rose": {
			ZH: "玫瑰",
			EN: "Rose",
		},
		"settings.beam.panel.palette.glacier": {
			ZH: "冰川",
			EN: "Glacier",
		},
		"settings.beam.panel.palette.cyber": {
			ZH: "赛博",
			EN: "Cyber",
		},
		"settings.beam.panel.palette.sunset": {
			ZH: "日落",
			EN: "Sunset",
		},
		"settings.beam.panel.palette.cyan": {
			ZH: "青色",
			EN: "Cyan",
		},
		"settings.beam.panel.color": {
			ZH: "颜色 {value}",
			EN: "Color {value}",
		},
		"settings.beam.panel.removeColor": {
			ZH: "移除颜色 {value}",
			EN: "Remove color {value}",
		},
		"settings.beam.panel.addColor": {
			ZH: "添加颜色",
			EN: "Add color",
		},
		"settings.beam.panel.effects": {
			ZH: "效果",
			EN: "Effects",
		},
		"settings.beam.panel.breathing": {
			ZH: "呼吸亮度",
			EN: "Brightness breathing",
		},
		"settings.beam.panel.breathingAmplitude": {
			ZH: "呼吸幅度",
			EN: "Breathing amplitude",
		},
		"settings.beam.panel.breathingPeriod": {
			ZH: "呼吸周期",
			EN: "Breathing period",
		},
		"settings.beam.panel.hueCycle": {
			ZH: "色相循环",
			EN: "Hue cycling",
		},
		"settings.beam.panel.hueCycleRange": {
			ZH: "色相范围",
			EN: "Hue range",
		},
		"settings.beam.panel.hueCyclePeriod": {
			ZH: "色相周期",
			EN: "Hue cycle period",
		},
		"settings.beam.panel.night": {
			ZH: "夜间调暗",
			EN: "Night dimming",
		},
		"settings.beam.panel.nightEnabled": {
			ZH: "启用夜间时段",
			EN: "Enable night window",
		},
		"settings.beam.panel.nightFrom": {
			ZH: "开始时间",
			EN: "Start time",
		},
		"settings.beam.panel.nightTo": {
			ZH: "结束时间",
			EN: "End time",
		},
		"settings.beam.panel.nightDim": {
			ZH: "调暗幅度",
			EN: "Dim amount",
		},
		"settings.beam.panel.presets": {
			ZH: "用户预设",
			EN: "User presets",
		},
		"settings.beam.panel.presetSelect": {
			ZH: "选择预设",
			EN: "Select preset",
		},
		"settings.beam.panel.deletePreset": {
			ZH: "删除预设",
			EN: "Delete preset",
		},
		"settings.beam.panel.presetName": {
			ZH: "预设名称",
			EN: "Preset name",
		},
		"settings.beam.panel.savePreset": {
			ZH: "保存预设",
			EN: "Save preset",
		},
		"settings.beam.panel.exchange": {
			ZH: "配置交换",
			EN: "Configuration exchange",
		},
		"settings.beam.panel.exportJson": {
			ZH: "导出 JSON",
			EN: "Export JSON",
		},
		"settings.beam.panel.importJson": {
			ZH: "导入 JSON",
			EN: "Import JSON",
		},
		"settings.beam.panel.jsonLimit": {
			ZH: "最大 {value} KiB；导入后仍需点击保存",
			EN: "Maximum {value} KiB; click Save after importing",
		},
		"settings.beam.panel.pixels": {
			ZH: "{value} px",
			EN: "{value} px",
		},
		"settings.beam.panel.presetNameRequired": {
			ZH: "请输入预设名称",
			EN: "Enter a preset name",
		},
		"settings.beam.panel.presetLimit": {
			ZH: "最多保存 5 个预设",
			EN: "Up to five presets can be saved",
		},
		"settings.beam.panel.clipboardWriteFailed": {
			ZH: "无法写入剪贴板",
			EN: "Could not write to the clipboard",
		},
		"settings.beam.panel.clipboardReadFailed": {
			ZH: "剪贴板 JSON 无效或不可读取",
			EN: "Clipboard JSON is invalid or unavailable",
		},
		"settings.beam.panel.saveFailed": {
			ZH: "保存失败，请重试",
			EN: "Save failed; try again",
		},
		"settings.beam.panel.seconds": {
			ZH: "{value} 秒",
			EN: "{value} s",
		},
		"settings.beam.panel.degrees": {
			ZH: "{value}°",
			EN: "{value}°",
		},
		"settings.beam.panel.reset": {
			ZH: "恢复默认",
			EN: "Reset",
		},
		"settings.beam.panel.cancel": {
			ZH: "取消",
			EN: "Cancel",
		},
		"settings.beam.panel.save": {
			ZH: "保存",
			EN: "Save",
		},
		"settings.typingFx.title": {
			ZH: "输入特效",
			EN: "Typing effects",
		},
		"settings.typingFx.description": {
			ZH: "键入字符时在输入框播放短暂回显，可替换光标样式。",
			EN: "Echo typed characters in the composer, with an optional custom caret.",
		},
		"settings.typingFx.configure": {
			ZH: "设置输入特效",
			EN: "Configure typing effects",
		},
		"settings.typingFx.panel.title": {
			ZH: "输入特效",
			EN: "Typing effects",
		},
		"settings.typingFx.panel.description": {
			ZH: "调整键入回显与光标样式；保存后立即用于所有会话。",
			EN: "Tune the typing echo and caret style; saving applies to every session at once.",
		},
		"settings.typingFx.panel.close": {
			ZH: "关闭输入特效设置",
			EN: "Close typing effect settings",
		},
		"settings.typingFx.panel.preview": {
			ZH: "输入特效预览",
			EN: "Typing effect preview",
		},
		"settings.typingFx.panel.previewText": {
			ZH: "输入特效",
			EN: "Typing effects",
		},
		"settings.typingFx.panel.effect": {
			ZH: "效果",
			EN: "Effect",
		},
		"settings.typingFx.panel.effect.drop": {
			ZH: "落锤",
			EN: "Drop",
		},
		"settings.typingFx.panel.effect.rise": {
			ZH: "上浮",
			EN: "Rise",
		},
		"settings.typingFx.panel.effect.flash": {
			ZH: "闪现",
			EN: "Flash",
		},
		"settings.typingFx.panel.cursor": {
			ZH: "光标",
			EN: "Caret",
		},
		"settings.typingFx.panel.cursor.native": {
			ZH: "跟随系统",
			EN: "System caret",
		},
		"settings.typingFx.panel.cursor.block": {
			ZH: "方块",
			EN: "Block",
		},
		"settings.typingFx.panel.cursor.underline": {
			ZH: "下划线",
			EN: "Underline",
		},
		"settings.typingFx.panel.blink": {
			ZH: "光标闪烁",
			EN: "Caret blink",
		},
		"settings.typingFx.panel.colors": {
			ZH: "颜色",
			EN: "Colors",
		},
		"settings.typingFx.panel.colors.theme": {
			ZH: "跟随主题",
			EN: "Follow theme",
		},
		"settings.typingFx.panel.colors.custom": {
			ZH: "自定义",
			EN: "Custom",
		},
		"settings.typingFx.panel.colors.echo": {
			ZH: "回显颜色",
			EN: "Echo color",
		},
		"settings.typingFx.panel.colors.caret": {
			ZH: "光标颜色",
			EN: "Caret color",
		},
		"settings.typingFx.panel.colors.text": {
			ZH: "文本颜色",
			EN: "Text color",
		},
		"settings.typingFx.panel.colors.ocean": {
			ZH: "海洋",
			EN: "Ocean",
		},
		"settings.typingFx.panel.colors.sunset": {
			ZH: "落日",
			EN: "Sunset",
		},
		"settings.typingFx.panel.colors.candy": {
			ZH: "糖果",
			EN: "Candy",
		},
		"settings.typingFx.panel.colors.forest": {
			ZH: "森林",
			EN: "Forest",
		},
		"settings.typingFx.panel.colors.violet": {
			ZH: "紫藤",
			EN: "Violet",
		},
		"settings.typingFx.panel.colors.ember": {
			ZH: "熔金",
			EN: "Ember",
		},
		"settings.typingFx.panel.speed": {
			ZH: "回显速度",
			EN: "Echo speed",
		},
		"settings.typingFx.panel.presets": {
			ZH: "用户预设",
			EN: "User presets",
		},
		"settings.typingFx.panel.presetSelect": {
			ZH: "选择预设",
			EN: "Select a preset",
		},
		"settings.typingFx.panel.deletePreset": {
			ZH: "删除预设",
			EN: "Delete preset",
		},
		"settings.typingFx.panel.presetName": {
			ZH: "预设名称",
			EN: "Preset name",
		},
		"settings.typingFx.panel.savePreset": {
			ZH: "保存预设",
			EN: "Save preset",
		},
		"settings.typingFx.panel.exchange": {
			ZH: "配置交换",
			EN: "Configuration exchange",
		},
		"settings.typingFx.panel.exportJson": {
			ZH: "导出 JSON",
			EN: "Export JSON",
		},
		"settings.typingFx.panel.importJson": {
			ZH: "导入 JSON",
			EN: "Import JSON",
		},
		"settings.typingFx.panel.jsonLimit": {
			ZH: "最大 {value} KiB；导入后仍需点击保存",
			EN: "Maximum {value} KiB; click Save after importing",
		},
		"settings.typingFx.panel.presetNameRequired": {
			ZH: "请输入预设名称",
			EN: "Enter a preset name",
		},
		"settings.typingFx.panel.presetLimit": {
			ZH: "最多保存 5 个预设",
			EN: "Up to five presets can be saved",
		},
		"settings.typingFx.panel.clipboardWriteFailed": {
			ZH: "无法写入剪贴板",
			EN: "Could not write to the clipboard",
		},
		"settings.typingFx.panel.clipboardReadFailed": {
			ZH: "剪贴板 JSON 无效或不可读取",
			EN: "Clipboard JSON is invalid or unavailable",
		},
		"settings.typingFx.panel.saveFailed": {
			ZH: "保存失败，请重试",
			EN: "Save failed; try again",
		},
		"settings.typingFx.panel.reset": {
			ZH: "恢复默认",
			EN: "Reset",
		},
		"settings.typingFx.panel.cancel": {
			ZH: "取消",
			EN: "Cancel",
		},
		"settings.typingFx.panel.save": {
			ZH: "保存",
			EN: "Save",
		},
		"settings.resize.title": {
			ZH: "拖动调整输入框大小",
			EN: "Resize input by dragging",
		},
		"settings.resize.description": {
			ZH: "允许拖动输入区上、左、右边框改变高度和宽度。关闭后高度仍按草稿自动增高，宽度回到栏宽。",
			EN: "Drag the top, left, or right edge of the composer to change its height and width. Turning this off keeps auto-growing height with the draft and restores column width.",
		},
		"settings.statsLine.title": {
			ZH: "会话统计",
			EN: "Session stats",
		},
		"settings.statsLine.description": {
			ZH: "在输入框上方显示轮次、耗时、缓存与 token。关闭后隐藏数字，保留这一行间隔。",
			EN: "Show turn, duration, cache, and token figures above the composer. Turning this off hides the figures and keeps the row gap.",
		},
		"settings.peakValley.title": {
			ZH: "官方峰谷时",
			EN: "Official peak/valley hours",
		},
		"settings.peakValley.description": {
			ZH: "会话累计费用开启时，在会话统计下方显示官方峰谷时段状态条。开启后始终显示；关闭时仅在检测到 DeepSeek API 路线时显示。",
			EN: "When session cost is on, show the official peak/valley status row under the session stats. On keeps it always visible; off keeps detection-only (a DeepSeek API route).",
		},
		"settings.sessionCost.title": {
			ZH: "会话累计费用",
			EN: "Session cost",
		},
		"settings.sessionCost.description": {
			ZH: "在输入框下方显示峰谷状态条与当前会话累计费用。",
			EN: "Show the peak/valley status row and the running session cost under the composer.",
		},
		"settings.viewTabs.title": {
			ZH: "对话与轨迹页签",
			EN: "Chat and Trajectory tabs",
		},
		"settings.viewTabs.description": {
			ZH: "在会话顶栏显示「对话」「轨迹」切换。关闭后仍停留在当前视图。",
			EN: "Show the Chat and Trajectory switcher in the session header. Turning this off keeps the current view.",
		},
		"settings.nextPrompt.label": {
			ZH: "下一句提示",
			EN: "Next-prompt suggestions",
		},
		"settings.nextPrompt.hint": {
			ZH: "回复结束后建议下一句，可按 Tab 采纳，不会自动发送。生成提示会额外调用当前模型；关闭后不再调用。",
			EN: "Suggest a follow-up after each reply. Press Tab to accept without sending. Generating a suggestion makes an additional call to the current model; turning this off stops those calls.",
		},
	},
	"dsh-client-ui-titlebar": {
		"settings.terminalToggle.title": {
			ZH: "终端抽屉开关",
			EN: "Terminal drawer toggle",
		},
		"settings.terminalToggle.description": {
			ZH: "在标题栏显示终端抽屉按钮。关闭后仍可使用 Ctrl+` 切换。",
			EN: "Show the terminal drawer button in the titlebar. Turning this off still leaves Ctrl+` available.",
		},
		"settings.surfacesToggle.title": {
			ZH: "右侧栏按钮",
			EN: "Right Sidebar button",
		},
		"settings.surfacesToggle.description": {
			ZH: "在标题栏显示右侧栏按钮。关闭后仍可使用 Ctrl+\\ 开合同一条右栏。",
			EN: "Show the right Sidebar button in the titlebar. Turning this off still leaves Ctrl+\\ for the same Sidebar.",
		},
	},
}

func settingsVendorLocale(module, key string) string {
	cleanMod := strings.TrimPrefix(strings.TrimSpace(module), "@deepseek-ai/")
	mod, ok := settingsVendorLocales[cleanMod]
	if !ok {
		return ""
	}
	entry, ok := mod[key]
	if !ok {
		return ""
	}
	if i18n.Locale() == i18n.LocaleEN {
		return entry.EN
	}
	return entry.ZH
}

func settingsVendorFormat(module, key string, vars map[string]string) string {
	text := settingsVendorLocale(module, key)
	if text == "" {
		return ""
	}
	for k, v := range vars {
		text = strings.ReplaceAll(text, "{"+k+"}", v)
	}
	return text
}

func normalizeSettingsVendorNotice(module, raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw
	}
	if text := settingsVendorLocale(module, trimmed); text != "" {
		return text
	}
	for modName := range settingsVendorLocales {
		if text := settingsVendorLocale(modName, trimmed); text != "" {
			return text
		}
	}
	return raw
}
