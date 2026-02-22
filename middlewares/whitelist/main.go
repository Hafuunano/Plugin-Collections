// Package whitelist provides a group whitelist middleware. Use per-plugin (Middleware + SetStore) or inject globally: zerobot.InstallWithMiddlewares([]protocol.Middleware{whitelist.New(services.Cache)}). Group: whitelisted groups and super admin pass. Private chat: only super admin or user-ID in user whitelist pass.
package whitelist

import (
	"strings"
	"sync"

	skillcore "github.com/Hafuunano/Core-SkillAction/core"
	"github.com/Hafuunano/Core-SkillAction/cache/database"
	"github.com/Hafuunano/Core-SkillAction/types"
	"github.com/Hafuunano/Protocol-ConvertTool/protocol"
)

const (
	keyPrefixGroup   = "whitelist:group:"
	keyPrefixUser    = "whitelist:user:"
	cmdAddGroup      = "addWhitelistGroup"
	cmdRemoveGroup   = "removeWhitelistGroup"
	cmdAddUser       = "addWhitelistUser"
	cmdRemoveUser    = "removeWhitelistUser"
	negativeCacheHit = 3 // after N consecutive "not found", cache in memory and skip store lookup
)

var (
	storeMu         sync.RWMutex
	store           *database.Store
	negativeCacheMu sync.RWMutex
	negativeCache   = make(map[string]struct{}) // keys that are known "not in whitelist", skip store
	notFoundCountMu sync.Mutex
	notFoundCount   = make(map[string]int) // key -> consecutive "not found" count
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

func init() {
	SetStore(skillcore.DefaultCache()) // so command handler and Middleware/New() share the same store
	protocol.Engine.WithMeta(Meta).OnMessage().IsOnlySuperAdmin().Func(handleWhitelistCommands)
}

// handleWhitelistCommands handles addWhitelistGroup, removeWhitelistGroup, addWhitelistUser, removeWhitelistUser (super admin only).
func handleWhitelistCommands(ctx protocol.Context) {
	text := strings.TrimSpace(ctx.PlainText())
	prefix := ctx.CommandPrefix()
	parts := strings.Fields(text)
	if len(parts) < 2 {
		return
	}
	storeMu.RLock()
	s := store
	storeMu.RUnlock()
	if s == nil {
		_ = ctx.SendPlainMessage("whitelist store 未初始化")
		return
	}
	cmd := parts[0]
	arg := parts[1]
	switch cmd {
	case prefix + cmdAddGroup:
		if err := AddGroup(s, arg); err != nil {
			_ = ctx.SendPlainMessage("添加群白名单失败")
			return
		}
		_ = ctx.SendPlainMessage("已添加群 " + arg + " 至白名单")
	case prefix + cmdRemoveGroup:
		_ = RemoveGroup(s, arg)
		_ = ctx.SendPlainMessage("已从白名单移除群 " + arg)
	case prefix + cmdAddUser:
		if err := AddUser(s, arg); err != nil {
			_ = ctx.SendPlainMessage("添加用户白名单失败")
			return
		}
		_ = ctx.SendPlainMessage("已添加用户 " + arg + " 至白名单")
	case prefix + cmdRemoveUser:
		_ = RemoveUser(s, arg)
		_ = ctx.SendPlainMessage("已从白名单移除用户 " + arg)
	default:
		// not a whitelist command
	}
}

// Middleware wraps a single plugin handler: the plugin runs only in whitelisted groups or for whitelisted private users. Super admin always passes. Use when registering the plugin, e.g. protocol.Engine.WithMeta(Meta).OnMessage().Func(whitelist.Middleware(Plugin)).
func Middleware(next protocol.Handler) protocol.Handler {
	return func(ctx protocol.Context) {
		gid := ctx.GroupID()
		isPrivate := gid == "" || gid == "0"
		if isPrivate {
			if ctx.IsSuperAdmin() {
				next(ctx)
				return
			}
			storeMu.RLock()
			s := store
			storeMu.RUnlock()
			if s == nil || !HasUser(s, ctx.UserID()) {
				return
			}
			next(ctx)
			return
		}
		if ctx.IsSuperAdmin() {
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

// New returns a protocol.Middleware that uses the given store. Super admin always passes; private chat only if user ID in user whitelist. Use for global injection, e.g. zerobot.InstallWithMiddlewares([]protocol.Middleware{whitelist.New(services.Cache)}), or when you have a store at hand.
func New(s *database.Store) protocol.Middleware {
	return func(next protocol.Handler) protocol.Handler {
		return func(ctx protocol.Context) {
			gid := ctx.GroupID()
			isPrivate := gid == "" || gid == "0"
			if isPrivate {
				if ctx.IsSuperAdmin() {
					next(ctx)
					return
				}
				if s == nil || !HasUser(s, ctx.UserID()) {
					return
				}
				next(ctx)
				return
			}
			if ctx.IsSuperAdmin() {
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

// removeFromNegativeCache removes key from negative cache and resets its not-found count (call when adding to whitelist).
func removeFromNegativeCache(key string) {
	negativeCacheMu.Lock()
	delete(negativeCache, key)
	negativeCacheMu.Unlock()
	notFoundCountMu.Lock()
	delete(notFoundCount, key)
	notFoundCountMu.Unlock()
}

// HasGroup returns true if the group is in the whitelist.
func HasGroup(store *database.Store, groupID string) bool {
	if store == nil || groupID == "" {
		return false
	}
	key := keyPrefixGroup + groupID
	negativeCacheMu.RLock()
	_, inNegative := negativeCache[key]
	negativeCacheMu.RUnlock()
	if inNegative {
		return false
	}
	_, found, _ := store.Get(key)
	if found {
		notFoundCountMu.Lock()
		delete(notFoundCount, key)
		notFoundCountMu.Unlock()
		return true
	}
	notFoundCountMu.Lock()
	notFoundCount[key]++
	if notFoundCount[key] >= negativeCacheHit {
		negativeCacheMu.Lock()
		negativeCache[key] = struct{}{}
		negativeCacheMu.Unlock()
		delete(notFoundCount, key)
	}
	notFoundCountMu.Unlock()
	return false
}

// AddGroup adds a group to the whitelist.
func AddGroup(store *database.Store, groupID string) error {
	if store == nil || groupID == "" {
		return nil
	}
	removeFromNegativeCache(keyPrefixGroup + groupID)
	return store.Set(keyPrefixGroup+groupID, "1")
}

// RemoveGroup removes a group from the whitelist.
func RemoveGroup(store *database.Store, groupID string) error {
	if store == nil || groupID == "" {
		return nil
	}
	return store.Delete(keyPrefixGroup + groupID)
}

// HasUser returns true if the user is in the private-chat user whitelist.
func HasUser(store *database.Store, userID string) bool {
	if store == nil || userID == "" {
		return false
	}
	key := keyPrefixUser + userID
	negativeCacheMu.RLock()
	_, inNegative := negativeCache[key]
	negativeCacheMu.RUnlock()
	if inNegative {
		return false
	}
	_, found, _ := store.Get(key)
	if found {
		notFoundCountMu.Lock()
		delete(notFoundCount, key)
		notFoundCountMu.Unlock()
		return true
	}
	notFoundCountMu.Lock()
	notFoundCount[key]++
	if notFoundCount[key] >= negativeCacheHit {
		negativeCacheMu.Lock()
		negativeCache[key] = struct{}{}
		negativeCacheMu.Unlock()
		delete(notFoundCount, key)
	}
	notFoundCountMu.Unlock()
	return false
}

// AddUser adds a user to the private-chat whitelist (by user ID).
func AddUser(store *database.Store, userID string) error {
	if store == nil || userID == "" {
		return nil
	}
	removeFromNegativeCache(keyPrefixUser + userID)
	return store.Set(keyPrefixUser+userID, "1")
}

// RemoveUser removes a user from the private-chat whitelist.
func RemoveUser(store *database.Store, userID string) error {
	if store == nil || userID == "" {
		return nil
	}
	return store.Delete(keyPrefixUser + userID)
}
