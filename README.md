## Plugin-Collections

通过显式导入和 `[]func(protocol.Context)` 列表加载插件；中间件会在插件分发前组合成调用链。

- **协议依赖**：插件与中间件仅依赖 `github.com/Hafuunano/Protocol-ConvertTool` 及其 `protocol` 包（包括 `Context`、`Message`、`Handler`、`Middleware`）。
- **插件入口**：实现 `func Plugin(ctx protocol.Context)`；可选的初始化函数为 `func Init()`，用于启动时初始化。
- **中间件**：形式为 `func(next protocol.Handler) protocol.Handler`，需调用 `next(ctx)` 以继续处理链。

详见 [docs/WRITING_PLUGINS_AND_MIDDLEWARES.md](docs/WRITING_PLUGINS_AND_MIDDLEWARES.md)，了解完整编写约定（包含目录结构、函数签名及与 Core-SkillAction 配置的映射）。

## License

AGPL-3.0