# cjv 开发指南

本指南说明 cjv 的构建、架构、测试、网站维护和版本发布流程。安装与使用仓颉 SDK 的说明见[用户手册](https://cjv.zxilly.com/book/user-guide/zh-CN/)。

## 仓库目录结构

| 目录 | 内容 |
| --- | --- |
| `cmd/cjv/` | `cjv`、`cjv-init` 及 SDK 代理入口的 Go 主程序 |
| `internal/` | 命令组、安装生命周期、分发下载、持久化事务和运行环境 |
| `tests/` | CLI 端到端集成测试、安装脚本测试和真实下载冒烟测试 |
| `web/` | React 落地页、浏览器平台检测和公开安装脚本（`install.sh`、`install.ps1`） |
| `docs/` | 用户手册与开发指南（`src/` 为中文、`en/` 为英文）以及共享主题 |
| `deployment/` | Cloudflare Worker 与腾讯云 EdgeOne 的边缘函数构建及站点验证工具 |
| `scripts/` | 平台清单生成、CI 范围筛选、文档站构建与鸿蒙模拟器测试辅助脚本 |

## 参与开发的基本流程

1. 先按[从源码构建](building.md)编译本地 CLI，并在隔离的 `CJV_HOME` 目录中试用改动。
2. 阅读[代码架构](architecture.md)定位改动所属的模块，了解并发锁、持久化事务与状态隔离约束。
3. 按[测试与检查](testing.md)运行相应的 Go 包测试、安装脚本测试、前端测试与双语文档检查。
4. 参考[贡献指南](contributing.md)拆分原子提交、编写提交信息并发起 Pull Request。
5. 涉及官网落地页、文档站或正式版本发布时，请查阅[落地页](web.md)、[文档站](documentation.md)和[发布流程](releasing.md)。
