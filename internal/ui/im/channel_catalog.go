package im

import "whaleisle/internal/ui/i18n"

func channelText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}

type channelCopy struct {
	name, transport, title, description string
	steps                               []string
}

func copyForChannel(id string) channelCopy {
	switch id {
	case "feishu":
		return channelCopy{channelText("飞书", "Feishu"), channelText("事件长连接", "Event connection"), channelText("使用飞书扫码创建机器人", "Create a bot with Feishu"), channelText("扫码只会新增一个机器人，已接入的机器人会继续正常收发消息。", "Scanning adds one bot; existing bots keep working."), []string{channelText("打开飞书移动端，使用扫一扫读取二维码", "Scan with the Feishu mobile app"), channelText("核对应用名称与权限范围，并确认创建", "Check the app name and permissions, then confirm"), channelText("保持本页打开，等待新机器人的长连接就绪", "Keep this page open until the bot connects")}}
	case "weixin":
		return channelCopy{channelText("微信", "WeChat"), channelText("消息长轮询", "Message polling"), channelText("使用手机微信扫描二维码", "Scan with WeChat"), channelText("请在手机上核对并确认授权。部分账号会额外显示配对数字，需要时本页会提示输入。", "Confirm on your phone. If WeChat shows a pairing code, this page will ask for it."), []string{channelText("打开手机微信并扫描二维码", "Scan with the WeChat mobile app"), channelText("在微信中确认连接该机器人", "Confirm the bot connection in WeChat"), channelText("保持本页打开，等待消息长轮询变为在线", "Keep this page open until message polling is online")}}
	case "dingtalk":
		return channelCopy{channelText("钉钉", "DingTalk"), "Stream", channelText("使用钉钉 App 完成机器人授权", "Authorize with DingTalk"), channelText("扫码账号必须已加入企业/组织并有权创建机器人。若提示未加入组织，请创建组织或换用已有组织账号。", "Use an account that belongs to an organization and can create bots. Create an organization or switch accounts if required."), []string{channelText("使用已加入企业/组织的钉钉账号扫码", "Scan with an organization account"), channelText("在授权页点击“一键创建新机器人”", "Choose Create new bot on the authorization page"), channelText("保持本页打开，等待机器人自动连接", "Keep this page open while the bot connects")}}
	case "qq":
		return channelCopy{"QQ", channelText("机器人长连接", "Bot connection"), channelText("使用手机 QQ 完成机器人绑定", "Link a bot with QQ"), channelText("腾讯页面会创建或绑定一个 QQ 机器人，并将连接凭据安全交给本机 Host。", "Tencent creates or links a QQ bot and securely hands its credentials to this Host."), []string{channelText("打开手机 QQ 并扫码", "Scan with the QQ mobile app"), channelText("在腾讯授权页确认创建或绑定机器人", "Confirm creation or linking on Tencent's page"), channelText("返回这里等待连接完成", "Return here and wait for the connection")}}
	case "wecom":
		return channelCopy{channelText("企业微信", "WeCom"), channelText("智能机器人长连接", "Smart bot connection"), channelText("使用企业微信 App 完成智能机器人授权", "Authorize a WeCom smart bot"), channelText("企业微信官方页面会创建智能机器人，并将连接凭据安全交给本机 Host。", "WeCom creates a smart bot and securely hands its credentials to this Host."), []string{channelText("打开企业微信 App 并扫码", "Scan with the WeCom app"), channelText("在腾讯授权页确认创建智能机器人", "Confirm smart bot creation on Tencent's page"), channelText("返回这里等待连接完成", "Return here and wait for the connection")}}
	case "whatsapp":
		return channelCopy{"WhatsApp", channelText("关联设备连接", "Linked device"), channelText("用手机 WhatsApp 扫描二维码", "Scan with WhatsApp"), "", []string{channelText("打开 WhatsApp → 设置 → 已关联设备", "Open WhatsApp → Settings → Linked devices"), channelText("点击“关联设备”并扫描二维码", "Choose Link a device and scan the code")}}
	case "telegram":
		return channelCopy{"Telegram", channelText("Bot API 长轮询", "Bot API polling"), channelText("接入 Telegram 机器人", "Connect a Telegram bot"), channelText("先通过 @BotFather 获取 Bot Token，再在这里完成接入。", "Get a Bot Token from @BotFather, then connect it here."), nil}
	case "discord":
		return channelCopy{"Discord", channelText("Gateway 长连接", "Gateway connection"), channelText("接入 Discord 机器人", "Connect a Discord bot"), channelText("先在 Developer Portal 创建 Bot 并邀请到服务器，再在这里完成接入。", "Create a bot in Developer Portal and invite it to a server, then connect it here."), nil}
	case "slack":
		return channelCopy{"Slack", "Socket Mode", channelText("接入 Slack 机器人", "Connect a Slack bot"), channelText("先用 Manifest 创建并配置 Slack App；安装到工作区，并生成包含 connections:write 的 App Token。", "Create the Slack app from the Manifest, install it to a workspace, and generate an App Token with connections:write."), nil}
	}
	return channelCopy{}
}
