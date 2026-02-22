// Package pluginecho: example plugin that echoes back user message when user says "echo <text>".
// Demonstrates multi-file layout (entry in main.go, logic in handler.go).
// See docs/WRITING_PLUGINS_AND_MIDDLEWARES.md for the plugin contract.
package pluginecho

import (
	"github.com/Hafuunano/Core-SkillAction/types"
	"github.com/Hafuunano/Protocol-ConvertTool/protocol"
)

// Meta and registration (required: use WithMeta(Meta) then chain).
var Meta = types.NewPluginEngine("plugin-echo-001", "plugin-echo", "skill", true)
var p = protocol.Engine.WithMeta(Meta)

func init() {
	p.OnMessage().Func(Plugin)
}

// Plugin is the required entry. Host calls it for each message with a protocol.Context.
func Plugin(ctx protocol.Context) {
	handleEcho(ctx)
}
