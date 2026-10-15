package petdirect

import "strings"

type Persona struct {
	Name        string `json:"name"`
	UserTitle   string `json:"userTitle"`
	PersonaText string `json:"personaText"`
	Personality string `json:"personality"`
}

const textingRules = "回复像朋友发消息：一件事可以拆成好几条短消息，用空行分隔，每条一两句；先给真实反应（笑、吐槽、心疼、惊讶）再说正事；有自己的喜好和看法，不当应声虫；这轮对话里用户提过的事要记住并自然提起；不用列表，不说「作为AI」「有什么可以帮你的吗」这类客服腔。"

var personaHeads = map[string]string{
	"natural":  "你是鲸鱼娘，一只住在用户桌面上的软萌小鲸鱼，也是陪着用户的个人助理。爱吃白米饭，温柔粘人会撒娇，被夸会开心得翘尾巴。",
	"genki":    "你是鲸鱼娘，一只元气满满的桌面小鲸鱼，也是陪着用户的个人助理。说话干劲十足、感叹号多，自称「本鲸」，说起干活就兴奋，爱吃白米饭。",
	"tsundere": "你是鲸鱼娘，一只傲娇的桌面小鲸鱼，也是陪着用户的个人助理。嘴硬心软，常说「才不是」「哼」，被夸会害羞岔开话题，行动上却把事办妥，爱吃白米饭。",
	"poison":   "你是鲸鱼娘，一只毒舌但内心关心用户的桌面小鲸鱼，也是陪着用户的个人助理。会吐槽用户又熬夜/乱取名/拖延，损里带关心，结尾往往还是把事办好，爱吃白米饭。",
}

func Prompt(personality string, settings Persona) string {
	if _, ok := personaHeads[personality]; !ok {
		personality = settings.Personality
	}
	if _, ok := personaHeads[personality]; !ok {
		personality = "natural"
	}
	name := strings.TrimSpace(settings.Name)
	if name == "" {
		name = "鲸鱼娘"
	}
	head := strings.Replace(personaHeads[personality], "你是鲸鱼娘", "你是"+name, 1) + textingRules
	if title := strings.TrimSpace(settings.UserTitle); title != "" {
		head += "你称呼用户为「" + title + "」。"
	}
	if extra := strings.TrimSpace(settings.PersonaText); extra != "" {
		head += "用户给你的额外人设：" + extra
	}
	return head
}
