// Package whitelist provides a group whitelist middleware. Use per-plugin (Middleware + SetStore) or inject globally: zerobot.InstallWithMiddlewares([]protocol.Middleware{whitelist.New(services.Cache)}). Private chat always passes.
package whitelist

import (
	"sync"

	"github.com/Hafuunano/Core-SkillAction/cache/database"
	"github.com/Hafuunano/Core-SkillAction/types"
	"github.com/Hafuunano/Protocol-ConvertTool/protocol"
)

const keyPrefix = "whitelist:group:"

var (
	storeMu sync.RWMutex
	store   *database.Store
)

// Meta is this middleware's metadata for config (MiddlewareName etc.).
var Meta = types.MiddlewareEngine{
	MiddlewareID:   "middleware-whitelist",
	MiddlewareName: "whitelist",
}

// SetStore sets the Skill cache/database store used by Middleware. Call from host at startup (e.g. whitelist.SetStore(services.Cache)) before loading plugins.
func SetStore(s *database.Store) {
	storeMu.Lock()
	defer storeMu.Unlock()
	store = s
}

// Middleware wraps a single plugin handler: the plugin runs only in whitelisted groups (or private chat). Use when registering the plugin, e.g. protocol.Engine.WithMeta(Meta).OnMessage().Func(whitelist.Middleware(Plugin)).
func Middleware(next protocol.Handler) protocol.Handler {
	return func(ctx protocol.Context) {
		gid := ctx.GroupID()
		if gid == "" || gid == "0" {
			next(ctx)
			return
		}
		storeMu.RLock()
		s := store
		storeMu.RUnlock()
		if s != nil && !HasGroup(s, gid) {
			return
		}
		next(ctx)
	}
}

// New returns a protocol.Middleware that uses the given store. Use for global injection, e.g. zerobot.InstallWithMiddlewares([]protocol.Middleware{whitelist.New(services.Cache)}), or when you have a store at hand.
func New(s *database.Store) protocol.Middleware {
	return func(next protocol.Handler) protocol.Handler {
		return func(ctx protocol.Context) {
			gid := ctx.GroupID()
			if gid == "" || gid == "0" {
				next(ctx)
				return
			}
			if s != nil && !HasGroup(s, gid) {
				return
			}
			next(ctx)
		}
	}
}

// HasGroup returns true if the group is in the whitelist.
func HasGroup(store *database.Store, groupID string) bool {
	if store == nil || groupID == "" {
		return false
	}
	_, found, _ := store.Get(keyPrefix + groupID)
	return found
}

// AddGroup adds a group to the whitelist.
func AddGroup(store *database.Store, groupID string) error {
	if store == nil || groupID == "" {
		return nil
	}
	return store.Set(keyPrefix+groupID, "1")
}

// RemoveGroup removes a group from the whitelist.
func RemoveGroup(store *database.Store, groupID string) error {
	if store == nil || groupID == "" {
		return nil
	}
	return store.Delete(keyPrefix + groupID)
}
