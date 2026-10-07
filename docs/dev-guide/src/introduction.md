# cjv 开发指南

本指南说明 cjv 的构建、代码结构、测试和发布。安装 SDK 与日常使用请看[用户手册](https://cjv.zxilly.dev/book/user-guide/zh-CN/)。

| 目录 | 内容 |
| --- | --- |
| `cmd/cjv/` | Go CLI 入口 |
| `internal/` | 命令、安装生命周期、分发和运行环境 |
| `tests/` | CLI 集成、安装脚本和真实下载测试 |
| `web/` | React 落地页和安装脚本 |
| `docs/` | 用户手册与开发指南，各有中英文源文件 |
| `scripts/` | 平台数据生成与发布辅助脚本 |

先按[构建](building.md)运行本地 CLI，再阅读[架构](architecture.md)定位改动所属模块。[测试与检查](testing.md)列出验证命令，[贡献指南](contributing.md)说明提交约定。
