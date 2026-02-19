package pluginordercard

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Hafuunano/UniTransfer/protocol"
)

const (
	ttlSeconds = 60
	setPrefix  = "设置 "
	delPrefix  = "删除查询池"
)

var (
	storeOnce sync.Once
	poolStore *store
	storeErr  error
	ttlMu     sync.RWMutex
	ttlExpiry = make(map[string]time.Time) // key: groupID_userID
)

func getStore() (*store, error) {
	storeOnce.Do(func() {
		dbPath := os.Getenv("ORDERCARD_DB_PATH")
		if dbPath == "" {
			dbPath = "ordercard_pools.db"
		}
		poolStore, storeErr = openStore(dbPath)
	})
	return poolStore, storeErr
}

func ttlKey(groupID, userID string) string {
	return groupID + "_" + userID
}

// checkAndConsumeTTL returns true if the user is allowed to perform +1/-1/= (and consumes the TTL). False if rate limited.
func checkAndConsumeTTL(groupID, userID string) bool {
	key := ttlKey(groupID, userID)
	ttlMu.Lock()
	defer ttlMu.Unlock()
	exp, ok := ttlExpiry[key]
	if ok && time.Now().Before(exp) {
		return false
	}
	ttlExpiry[key] = time.Now().Add(ttlSeconds * time.Second)
	return true
}

func replyText(ctx protocol.Context, text string) {
	msg := protocol.Message{
		protocol.Segment{Type: protocol.SegmentTypeText, Data: map[string]any{"text": text}},
	}
	_ = ctx.Send(msg)
}

// Plugin is the entry point called by the bot host for each message.
func Plugin(ctx protocol.Context) {
	groupID := ctx.GroupID()
	if groupID == "" {
		return
	}
	text := strings.TrimSpace(ctx.PlainText())
	if text == "" {
		return
	}

	store, err := getStore()
	if err != nil {
		replyText(ctx, "查询池服务暂不可用")
		return
	}

	// 设置 指令 [群组]
	if strings.HasPrefix(text, setPrefix) {
		rest := strings.TrimSpace(text[len(setPrefix):])
		if rest == "" {
			replyText(ctx, "用法：设置 <指令> [群组ID]")
			return
		}
		parts := strings.Fields(rest)
		instruction := parts[0]
		targetGroup := groupID
		if len(parts) >= 2 {
			if _, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				targetGroup = parts[1]
			}
		}
		if err := store.Create(targetGroup, instruction); err != nil {
			replyText(ctx, "设置失败")
			return
		}
		replyText(ctx, "已为群 "+targetGroup+" 设置指令：「"+instruction+"」")
		return
	}

	// 删除查询池 [群组]
	if strings.HasPrefix(text, delPrefix) {
		rest := strings.TrimSpace(text[len(delPrefix):])
		targetGroup := groupID
		if rest != "" {
			if _, err := strconv.ParseInt(rest, 10, 64); err == nil {
				targetGroup = rest
			}
		}
		if err := store.Delete(targetGroup); err != nil {
			replyText(ctx, "删除失败")
			return
		}
		replyText(ctx, "已删除群 "+targetGroup+" 的查询池")
		return
	}

	// Need current group's instruction for the rest
	instruction, value, lastResetAt, err := store.Get(groupID)
	if err != nil {
		replyText(ctx, "查询失败")
		return
	}
	if instruction == "" {
		return // no pool for this group, ignore
	}

	// 查询：仅发指令
	if text == instruction {
		replyText(ctx, "当前值："+strconv.Itoa(value))
		return
	}

	// 指令+1
	if text == instruction+"+1" || text == instruction+" +1" {
		if !checkAndConsumeTTL(groupID, ctx.UserID()) {
			replyText(ctx, "请稍后再试（每分钟限一次）")
			return
		}
		value++
		if err := store.UpdateValue(groupID, value, lastResetAt); err != nil {
			replyText(ctx, "操作失败")
			return
		}
		replyText(ctx, "当前值："+strconv.Itoa(value))
		return
	}

	// 指令-1
	if text == instruction+"-1" || text == instruction+" -1" {
		if !checkAndConsumeTTL(groupID, ctx.UserID()) {
			replyText(ctx, "请稍后再试（每分钟限一次）")
			return
		}
		if value > 0 {
			value--
		}
		if err := store.UpdateValue(groupID, value, lastResetAt); err != nil {
			replyText(ctx, "操作失败")
			return
		}
		replyText(ctx, "当前值："+strconv.Itoa(value))
		return
	}

	// 指令=数字
	if strings.HasPrefix(text, instruction+"=") || strings.HasPrefix(text, instruction+" =") {
		numStr := strings.TrimPrefix(text, instruction+"=")
		numStr = strings.TrimPrefix(numStr, instruction+" =")
		numStr = strings.TrimSpace(numStr)
		n, err := strconv.Atoi(numStr)
		if err != nil || n < 0 {
			replyText(ctx, "请输入非负整数")
			return
		}
		if !checkAndConsumeTTL(groupID, ctx.UserID()) {
			replyText(ctx, "请稍后再试（每分钟限一次）")
			return
		}
		if err := store.SetValue(groupID, n); err != nil {
			replyText(ctx, "操作失败")
			return
		}
		replyText(ctx, "当前值："+strconv.Itoa(n))
		return
	}
}
