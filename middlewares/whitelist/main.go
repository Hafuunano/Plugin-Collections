// Package whitelist provides a middleware that allows only groups listed in config.
// Config is read from data/config/whitelist/config.yaml; if missing, a default empty list is created and saved.
package whitelist

import (
	"os"
	"sync"

	"github.com/Hafuunano/Plugin-Collections/lib/database/config"
	"github.com/Hafuunano/UniTransfer/protocol"
)

const pluginName = "whitelist"

// Config is the whitelist config file structure.
type Config struct {
	GroupIDs []string `yaml:"group_ids"`
}

var (
	mu       sync.RWMutex
	allowed  map[string]struct{} // in-memory set of allowed group IDs
	dataDir  string
	loaded   bool
)

// load reads config from disk, creates default config if not exist, and refreshes in-memory set.
func load() error {
	mu.Lock()
	defer mu.Unlock()
	if dataDir == "" {
		dataDir = os.Getenv("DATA_DIR")
		if dataDir == "" {
			dataDir = "data"
		}
	}
	var cfg Config
	if err := config.Read(dataDir, pluginName, &cfg); err != nil {
		return err
	}
	if !config.Exists(dataDir, pluginName) || cfg.GroupIDs == nil {
		cfg.GroupIDs = []string{}
		if err := config.Save(dataDir, pluginName, &cfg); err != nil {
			return err
		}
	}
	allowed = make(map[string]struct{}, len(cfg.GroupIDs))
	for _, id := range cfg.GroupIDs {
		if id != "" {
			allowed[id] = struct{}{}
		}
	}
	loaded = true
	return nil
}

func isAllowed(groupID string) bool {
	mu.RLock()
	if !loaded {
		mu.RUnlock()
		if err := load(); err != nil {
			return false
		}
		mu.RLock()
	}
	_, ok := allowed[groupID]
	mu.RUnlock()
	return ok
}

// Handler returns a handler that continues only when ctx.GroupID() is in the whitelist.
// dataDir is the root data directory (e.g. "data"); if empty, DATA_DIR env or "data" is used.
// If the config file does not exist, a default config with an empty group_ids list is created.
func Handler(dataDirRoot string) func(protocol.Context, func()) {
	if dataDirRoot != "" {
		mu.Lock()
		dataDir = dataDirRoot
		loaded = false
		mu.Unlock()
	}
	return func(ctx protocol.Context, next func()) {
		groupID := ctx.GroupID()
		if groupID == "" {
			next()
			return
		}
		if !isAllowed(groupID) {
			msg := protocol.Message{
				protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": "此群未在白名单中"}},
			}
			_ = ctx.Send(msg)
			return
		}
		next()
	}
}
