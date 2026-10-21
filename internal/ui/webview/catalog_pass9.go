package webview

import "whaleisle/internal/ui/i18n"

const keyWebViewDownloadRuntime = "webview.download_runtime"

func init() {
	i18n.Register(keyWebViewDownloadRuntime, "下载 WebView2 运行时", "Download the WebView2 runtime")
}