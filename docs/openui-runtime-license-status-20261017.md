# OpenUI 运行包许可状态（2026-10-17）

## 当前结论

现有证据尚不足以确认恢复出的完整 viewer 可再分发。本项目的适配代码与第三方运行包分别处理；当前没有把该 viewer 加入源码或安装包。

核对的 viewer SHA256：`ee5470ac1f67c50e69e554f6be5d7202625eef572b1f272f2276a1e250876961`。

上游 `open-intelligent-ui` 当前 main 与原版固定版本相同，均为 `6a4a0ba192ea285b3a5ed39dcc1295d7d4080d6f`。该版本完整文件树没有 LICENSE、COPYING、NOTICE 文件，`package.json` 的私有包 0.1.1 也没有 license 字段。README 有构建、修改和分发的描述，但没有明确许可条款；这里记录证据，不对这些描述作法律结论。

- [固定版本文件树](https://api.github.com/repos/thesysdev/open-intelligent-ui/git/trees/6a4a0ba192ea285b3a5ed39dcc1295d7d4080d6f?recursive=1)
- [package.json](https://github.com/thesysdev/open-intelligent-ui/blob/6a4a0ba192ea285b3a5ed39dcc1295d7d4080d6f/package.json)
- [README](https://github.com/thesysdev/open-intelligent-ui/blob/6a4a0ba192ea285b3a5ed39dcc1295d7d4080d6f/README.md)
- [许可请求 #44](https://github.com/thesysdev/open-intelligent-ui/issues/44)：核对时尚未获得回复；第三方提出的问题不构成权利人的授权。

## 已明确的 SDK 许可

官方源码发布版本中的包声明与 LICENSE 均为 MIT：

| 包 | 版本 | 源码版本及许可 |
|---|---|---|
| @openuidev/lang-core | 1.0.0 | [f7109dd0 的 LICENSE](https://github.com/thesysdev/openui/blob/f7109dd01b7bdfb14c8c9006eaca5a8b3cfc3a20/LICENSE) |
| @openuidev/react-lang | 1.0.0 | [f7109dd0 的 LICENSE](https://github.com/thesysdev/openui/blob/f7109dd01b7bdfb14c8c9006eaca5a8b3cfc3a20/LICENSE) |
| @openuidev/react-ui | 0.18.1 | [eb20708b 的 LICENSE](https://github.com/thesysdev/openui/blob/eb20708b9cc41afced5625e99e6ce8ec2f8a5435/LICENSE) |
| @openuidev/cli | 0.4.1 | [c8932053 的 LICENSE](https://github.com/thesysdev/openui/blob/c893205377d486c7bb951aab5d5de7f6a783d0a2/LICENSE) |

这些许可不能直接证明上层应用自有组件、函数、样式、媒体和整个派生 bundle 的许可完整。本次核对的是官方源码发布版本，没有重新下载核对 npm tarball 的全部内容，也没有把三个相邻 SDK notice 当作完整依赖声明。

## 解除独立分发阻塞的路径

1. 获得覆盖固定上游版本及派生运行包分发的明确授权，并核对完整依赖和媒体声明。
2. 或基于已明确许可的 SDK 自行实现所需运行壳与组件，保留相应许可声明；这是一项尚未完成的替代实现，不能直接搬入未明确许可的应用源码。

现有本地运行资源读取功能不表示本项目已经核实其所有使用或再分发权利。后续若上游为其他版本增加许可，需要核对其适用范围，不能自动推及本次恢复的运行包。

## 当前实施

按源码级接入推进已许可 SDK 与独立编写的兼容组件、状态和资源桥接，目标随 WhaleIsle-MyGo 构建并嵌入，不依赖原版安装或用户提供 viewer。该替代实现正在整合，尚未作为完成的运行时发布。
