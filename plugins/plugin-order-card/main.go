// Package pluginordercard: orderCard plugin. Register groups (one at a time, 1 or 2 passwords per group),
// respond when message matches group password with +n/-n/=n and 1-min cooldown per user; daily 4am reset value only.
package pluginordercard

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Hafuunano/Core-SkillAction/cache/database"
	skillcore "github.com/Hafuunano/Core-SkillAction/core"
	"github.com/Hafuunano/Core-SkillAction/types"
	"github.com/Hafuunano/Protocol-ConvertTool/protocol"
)

const (
	keyPrefixGroup    = "orderCard:group:"
	keyPrefixData     = "orderCard:data:"
	keyPrefixCooldown = "orderCard:cooldown:"
	cooldownDuration  = time.Minute
	resetHour         = 4
	timezone          = "Asia/Shanghai"
)

// GroupData is the JSON stored at orderCard:data:{gid}.
type GroupData struct {
	Value           int      `json:"value"`
	UpdatedAt       string   `json:"updatedAt"`
	LastUpdaterName string   `json:"lastUpdaterName"`
	Passwords       []string `json:"passwords"`
}

var (
	storeMu       sync.RWMutex
	store         *database.Store
	storeInitOnce sync.Once
)

// Meta for this plugin.
var Meta = types.NewPluginEngine("plugin-order-card-001", "plugin-order-card", "skill", true)
var p = protocol.Engine.WithMeta(Meta)

// SetStore sets the cache/database store. Call from host at startup (e.g. orderCard.SetStore(services.Cache)).
func SetStore(s *database.Store) {
	storeMu.Lock()
	defer storeMu.Unlock()
	store = s
}

func init() {
	// Super admin only: register one group with 1 or 2 passwords.
	p.OnMessage().IsOnlySuperAdmin().Func(handleSetOrderCardRegister)
	// Super admin only: remove one group.
	p.OnMessage().IsOnlySuperAdmin().Func(handleRemoveOrderCardRegister)
	// All messages in registered groups: hit password then +n/-n/=n.
	p.OnMessage().Func(handleOrderCardMessage)
	// Daily 4am reset value only (keep updatedAt, lastUpdaterName, passwords).
	go runDailyReset()
}

func getStore() *database.Store {
	storeInitOnce.Do(func() {
		storeMu.Lock()
		defer storeMu.Unlock()
		if store == nil {
			store = skillcore.DefaultCache()
		}
	})
	storeMu.RLock()
	defer storeMu.RUnlock()
	return store
}

func isGroupRegistered(s *database.Store, gid string) bool {
	if s == nil || gid == "" {
		return false
	}
	_, found, _ := s.Get(keyPrefixGroup + gid)
	return found
}

func handleSetOrderCardRegister(ctx protocol.Context) {
	text := strings.TrimSpace(ctx.PlainText())
	prefix := ctx.CommandPrefix()
	if !strings.HasPrefix(text, prefix+"setOrderCardRegister") {
		return
	}
	parts := strings.Fields(text)
	// {prefix}setOrderCardRegister gid [p1] [p2]
	if len(parts) < 2 {
		_ = ctx.SendPlainMessage("用法: " + prefix + "setOrderCardRegister <群号> <口令1> [口令2]")
		return
	}
	gid := parts[1]
	if gid == "" {
		return
	}
	passwords := parts[2:]
	if len(passwords) == 0 || len(passwords) > 2 {
		_ = ctx.SendPlainMessage("请提供 1 或 2 个口令")
		return
	}
	s := getStore()
	if s == nil {
		_ = ctx.SendPlainMessage("orderCard 未初始化 store")
		return
	}
	if err := s.Set(keyPrefixGroup+gid, "1"); err != nil {
		_ = ctx.SendPlainMessage("注册群组失败")
		return
	}
	data := GroupData{Value: 0, Passwords: passwords}
	raw, _ := json.Marshal(data)
	_ = s.Set(keyPrefixData+gid, string(raw))
	_ = ctx.SendPlainMessage("已注册群组 " + gid + "，口令已设置")
}

func handleRemoveOrderCardRegister(ctx protocol.Context) {
	text := strings.TrimSpace(ctx.PlainText())
	prefix := ctx.CommandPrefix()
	if !strings.HasPrefix(text, prefix+"removeOrderCardRegister") {
		return
	}
	parts := strings.Fields(text)
	if len(parts) < 2 {
		_ = ctx.SendPlainMessage("用法: " + prefix + "removeOrderCardRegister <群号>")
		return
	}
	gid := parts[1]
	s := getStore()
	if s == nil {
		return
	}
	_ = s.Delete(keyPrefixGroup + gid)
	_ = s.Delete(keyPrefixData + gid)
	// Remove cooldown keys for this group
	cooldownKeyPrefix := keyPrefixCooldown + gid + ":"
	for _, e := range s.List() {
		if strings.HasPrefix(e.Key, cooldownKeyPrefix) {
			_ = s.Delete(e.Key)
		}
	}
	_ = ctx.SendPlainMessage("已删除群组 " + gid)
}

var opRegex = regexp.MustCompile(`(\+|-|=)(\d+)`)

func handleOrderCardMessage(ctx protocol.Context) {
	gid := ctx.GroupID()
	if gid == "" || gid == "0" {
		return
	}
	s := getStore()
	if s == nil || !isGroupRegistered(s, gid) {
		return
	}
	raw, found, _ := s.Get(keyPrefixData + gid)
	if !found || raw == "" {
		return
	}
	var data GroupData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return
	}
	plain := strings.TrimSpace(ctx.PlainText())
	// Trigger when message equals a password or starts with password then optional ops (e.g. "mypass" or "mypass +10")
	firstWord := ""
	if f := strings.Fields(plain); len(f) > 0 {
		firstWord = strings.TrimSpace(f[0])
	}
	hitPassword := false
	for _, pw := range data.Passwords {
		if plain == strings.TrimSpace(pw) || firstWord == strings.TrimSpace(pw) {
			hitPassword = true
			break
		}
	}
	if !hitPassword {
		return
	}
	// Apply +n / -n / =n from same message (one cooldown check for the whole message)
	ops := opRegex.FindAllStringSubmatch(plain, -1)
	uid := ctx.UserID()
	now := time.Now()
	cooldownKey := keyPrefixCooldown + gid + ":" + uid
	if len(ops) > 0 {
		lastStr, found, _ := s.Get(cooldownKey)
		if found && lastStr != "" {
			last, _ := time.Parse(time.RFC3339, lastStr)
			if now.Sub(last) < cooldownDuration {
				_ = ctx.SendPlainMessage("每人一分钟内只能更新一次")
				return
			}
		}
		for _, m := range ops {
			if len(m) != 3 {
				continue
			}
			op, numStr := m[1], m[2]
			n, err := strconv.Atoi(numStr)
			if err != nil {
				continue
			}
			switch op {
			case "+":
				data.Value += n
			case "-":
				data.Value -= n
			case "=":
				data.Value = n
			default:
				continue
			}
		}
		data.UpdatedAt = now.Format(time.RFC3339)
		data.LastUpdaterName = ctx.SenderNickname()
		newRaw, _ := json.Marshal(data)
		_ = s.Set(keyPrefixData+gid, string(newRaw))
		_ = s.Set(cooldownKey, now.Format(time.RFC3339))
	}
	// Output current value (and state)
	msg := "当前数值: " + strconv.Itoa(data.Value)
	if data.LastUpdaterName != "" {
		msg += "，上次更新: " + data.LastUpdaterName
	}
	if data.UpdatedAt != "" {
		msg += " @ " + data.UpdatedAt
	}
	_ = ctx.SendPlainMessage(msg)
}

func runDailyReset() {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	for {
		now := time.Now().In(loc)
		next := time.Date(now.Year(), now.Month(), now.Day(), resetHour, 0, 0, 0, loc)
		if !now.Before(next) {
			next = next.AddDate(0, 0, 1)
		}
		time.Sleep(next.Sub(now))
		s := getStore()
		if s == nil {
			continue
		}
		for _, e := range s.List() {
			if !strings.HasPrefix(e.Key, keyPrefixData) {
				continue
			}
			var data GroupData
			if json.Unmarshal([]byte(e.Value), &data) != nil {
				continue
			}
			data.Value = 0
			newRaw, _ := json.Marshal(data)
			_ = s.Set(e.Key, string(newRaw))
		}
	}
}
