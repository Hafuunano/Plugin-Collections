package pluginping

import "github.com/Hafuunano/Protocol-ConvertTool/protocol"

func init() { protocol.Engine.WithMeta(nil).OnMessage().Func(Plugin) }

func Plugin(ctx protocol.Context) {
	if ctx.PlainText() == "ping" {
		ctx.Reply(protocol.Message{
			protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": "pong"}},
		})
	}
}
