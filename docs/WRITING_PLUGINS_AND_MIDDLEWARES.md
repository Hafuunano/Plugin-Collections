# 插件与中间件编写规范

本文约定 Plugin-Collections 中插件与中间件的契约。通过显式 import 与注册列表/链发现并调用它们；宿主侧行为见 其他端的调试。

---

## 一、插件契约

### 1.1 依赖与归属

- **协议依赖**：插件依赖 `github.com/Hafuunano/Protocol-ConvertTool`，使用其 `protocol` 包（`Context`、`Message`、`Segment` 等）。
- **可选依赖 Core 的 types**：若在插件内直接定义元信息（见 1.5），可 import `github.com/Hafuunano/Core-SkillAction/types`，使用 `types.NewPluginEngine` 或 `types.PluginEngine`。
- **不直接依赖宿主**：不要 import 宿主（Lucy）。若需 DB/Cache/Timer，通过 Protocol 对 Context 的扩展（如 `Services()`）由宿主注入后使用。
- **归属**：插件代码放在本仓库 `plugins/plugin-<name>/` 下（如 `plugins/plugin-orderCard/`）。

### 1.2 包与目录结构

- 每个插件一个子包，包名与目录对应（如目录 `plugin-orderCard/` 对应包名 `pluginordercard`）。
- 可拆成多文件（如 `main.go` 放入口，`handler.go`、`store.go` 放逻辑）；对外仅暴露约定入口。

目录示例：

```
Plugin-Collections/
  plugins/
    plugin-orderCard/
      main.go      # 入口 func Plugin(ctx protocol.Context)
      model.go     # 可选
      store.go     # 可选
```

### 1.3 入口与生命周期（约定签名）

- **必须**：实现入口函数  
  `func Plugin(ctx protocol.Context)`  
  宿主在每条消息/事件上构造 `protocol.Context` 后，对该插件的调用即执行此函数。
- **可选**：进程级初始化  
  `func Init()`  
  无参、无返回值；宿主在启动时对每个插件至多调用一次，用于加载配置、初始化全局状态等。不需要时可省略。
- **配置**：推荐通过宿主（如 Context 或 Core 的 `types.PluginEngine`）传入；插件内部也可自读文件，但更推荐宿主注入或 Context 扩展。

### 1.4 与 Core-SkillAction 配置对应

- Core-SkillAction 的 `types.PluginEngine` 提供 `PluginID`、`PluginName`、`PluginType`、`PluginIsDefaultOn`。
- 约定：插件的「逻辑名」与配置中的 `PluginName` 对应，便于宿主按配置开关（是否加入插件列表）。

### 1.5 插件内定义元信息（Plugin Meta）

- 推荐在插件包内**直接定义**本插件的元信息，便于在插件内部直接使用（如 `Meta.PluginID`、`Meta.PluginName`），无需从中心 config 读取。
- 依赖：`github.com/Hafuunano/Core-SkillAction/types`，使用 `types.NewPluginEngine(id, name, pluginType string, isDefaultOn bool)` 或 `types.PluginEngine{...}`。
- 约定：包级变量命名为 `Meta` 或 `Plugin`；若入口函数已命名为 `Plugin(ctx)`，则元信息变量建议用 `Meta`，避免同名。

**示例一（一行定义）**：见 `plugins/plugin-hello/main.go`、`plugins/plugin-echo/main.go`：

```go
import "github.com/Hafuunano/Core-SkillAction/types"

// Meta is this plugin's metadata; use Meta.PluginID, Meta.PluginName, etc. inside this package.
var Meta = types.NewPluginEngine("plugin-hello-001", "plugin-hello", "skill", true)
```

**示例二（结构体字面量）**：

```go
var Meta = types.PluginEngine{
	PluginID:          "my-plugin-001",
	PluginName:        "my-plugin",
	PluginType:        "skill",
	PluginIsDefaultOn: true,
}
```

在插件内使用：`id := Meta.PluginID`、`name := Meta.PluginName`、`typ := Meta.PluginType`、`on := Meta.PluginIsDefaultOn`。

### 1.5 插件内行为约定

- 仅通过 `ctx` 与用户/环境交互：`ctx.PlainText()`、`ctx.Reply()`、`ctx.Send()`、`ctx.UserID()`、`ctx.GroupID()`、`ctx.IsSuperAdmin()` 等。
- 不阻塞事件循环：耗时操作建议异步或交给 Context 扩展（如 Timer/Cache）。
- 注释统一使用英文（与项目规范一致）。

### 1.7 插件示例

以下为一个最小可用的插件示例：收到「hello」时回复一条消息，并展示可选 `Init()` 的用法。

**目录**：`plugins/plugin-hello/`

**main.go**（入口 + 简单逻辑）：

```go
package pluginhello

import (
	"github.com/Hafuunano/Protocol-ConvertTool/protocol"
)

// triggerWord is loaded in Init(); default is "hello".
var triggerWord string = "hello"

// Init is optional. Host may call it once at startup.
func Init() {
	// e.g. load from config file or env; here we keep default.
	triggerWord = "hello"
}

// Plugin is the required entry. Host calls it for each message with a protocol.Context.
func Plugin(ctx protocol.Context) {
	text := ctx.PlainText()
	if text != triggerWord {
		return
	}
	nick := ctx.SenderNickname()
	if nick == "" {
		nick = "user"
	}
	// Reply with a simple text segment.
	_ = ctx.Reply(protocol.Message{
		{Type: protocol.SegmentTypeText, Data: map[string]any{"text": "Hello, " + nick + "!"}},
	})
}
```

**多文件时的拆分示例**（可选）：将业务逻辑放到 `handler.go`，入口只做转发：

```go
// main.go - only expose entry
package pluginhello

import "github.com/Hafuunano/Protocol-ConvertTool/protocol"

func Plugin(ctx protocol.Context) {
	handleHello(ctx)
}
```

```go
// handler.go - business logic
package pluginhello

import "github.com/Hafuunano/Protocol-ConvertTool/protocol"

func handleHello(ctx protocol.Context) {
	text := ctx.PlainText()
	if text != "hello" {
		return
	}
	_ = ctx.Reply(protocol.Message{
		{Type: protocol.SegmentTypeText, Data: map[string]any{"text": "Hello!"}},
	})
}
```

---

## 二、中间件契约

### 2.1 职责与位置

- **职责**：在「收到事件 → 构造 Context → 调用插件」之间执行过滤/增强（白名单、OnlyToMe、日志、限速等），插件只关心 `protocol.Context`。
- **位置**：中间件可放在本仓库 `middlewares/<name>/`（如 `middlewares/whitelist/`），与插件同仓、同协议依赖；也可放在宿主内仅供宿主使用。放在本仓便于复用与版本一致。

### 2.2 签名与链式约定

- **Handler**（在 Protocol-ConvertTool 的 `protocol` 包中定义）：  
  `type Handler = func(ctx protocol.Context)`  
  表示「处理一条消息」的单元。
- **中间件**：  
  `type Middleware = func(next Handler) Handler`  
  接收下一个 Handler，返回新的 Handler；在返回的 Handler 内先执行本中间件逻辑，再按需调用 `next(ctx)`。

示例（白名单：未通过则不调用 next）：

```go
package whitelist

import "github.com/Hafuunano/Protocol-ConvertTool/protocol"

func Whitelist(next protocol.Handler) protocol.Handler {
	return func(ctx protocol.Context) {
		if !isAllowed(ctx.UserID(), ctx.GroupID()) {
			return
		}
		next(ctx)
	}
}
```

- **链的组装**：由**宿主**在启动时按配置顺序将多个 Middleware 与最终的「插件列表调用」组合成一条链，例如 `h := mw1(mw2(mw3(pluginDispatcher)))`，对每条消息执行 `h(ctx)`。

### 2.3 与 Core-SkillAction 配置对应

- Core 的 `types.MiddlewareEngine` 提供 `MiddlewareID`、`MiddlewareName`。
- 约定：中间件的「逻辑名」与配置中的 `MiddlewareName` 对应，宿主据此决定是否加入链及顺序。

### 2.4 中间件依赖

- 与插件相同：仅依赖 `github.com/Hafuunano/Protocol-ConvertTool` 的 `protocol` 包。
- 若需配置（如白名单列表），可通过可选 `Init()` 或由宿主在构造中间件时注入（如闭包），避免中间件直接依赖 Core；也可通过 Context 扩展获取配置。

---

## 三、速查表

| 类型     | 入口/签名                                              | 依赖                         | 配置对应                      |
|----------|--------------------------------------------------------|------------------------------|-------------------------------|
| 插件     | `func Plugin(ctx protocol.Context)`；可选 `func Init()` | Protocol-ConvertTool `protocol` | `types.PluginEngine.PluginName`   |
| 中间件   | `func(next Handler) Handler`，内部调用 `next(ctx)`     | Protocol-ConvertTool `protocol` | `types.MiddlewareEngine.MiddlewareName` |
| Handler  | `func(ctx protocol.Context)`                           | —                            | 在 `protocol` 中定义          |

按此契约编写的插件与中间件可与四仓架构（Lucy、Protocol-ConvertTool、Core-SkillAction、Plugin-Collections）及宿主的显式注册与中间件链配合使用。
