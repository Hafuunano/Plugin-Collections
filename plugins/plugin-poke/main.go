// Package pluginpoke: when user @ bot with empty message or "戳我"/"戳戳", reply with random text then poke the user.
// See docs/WRITING_PLUGINS_AND_MIDDLEWARES.md for the plugin contract.
package pluginpoke

import (
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/Hafuunano/Core-SkillAction/types"
	"github.com/Hafuunano/Protocol-ConvertTool/protocol"
)

// Meta and registration (required: use WithMeta(Meta) then chain).
var Meta = types.NewPluginEngine("plugin-poke-001", "plugin-poke", "skill", true)
var p = protocol.Engine.WithMeta(Meta)

// botNick returns the first bot nickname from env NICK_NAMES (same as Lucy), or "咱" if not set.
func botNick() string {
	nickNames := strings.TrimSpace(os.Getenv("NICK_NAMES"))
	if nickNames == "" {
		return "咱"
	}
	first := strings.TrimSpace(strings.Split(nickNames, ",")[0])
	if first == "" {
		return "咱"
	}
	return first
}

// replyTexts use bot nickname; built at send time via getReplyTexts().
func getReplyTexts() []string {
	nick := botNick()
	return []string{
		"这里是" + nick + "(っ●ω●)っ",
		nick + "不在呢~",
		"哼！" + nick + "不想理你~",
	}
}

// triggerWords: only respond when plain text is empty or one of these.
var triggerWords = map[string]bool{
	"": true, "戳我": true, "戳戳": true,
}

func init() {
	p.OnMessageReply().Func(Plugin)
}

// Plugin is the required entry. Runs only when message is reply/@ or prefix is bot name (OnlyToMe).
func Plugin(ctx protocol.Context) {
	plain := strings.TrimSpace(ctx.PlainText())
	if !triggerWords[plain] {
		return
	}
	// Send random reply text then poke user (align with chat.go behavior).
	replyTexts := getReplyTexts()
	idx := rand.Intn(len(replyTexts))
	_ = ctx.SendPlainMessage(replyTexts[idx])
	time.Sleep(1 * time.Second)
	_ = ctx.SendPoke(ctx.UserID())
	ctx.BlockNext()
}
