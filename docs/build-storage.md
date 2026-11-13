# 验收结束后，不再留一整套临时环境

普通编译继续使用原来的入口：

```powershell
. .\scripts\env.ps1
go build ./cmd/whaleisle ./cmd/whaleisle-preview
```

`env.ps1` 保留 Go 1.27.2 与 CGO 配置，不再强制把 GOPATH、模块缓存、编译缓存和临时目录放进 `.toolchain`。默认复用 Go 已有的用户级缓存，不另建一套 DevCache。用 `go env GOCACHE GOMODCACHE` 查看实际位置；调用方显式设置的路径仍优先。旧终端若已加载过项目私有缓存变量，重新打开终端即可使用新默认值，现有工具链文件不会被移动或删除。

## 临时 runtime、下载工具和测试数据交给命令拥有

```powershell
.\scripts\with-build-workspace.ps1 {
    param($work)
    # 只在这里创建可丢弃的验收副本；不要把源码或唯一恢复资料移进来。
    go test ./internal/store/...
}
```

包装器在系统 TEMP 下新建唯一目录，将本次命令的 TEMP、TMP、GOTMPDIR 指向它。命令正常结束或报错后回收该目录，并恢复调用方的临时目录变量。日志和最终交付物应显式写到 `$work` 外。编译/模块缓存不随临时目录清理。

需要运行时副本时，把原有脚本的 `-RuntimeDestination` 指向 `$work` 下的路径；只读 runtime 可以直接复用。包装器用于**同步完成的验收命令**，不要在其中启动需要长期运行的后台应用。若仍有进程引用临时目录，包装器会保留它并显示路径，不停止进程。

特殊的冷缓存验收可在命令块中显式设置缓存到 `$work` 下。该命令结束后这份缓存也会回收；GOPATH、GOCACHE、GOMODCACHE、npm_config_cache 会恢复调用前的值，避免后续命令重建已删除的私有目录。不要为每轮普通验收新建缓存。

## 安装包保留，打包副本按需回收

```powershell
.\scripts\package.ps1 -OutputRoot .\dist\release-candidate -CleanStage
```

成功生成安装包、blockmap 和 SHA256 清单后，只移除这次新建的 `whaleisle` payload，保留交付文件。链接只解除，不跟随到外部源码或依赖。失败时保留 staging 供定位；缺少 NSIS 时也保留，并返回明确错误。

原有 `package.ps1` 不带新参数时仍保留 staging；`-StageOnly` 和 NSIS smoke 继续按原合同使用输出。现有目录不会被覆盖或当作本轮产物删除。

Host 的固定版本 pnpm 仍安装在选定工作目录，npm 下载内容改为复用配置的共享缓存。用户显式提供 `npm_config_cache` 时仍按该路径运行。
