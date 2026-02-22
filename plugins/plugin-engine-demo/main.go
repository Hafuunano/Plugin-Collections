// Package pluginenginedemo: example plugin using the fluent Engine API with multiple registrations.
// WithMeta(Meta) is required at init; then chain OnMessage()/OnMessageReply(), conditions, and Func(handler).
package pluginenginedemo

import (
	"github.com/Hafuunano/Core-SkillAction/types"
	"github.com/Hafuunano/Protocol-ConvertTool/protocol"
)

// Meta and registration (required: use WithMeta(Meta) then chain).
var Meta = types.NewPluginEngine("plugin-engine-demo-001", "plugin-engine-demo", "skill", true)
var p = protocol.Engine.WithMeta(Meta)

func init() {
	// All messages: inline func(ctx). Respond to "help" or "引擎示例"
	p.OnMessage().Func(func(ctx protocol.Context) {
		text := ctx.PlainText()
		if text != "help" && text != "引擎示例" {
			return
		}
		_ = ctx.SendPlainMessage("Engine demo: say 管理@bot (admin) or 超管@bot (super admin).")
	})
	// Only when reply/@ bot and sender is group admin: inline handler for "管理" or "admin"
	p.OnMessage().IsOnlyToMe().IsOnlyAdmin().Func(func(ctx protocol.Context) {
		text := ctx.PlainText()
		if text != "管理" && text != "admin" {
			return
		}
		_ = ctx.SendWithReply(protocol.Message{
			protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": "You are admin, reply chain."}},
		})
	})
	// Only on reply chain and super admin: inline handler for "超管" or "super"
	p.OnMessageReply().IsOnlySuperAdmin().Func(func(ctx protocol.Context) {
		text := ctx.PlainText()
		if text != "超管" && text != "super" {
			return
		}
		_ = ctx.SendPlainMessage("You are super admin, reply chain.")
	})
}
