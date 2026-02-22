package pluginecho

import (
	"strings"

	"github.com/Hafuunano/Protocol-ConvertTool/protocol"
)

// echoPrefix is the command prefix for echo (e.g. "echo hello" -> reply "hello").
const echoPrefix = "echo "

// handleEcho handles "echo <text>": reply with the rest of the message as plain text.
func handleEcho(ctx protocol.Context) {
	text := strings.TrimSpace(ctx.PlainText())
	if !strings.HasPrefix(text, echoPrefix) {
		return
	}
	rest := strings.TrimSpace(text[len(echoPrefix):])
	if rest == "" {
		rest = "(empty)"
	}
	_ = ctx.Reply(protocol.Message{
		protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": rest}},
	})
}
