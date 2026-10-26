# cjv 开发指南

本指南介绍如何从源码构建 cjv，说明内部各模块的职责与架构约束，并给出测试、站点维护与发布流程。如果你需要安装仓颉 SDK 并在日常项目中使用 cjv，请阅读 [cjv 用户手册](https://cjv.zxilly.com/book/user-guide/zh-CN/)。

## 仓库目录结构

| 目录 | 包含内容 |
| --- | --- |
| `cmd/cjv/` | `cjv`、`cjv-init` 及 SDK 代理命令的 Go 主入口 |
| `internal/` | CLI 命令组、安装生命周期、分发下载、持久化事务与运行环境配置 |
| `tests/` | CLI 端到端集成测试、安装脚本测试与真实下载冒烟测试 |
| `web/` | React 落地页、浏览器平台识别与公开安装脚本（`install.sh`、`install.ps1`） |
| `docs/` | 用户手册与开发指南的中英文 mdBook 源文件及共享主题 |
| `deployment/` | Cloudflare Worker 与腾讯云 EdgeOne 的边缘函数构建及站点验证工具 |
| `scripts/` | 平台清单生成、CI 范围筛选、文档站构建与鸿蒙模拟器测试脚本 |

## 如何使用本指南

1. 先按[从源码构建](building.md)在本地编译 `cjv` 可执行文件、构建 `mirror` 镜像版或交叉编译鸿蒙版本。
2. 阅读[代码架构](architecture.md)，定位改动所属的业务包，并了解并发锁与事务恢复约束。
3. 按[测试与检查](testing.md)运行对应的 Go 单元测试、安装脚本测试、前端测试与文档校验。
4. 按[贡献指南](contributing.md)整理提交信息并发起 Pull Request。
5. 涉及官网、文档站或版本发布时，请阅读[落地页](web.md)、[文档站](documentation.md)与[发布流程](releasing.md)。
