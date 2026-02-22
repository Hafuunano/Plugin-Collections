// Package pluginping: replies "pong" when user says "ping". WithMeta(nil) at init (no meta).
package pluginping

import "github.com/Hafuunano/Protocol-ConvertTool/protocol"

var p = protocol.Engine.WithMeta(nil)

func init() {
	p.OnMessage("ping").Func(Plugin)
}

func Plugin(ctx protocol.Context) {
	ctx.Reply(protocol.Message{
		protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": "pong"}},
	})
}
