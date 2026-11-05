# 全量功能审查工作口径

## 当前补审：以原版用户动作补齐分母

用户在四条关键运行失败后明确指出缺失仍被低估。前轮 367 项是粗粒度台账，不是全量原版用户能力清单。当前从实际安装 0.3.13 的入口、菜单/快捷动作、字段交互和随包插件贡献重新枚举；每项必须映射 MyGo 的可达入口与最终消费，不能只用同名回调、组件、Host RPC 或资源文件证明功能存在。

各补审 owner 写指定 `supplement-*.json`，不改产品、不操作桌面、不读取用户 profile/凭据、不发外部请求。已知故障及有效旧运行证据不重跑；新增结论要有具体反证。报告要同时保留已覆盖映射、独立遗漏、旧混合项需要的状态纠正和真实运行缺口。主负责人整合并修改旧错误结论，动作数量不得宣传为独立根因数。下文保留前轮基线和审查格式。

补审交付为 [ACTIONS.md](ACTIONS.md) 的 758 个动作/动作组及九份原始补充 JSON；33 个旧粗项状态已回写纠正。79 个 client 包均有直接动作引用或明确条件/支持层处置。动态任意第三方实现、外部协议/账户/设备组合和可见窗口仍保留各自验收缺口；本次没有新建产品候选或重做已有运行检查。

用户要求对照原版鲸屿做全量功能缺失及可用性审查。当前阶段只审查和记录，不修产品、不新建测试或 QA 夹具、不构建安装、不操作桌面、不停止/重启用户进程、不发模型请求、不触发 CI、不推送。各领域只写自己指定的审查 JSON；主报告和覆盖清单由主代理维护。不得再派子代理。

## 固定基线

- MyGo：`C:/AI/WhaleIsle-MyGo`，HEAD `b8b93ed8d64a4d1e79b80a423dd22f7fcb289eeb` 加当前工作树（轨迹 phase 修复及 opt-in 日志等）。原交付 rc.3 的 app 源码为 `5549e627`，不要混作相同版本。
- 用户当前诊断程序：stage 中 `whaleisle-trajectory-diagnostic.exe`，`0.2.1-rc.3+trajectory.1`，保留运行，不操作它。当前 profile 含真实用户内容和凭据，领域审查不得读取该 profile。主代理仅消费固定字段诊断日志。
- 已安装原版生产基线：`C:/软件/Whale Isle/resources/`，本轮直接读取实际 app.asar 确认桌面版本已为 **0.3.13**、Host/client package 0.2.1-rc.2。当前桌面源码已只读提取到 `tmp/full-audit-20261102/installed-source/src`（224 文件，2,634,930 字节，未运行应用）；清单为同目录上级 `installed-source-manifest.json`。可直接读安装目录 `vendor/deepseek-harness/packages/client/*/lib`。旧 `tmp/electron-parity-fixture-20261027/installed-source/src` 仅为历史 0.3.10 材料，不作当前基线，不计算普通文件 hash。
- 原项目当前源码：`C:/AI/Deepseek-Harness-Desktop`，HEAD `9ac095515f506ef719478f6a0d1bdb2325aeb7e8`，分支 `fix/project-file-links-open-in-panel`，桌面版本 0.3.13，44 条 dirty 变更。**全程只读，不在原项目构建、装依赖或运行应用。** 与实际安装版的额外 dirty 能力分开记录，不因版本号一致假定内容完全相同。
- `reference/electron` 是 2026-10-09 / 0.3.3 旧快照，只作补充，不作当前原版基线。
- 旧 `docs/status.md`、25 项 feature-matrix 和测试名称是待审声明/线索，不是通过凭证。

## 审查方法

从原版用户能力清单与本领域全部 MyGo 模块双向枚举，不能只查已知几个 bug 或 grep TODO。每个可见控件/重要动作沿真实 production 装配追踪：入口 → 回调 → 状态/路由 → Host/桌面动作 → 持久化 → 成功/失败/取消反馈。区分目录、工作区身份、会话身份和分组身份；不能因同名字段当语义相同。检视菜单、快捷键、空/加载/错误/断线状态、窄窗入口、动态插件及重启恢复。

不执行有真实外部副作用的动作。既有测试可阅读；只在能决定一个具体结论且不会操作桌面/用户数据时运行既有定向测试，禁止全套机械重跑。代码链完整只能记 wired-unverified；历史测试只证明当时范围。用户报告的失败是有效失败证据，不能被未覆盖该环境的绿测覆盖。没有复现根因时明确 unknown。

## 唯一领域输出

`docs/parity/full-audit-20261102/<domain>.json`，UTF-8，有效 JSON：

```
{
  "domain": "...",
  "modulesReviewed": ["internal/ui/...", "cmd/whaleisle/xxx.go"],
  "originalAreasReviewed": ["absolute/or/relative/path"],
  "features": [{
    "id": "PREFIX-001",
    "feature": "一个可独立使用/验收的动作或紧密能力",
    "baseline": "installed-0.3.13 | current-original-0.3.13 | existing-MyGo-contract",
    "expected": "原版用户可完成什么，保持关键数据含义",
    "entry": "用户入口及生产装配位置",
    "status": "confirmed-defect | missing | partial | wired-unverified | verified-scoped | external-blocked",
    "priority": "P0 | P1 | P2 | P3",
    "originalEvidence": ["path:line-range: what it proves"],
    "mygoEvidence": ["path:line-range: what it proves"],
    "finding": "具体观察/代码结论，不作超范围推断",
    "evidenceLevel": "user-report | current-runtime-log | code | scoped-test | historical-evidence | external-blocked",
    "requiredAction": "具体修复或验收动作；无需修复时说明缺的验收",
    "acceptanceGap": "尚不能证明的用户路径或边界"
  }],
  "crossDomainDependencies": ["other-domain: interface/question"],
  "uncovered": ["实际仍未审完的区域，禁止隐去"],
  "checksRun": [{"command":"...","result":"...","scope":"..."}],
  "summary": "领域结论和最高风险"
}
```

P0=基本路径无法完成/数据风险，P1=重大功能缺失/错误或发布阻断，P2=局部/进阶差异，P3=轻微差异。风险优先级与状态独立，不把 unverified 当 confirmed defect，也不把代码存在当可用。证据须可定位到行；无源代码证据的根因保持未知。不要为了凑数量把一个问题拆成数十个重复项，也不要把整个设置/插件面压成一条“齐全”。

完成条件：领域内所有用户能力和模块有去向；关键发现有能支持该结论的原版及 MyGo 引用；明确未完成审查和真实验收缺口。最终回报最高风险、输出路径及覆盖范围，不修产品、不提交代码。
