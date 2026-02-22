// Package pluginecho: example plugin that echoes back user message when user says "echo <text>".
// Demonstrates multi-file layout (entry in main.go, logic in handler.go).
// See docs/WRITING_PLUGINS_AND_MIDDLEWARES.md for the plugin contract.
package pluginecho

import (
	"github.com/Hafuunano/Core-SkillAction/types"
	"github.com/Hafuunano/Protocol-ConvertTool/protocol"
)

// Meta is this plugin's metadata; use Meta.PluginID, Meta.PluginName, etc. inside this package.
var Meta = types.NewPluginEngine("plugin-echo-001", "plugin-echo", "skill", true)

func init() {
	protocol.Register(Plugin)
}

// Plugin is the required entry. Host calls it for each message with a protocol.Context.
func Plugin(ctx protocol.Context) {
	handleEcho(ctx)
}
