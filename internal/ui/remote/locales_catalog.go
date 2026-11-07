package remote

import "whaleisle/internal/ui/i18n"

type remoteLocaleEntry struct {
	module string
	key    string
	zh     string
	en     string
}

var remoteLocaleCatalog = []remoteLocaleEntry{
	{module: `dsh-client-ui-settings-remote`, key: `intro`, zh: `手机在侧栏扫码配对：Android App 内扫码＝链接设备，相机 / 浏览器扫码＝打开 web 端。此处只改连接方式与中继。`, en: `Pair from the sidebar: scanning inside the Android app links the device; a camera / browser scan opens the web client. Set LAN / Server and the relay here.`},
	{module: `dsh-client-ui-settings-remote`, key: `tabs`, zh: `远程分区标签`, en: `Remote section tabs`},
	{module: `dsh-client-ui-settings-remote`, key: `empty`, zh: `还没有可显示的标签页。`, en: `No tabs to show yet.`},
	{module: `dsh-client-ui-settings-remote`, key: `enable`, zh: `开启配对`, en: `Start pairing`},
	{module: `dsh-client-ui-settings-remote`, key: `modeHint`, zh: `局域网：扫码打开本机 :3180 网页。服务器：扫码打开公网网页；数据走中继。不要用微信转发链接（会丢掉 #offer=）。`, en: `LAN: scan to open this machine's :3180 page. Server: scan opens the public web page; traffic uses the relay. Do not forward links in WeChat — it strips #offer=.`},
	{module: `dsh-client-ui-settings-remote`, key: `relayNeedsBoth`, zh: `请先填写并保存自建中继主机（host:443），再选择服务器。`, en: `Save your self-hosted relay host (host:443) before choosing Server.`},
	{module: `dsh-client-ui-settings-remote`, key: `relayNeedsUrl`, zh: `请填写并保存自建中继主机后再选择服务器。`, en: `Save your self-hosted relay host before choosing Server.`},
	{module: `dsh-client-ui-settings-remote`, key: `relayNeedsToken`, zh: `请填写并保存自建中继主机后再选择服务器。`, en: `Save your self-hosted relay host before choosing Server.`},
	{module: `dsh-client-ui-settings-remote`, key: `relayNeedsEndpoint`, zh: `桌面已内置默认中继，可直接选择服务器。`, en: `Built-in relay is ready — Server works out of the box.`},
	{module: `dsh-client-ui-settings-remote`, key: `lanPlaintextWarning`, zh: `局域网直连仅限可信网络。`, en: `LAN direct connect is for trusted networks only.`},
	{module: `dsh-client-ui-settings-remote`, key: `lanTlsHint`, zh: `手机会话使用端到端加密，中继 TLS 由中继主机配置决定；局域网配对页使用 HTTP，只提供应用文件。`, en: `Phone sessions use end-to-end encryption. Relay TLS follows the relay host configuration; the LAN pairing page uses HTTP and only serves app files.`},
	{module: `dsh-client-ui-settings-remote`, key: `portInvalid`, zh: `端口必须是 1024 到 65535 之间的整数。`, en: `Port must be an integer from 1024 to 65535.`},
	{module: `dsh-client-ui-settings-remote`, key: `relayToken`, zh: `（已退役）宿主令牌`, en: `(Retired) host token`},
	{module: `dsh-client-ui-settings-remote`, key: `relayTokenHint`, zh: `dshd 配对不使用宿主令牌；设备密钥在扫码后自动绑定，直至解除。`, en: `dshd pairing does not use a host token; device secrets bind after scan until you unbind.`},
	{module: `dsh-client-ui-settings-remote`, key: `relayTokenSet`, zh: `中继已配置`, en: `Relay configured`},
	{module: `dsh-client-ui-settings-remote`, key: `legacyEmpty`, zh: `没有遗留的宿主令牌。`, en: `No retired host token is stored.`},
	{module: `dsh-client-ui-settings-remote`, key: `savedServiceError`, zh: `配置已保存，但远程服务异常：{message}`, en: `Settings saved, but the remote service has an error: {message}`},
	{module: `dsh-client-ui-settings-remote`, key: `copyFailed`, zh: `无法复制链接，请重试。`, en: `Could not copy the link. Please try again.`},
	{module: `dsh-client-ui-settings-remote`, key: `qr`, zh: `配对二维码`, en: `Pairing QR code`},
	{module: `dsh-client-ui-settings-remote`, key: `scanSplitHint`, zh: `同一张码：Android App 内扫码＝链接设备；手机相机 / 浏览器扫码＝打开 web 端。`, en: `One code: scan inside the Android app to link the device; a camera / browser scan opens the web client.`},
	{module: `dsh-client-ui-settings-remote`, key: `relayDown`, zh: `中继未连接，正在重试`, en: `Relay offline — retrying`},
	{module: `dsh-client-ui-settings-remote`, key: `relayDownWithError`, zh: `中继未连接：{message}`, en: `Relay offline: {message}`},
	{module: `dsh-client-ui-settings-remote`, key: `relayDownDisconnected`, zh: `中继已断开，正在重试`, en: `Relay disconnected — retrying`},
	{module: `dsh-client-ui-settings-remote`, key: `relayUnavailable`, zh: `默认中继暂时不可用`, en: `Default relay is unavailable`},
	{module: `dsh-client-ui-settings-remote`, key: `startingHint`, zh: `正在启动远程服务…`, en: `Starting remote…`},
	{module: `dsh-client-ui-settings-remote`, key: `mintingHint`, zh: `正在生成配对码…`, en: `Creating pairing code…`},
	{module: `dsh-client-ui-settings-remote`, key: `statusErrorGeneric`, zh: `远程暂时不可用，请关闭后再开启重试`, en: `Remote unavailable — turn off then on to retry`},
	{module: `dsh-client-ui-settings-remote`, key: `errorPortInUse`, zh: `配对端口被占用，请关闭占用程序后点「开启」重试`, en: `Pairing port is in use — free it, then press On`},
	{module: `dsh-client-ui-settings-remote`, key: `errorBindGone`, zh: `监听网卡已失效——到 设置 → 远程 改选「监听范围」后重试`, en: `Saved listen address is gone — pick a new Listen scope under Settings → Remote, then retry`},
	{module: `dsh-client-ui-settings-remote`, key: `offHint`, zh: `开启后显示扫码配对二维码`, en: `Turn on to show the pairing QR code`},
	{module: `dsh-client-ui-settings-remote`, key: `noQr`, zh: `还没有可扫描的配对二维码`, en: `No pairing QR code yet`},
	{module: `dsh-client-ui-settings-remote`, key: `notListening`, zh: `远程服务未在运行——点「开启」重试`, en: `Remote service is not running — press On to retry`},
	{module: `dsh-client-ui-settings-remote`, key: `devicesManage`, zh: `设备管理`, en: `Manage devices`},
	{module: `dsh-client-ui-settings-remote`, key: `devicesEmpty`, zh: `还没有已配对的设备。扫码配对后会出现在这里。`, en: `No paired devices yet. They appear after a successful scan.`},
	{module: `dsh-client-ui-settings-remote`, key: `devicesSeen`, zh: `最近访问 {time}`, en: `Last seen {time}`},
	{module: `dsh-client-ui-settings-remote`, key: `devicesSeenUnknown`, zh: `尚未访问`, en: `Not seen yet`},
	{module: `dsh-client-ui-settings-remote`, key: `devicesBound`, zh: `配对于 {time}`, en: `Paired {time}`},
	{module: `dsh-client-ui-settings-remote`, key: `renameLabel`, zh: `设备名称`, en: `Device name`},
	{module: `dsh-client-ui-settings-remote`, key: `error`, zh: `暂时无法读取远程状态。`, en: `Could not read remote status.`},
	{module: `dsh-client-ui-settings-remote`, key: `statusError`, zh: `远程出错：{message}`, en: `Remote error: {message}`},
	{module: `dshd-remote`, key: `closePage`, zh: `关闭页面或窗口`, en: `Close Page or Window`},
	{module: `dshd-remote`, key: `aboutMenu`, zh: `关于 DeepSeek Harness`, en: `About DeepSeek Harness`},
	{module: `dshd-remote`, key: `aboutVersion`, zh: `版本 V{version}`, en: `Version V{version}`},
	{module: `dshd-remote`, key: `hideApplication`, zh: `隐藏 DeepSeek Harness`, en: `Hide DeepSeek Harness`},
	{module: `dshd-remote`, key: `hideOtherApplications`, zh: `隐藏其他`, en: `Hide Others`},
	{module: `dshd-remote`, key: `showAllApplications`, zh: `显示全部`, en: `Show All`},
	{module: `dshd-remote`, key: `quitApplication`, zh: `退出 DeepSeek Harness`, en: `Quit DeepSeek Harness`},
	{module: `dshd-remote`, key: `openApplication`, zh: `打开 DeepSeek Harness`, en: `Open DeepSeek Harness`},
	{module: `dshd-remote`, key: `quitTitle`, zh: `退出 DeepSeek Harness？`, en: `Quit DeepSeek Harness?`},
	{module: `dshd-remote`, key: `quitActiveTasks`, zh: `当前正在运行的任务将会中断`, en: `Running tasks will be interrupted.`},
	{module: `dshd-remote`, key: `quitScheduledTasks`, zh: `应用关闭期间，定时任务不会运行`, en: `Scheduled tasks will not run while the app is closed.`},
	{module: `dshd-remote`, key: `quitActiveAndScheduledTasks`, zh: `当前正在运行的任务将会中断，且应用关闭期间，定时任务不会运行`, en: `Running tasks will be interrupted, and scheduled tasks will not run while the app is closed.`},
	{module: `dshd-remote`, key: `backgroundNoticeBody`, zh: `正在运行的任务不会中断，可在系统托盘中重新打开窗口`, en: `Running tasks will continue. You can reopen the window from the system tray.`},
	{module: `dshd-remote`, key: `menuBar`, zh: `应用菜单`, en: `Application menu`},
	{module: `dshd-remote`, key: `redo`, zh: `重做`, en: `Redo`},
	{module: `dshd-remote`, key: `cut`, zh: `剪切`, en: `Cut`},
	{module: `dshd-remote`, key: `startupFailed`, zh: `DeepSeek Harness 无法使用`, en: `DeepSeek Harness is unavailable`},
	{module: `dshd-remote`, key: `fatalSummary`, zh: `应用无法启动或已意外停止。`, en: `The application could not start or stopped unexpectedly.`},
	{module: `dshd-remote`, key: `startupAddressInUse`, zh: `有其他正在运行的 DSH（如其他 dsh web、桌面端），无法同时启动，请退出其他正在运行的 DSH 后重启。`, en: `Another DSH instance (such as dsh web or the desktop app) is running. They cannot start at the same time. Quit the other running DSH instance, then restart.`},
	{module: `dshd-remote`, key: `diagnosticTruncated`, zh: `… 错误详情已截短。`, en: `… Error details shortened.`},
	{module: `dshd-remote`, key: `reportWrittenTo`, zh: `诊断报告：{path}`, en: `Diagnostic report: {path}`},
	{module: `dshd-remote`, key: `startupReinstallAdvice`, zh: `如果应用文件缺失或损坏，请关闭应用并重新安装。任务数据存储在独立位置。`, en: `If application files are missing or damaged, close the application and reinstall it. Your tasks are stored separately.`},
	{module: `dshd-remote`, key: `recoveryOperationFailed`, zh: `恢复操作失败`, en: `The recovery operation failed`},
	{module: `dshd-remote`, key: `disableThirdPartyPlugins`, zh: `禁用第三方插件、备份 profile patch 并重启`, en: `Disable third-party plugins, back up profile patch, and restart`},
	{module: `dshd-remote`, key: `welcomeAuthExpiredDescription`, zh: `请重新登录后继续操作`, en: `Sign in again to continue`},
	{module: `dshd-remote`, key: `welcomeAuthCopyFailed`, zh: `复制失败，请重试`, en: `Could not copy. Try again.`},
	{module: `dshd-remote`, key: `welcomeContinueFailed`, zh: `无法打开工作区，请重试。`, en: `Could not open the workspace. Please try again.`},
	{module: `dshd-remote`, key: `reloadPageMenu`, zh: `刷新页面`, en: `Reload Page`},
	{module: `dshd-remote`, key: `restartAppHostMenu`, zh: `重启应用与 Host`, en: `Restart App and Host`},
	{module: `dshd-remote`, key: `updateCheckFailedTitle`, zh: `更新检查失败`, en: `Update Check Failed`},
	{module: `dshd-remote`, key: `updateCheckFailed`, zh: `检查更新失败，请稍后重试。`, en: `Could not check for updates. Please try again later.`},
	{module: `dshd-remote`, key: `updateDownloadFailed`, zh: `下载更新失败，请重试。`, en: `Could not download the update. Please try again.`},
	{module: `dshd-remote`, key: `updateInstallFailed`, zh: `安装更新失败，请稍后重试。`, en: `Could not install the update. Please try again later.`},
	{module: `dshd-remote`, key: `updateCheckNetworkFailed`, zh: `检查更新失败，请稍后重试。网络连接异常，请检查网络后重试。`, en: `Could not check for updates. Please try again later. The connection was interrupted. Check your network and try again.`},
	{module: `dshd-remote`, key: `updateDownloadNetworkFailed`, zh: `下载更新失败，请重试。网络连接异常，请检查网络后重试。`, en: `Could not download the update. Please try again. The connection was interrupted. Check your network and try again.`},
	{module: `dshd-remote`, key: `updateInstallNetworkFailed`, zh: `安装更新失败，请稍后重试。网络连接异常，请检查网络后重试。`, en: `Could not install the update. Please try again later. The connection was interrupted. Check your network and try again.`},
	{module: `dshd-remote`, key: `unknownError`, zh: `未知错误`, en: `Unknown error`},
	{module: `dshd-remote`, key: `updateCurrent`, zh: `当前暂无可用更新。当前版本：V{version}`, en: `No updates available. Current version: V{version}`},
	{module: `dshd-remote`, key: `updateDownloadedTitle`, zh: `DeepSeek Harness v{version} 下载完成`, en: `DeepSeek Harness v{version} downloaded`},
	{module: `dshd-remote`, key: `updateDownloadedDetail`, zh: `安装包已下载完毕，点击“安装并重启”，即刻重启客户端，开始部署。`, en: `The update package has downloaded. Select “Install and Restart” to restart the app and begin installation.`},
	{module: `dshd-remote`, key: `updateDownloadedTitleWindows`, zh: `新版本 v{version} 已准备就绪`, en: `New version v{version} is ready`},
	{module: `dshd-remote`, key: `updateDownloadedDetailWindows`, zh: `更新期间应用将暂时关闭，完成后会自动打开。

更新可能需要一些时间，请耐心等待，期间请勿重复启动应用。`, en: `The app will close temporarily during the update and reopen automatically when it is complete.

The update may take some time. Please wait and do not launch the app again during installation.`},
	{module: `dshd-remote`, key: `updateLater`, zh: `稍后更新`, en: `Update later`},
	{module: `dshd-remote`, key: `updateDownloading`, zh: `正在下载 {percent}%…`, en: `Downloading {percent}%…`},
	{module: `dshd-remote`, key: `updateVerifying`, zh: `正在校验更新文件…`, en: `Verifying update files…`},
	{module: `dshd-remote`, key: `updateActiveTasks`, zh: `仍有进行中的任务`, en: `Tasks are still in progress`},
	{module: `dshd-remote`, key: `updateActiveTasksDetail`, zh: `重启更新可能中断这些任务，是否要继续更新？`, en: `Restarting to update may interrupt these tasks. Continue updating?`},
	{module: `dshd-remote`, key: `updateStopTasks`, zh: `停止任务并更新`, en: `Stop tasks and update`},
	{module: `dshd-remote`, key: `updateTasksChanged`, zh: `有新任务开始，请重新确认更新。`, en: `New tasks started. Review the update confirmation again.`},
	{module: `dshd-remote`, key: `updateTasksUnavailable`, zh: `无法确认任务状态，请在工作区就绪后重试更新。`, en: `Task status is unavailable. Try updating again when the workspace is ready.`},
	{module: `dshd-remote`, key: `updateStopFailed`, zh: `未能安全停止任务，更新未安装。请稍后重试。`, en: `Tasks could not be stopped safely. The update was not installed. Please try again later.`},
	{module: `dshd-remote`, key: `updateTechnicalDetails`, zh: `查看技术详情`, en: `View technical details`},
	{module: `dshd-remote`, key: `updateTitle`, zh: `DeepSeek Harness 更新`, en: `DeepSeek Harness Update`},
	{module: `dshd-remote`, key: `updateAvailable`, zh: `发现可用更新`, en: `An update is available`},
	{module: `dshd-remote`, key: `updateDetail`, zh: `DeepSeek Harness {version}

DeepSeek Harness 将重启以完成更新。`, en: `DeepSeek Harness {version}

DeepSeek Harness will restart to complete the update.`},
	{module: `dshd-remote`, key: `mandatoryTitle`, zh: `需要更新`, en: `Update required`},
	{module: `dshd-remote`, key: `mandatoryDetail`, zh: `当前版本已停止支持，请更新后继续使用。在您确认重启之前，现有任务可以继续运行。`, en: `This version is no longer supported. Update to continue. Existing tasks can keep running until you approve a restart.`},
	{module: `dshd-remote`, key: `mandatoryUnavailable`, zh: `暂时无法检查更新要求，请在网络恢复后重试。`, en: `The update requirement could not be checked. Retry when the connection is available.`},
	{module: `dshd-remote`, key: `policyLoginTitle`, zh: `登录测试环境`, en: `Sign in to the test environment`},
	{module: `dshd-remote`, key: `policyLoginRequired`, zh: `这是测试版应用，检查更新要求需要先通过飞书登录。登录不会下载或安装更新。`, en: `This is a test build. Checking update requirements needs Feishu sign-in. Signing in does not download or install an update.`},
	{module: `dshd-remote`, key: `policyLogin`, zh: `通过飞书登录`, en: `Sign in with Feishu`},
	{module: `dshd-remote`, key: `policyLoginFailed`, zh: `测试环境登录未完成，请检查网络后重试。`, en: `Test environment sign-in did not complete. Please check your connection and try again.`},
	{module: `dshd-remote`, key: `policyLoginLoading`, zh: `正在加载登录页面…`, en: `Loading sign-in page…`},
	{module: `dshd-remote`, key: `mandatoryNoRelease`, zh: `暂时没有可用的更新，请重新检查或联系支持人员。`, en: `No applicable update is available. Check again or contact support.`},
	{module: `dshd-remote`, key: `mandatoryPage`, zh: `前往官网下载`, en: `Open download page`},
	{module: `dshd-remote`, key: `mandatoryCopy`, zh: `复制下载链接`, en: `Copy download address`},
	{module: `dshd-remote`, key: `mandatoryPageFailed`, zh: `无法打开浏览器，请复制下载链接后手动打开。`, en: `The download page could not be opened. Copy the address below and open it in your browser.`},
	{module: `dshd-remote`, key: `mandatoryActionFailed`, zh: `更新操作失败，请重试；应用仍需更新后才能继续使用。`, en: `The update action failed. Retry; the update requirement remains active.`},
	{module: `dshd-remote`, key: `mandatoryReady`, zh: `更新已准备就绪`, en: `Update ready`},
	{module: `dshd-remote`, key: `mandatoryVersion`, zh: `V{version}`, en: `V{version}`},
	{module: `dshd-remote`, key: `mandatoryReadyDetail`, zh: `安装后将重新启动应用。`, en: `Installing the update will restart the application.`},
	{module: `dshd-remote`, key: `mandatoryDeferred`, zh: `现有任务可以继续运行。完成更新后才能继续操作应用。`, en: `Existing tasks can keep running. Update to continue using the application.`},
	{module: `dshd-remote`, key: `mandatoryContinue`, zh: `继续安装更新`, en: `Continue installing update`},
	{module: `dshd-remote`, key: `mandatoryInspecting`, zh: `正在检查任务状态…`, en: `Checking tasks…`},
	{module: `dshd-remote`, key: `mandatoryStopping`, zh: `正在安全结束应用中的任务。`, en: `Safely stopping tasks in the application.`},
	{module: `dshd-remote`, key: `mandatoryRestarting`, zh: `应用即将重启，请稍候。`, en: `The application will restart shortly. Please wait.`},
	{module: `dshd-remote`, key: `mandatoryDownloadFailed`, zh: `更新文件下载或准备失败，请重试。`, en: `The update files could not be downloaded or prepared. Please retry.`},
	{module: `dshd-remote`, key: `mandatoryInstallFailed`, zh: `更新尚未安装，请重新检查任务后重试。`, en: `The update was not installed. Check tasks again and retry.`},
	{module: `dshd-remote`, key: `mandatoryOpenHelp`, zh: `若页面未打开，可`, en: `If the page did not open, you can`},
	{module: `dshd-remote`, key: `mandatoryReopen`, zh: `重新打开官网`, en: `Open download page again`},
	{module: `dshd-remote`, key: `mandatoryCopied`, zh: `已复制链接`, en: `Link copied`},
	{module: `dshd-remote`, key: `mandatoryCopyFailed`, zh: `复制失败，请手动选择下方地址复制。`, en: `Copy failed. Select and copy the address below manually.`},
	{module: `dshd-remote`, key: `mandatoryAddress`, zh: `下载地址`, en: `Download address`},
	{module: `dshd-remote`, key: `mandatoryNotification`, zh: `返回应用确认安装并重启。`, en: `Return to the application to confirm installation and restart.`},
}

func init() {
	for _, entry := range remoteLocaleCatalog {
		i18n.Register("remote"+"."+entry.module+"."+entry.key, entry.zh, entry.en)
	}
}

func remoteLocaleText(key string) string {
	zh := i18n.Locale() == i18n.LocaleZH
	for _, entry := range remoteLocaleCatalog {
		if entry.key == key || entry.module+"."+entry.key == key {
			if zh {
				return entry.zh
			}
			return entry.en
		}
	}
	return i18n.T(key)
}
