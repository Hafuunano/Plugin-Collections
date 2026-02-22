// Package pluginhello: example plugin that replies "Hello, {nick}!" when user says "hello".
// See docs/WRITING_PLUGINS_AND_MIDDLEWARES.md for the plugin contract.
package pluginhello

import (
	"github.com/Hafuunano/Core-SkillAction/types"
	"github.com/Hafuunano/Protocol-ConvertTool/protocol"
)

// Meta and registration (required: use WithMeta(Meta) then chain).
var Meta = types.NewPluginEngine("plugin-hello-001", "plugin-hello", "skill", true)
var p = protocol.Engine.WithMeta(Meta)

func init() {
	p.OnMessage().Func(Plugin)
}

// triggerWord is loaded in Init(); default is "hello".
var triggerWord = "hello"

// Init is optional. Host may call it once at startup.
func Init() {
	// e.g. load from config file or env; here we keep default.
	triggerWord = "hello"
}

// Plugin is the required entry. Host calls it for each message with a protocol.Context.
func Plugin(ctx protocol.Context) {
	text := ctx.PlainText()
	if text != triggerWord {
		return
	}
	nick := ctx.SenderNickname()
	if nick == "" {
		nick = "user"
	}
	// Reply with a simple text segment.
	_ = ctx.Reply(protocol.Message{
		protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": "Hello, " + nick + "!"}},
	})
}
