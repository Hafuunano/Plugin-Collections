// Package pluginecho: example plugin that echoes back user message when user says "echo <text>".
// Demonstrates multi-file layout (entry in main.go, logic in handler.go).
// See docs/WRITING_PLUGINS_AND_MIDDLEWARES.md for the plugin contract.
package pluginecho

import (
	"github.com/Hafuunano/Protocol-ConvertTool/protocol"
)

// Plugin is the required entry. Host calls it for each message with a protocol.Context.
func Plugin(ctx protocol.Context) {
	handleEcho(ctx)
}
