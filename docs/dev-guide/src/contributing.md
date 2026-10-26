# 贡献指南

欢迎提交问题反馈、功能建议和 Pull Request。开始编写代码前，请先阅读[从源码构建](building.md)和[代码架构](architecture.md)，定位负责目标行为的模块。

## 创建工作分支

请从最新的 `master` 分支创建职责单一的工作分支。外部贡献者可以先 fork [Zxilly/cjv](https://github.com/Zxilly/cjv)，再基于 `upstream/master` 创建分支：

```bash
git fetch upstream
git switch -c fix-windows-path upstream/master
```

每个分支应聚焦于单一问题修复或功能特性，避免将无关的代码格式化或依赖升级混入同一个分支。

## 编写改动与本地验证

### 代码与测试要求

1. 修改生产代码前，先复现问题或明确目标行为。
2. 在负责该职责的模块中实现修改（参见[代码架构](architecture.md)）。
3. 通过真实生产入口补充或更新回归测试；测试放置位置与命令见[测试与检查](testing.md#回归测试放置位置)。
4. 如果改动涉及平台差异（如 Windows 文件锁、Unix shell 配置文件或鸿蒙代码签名），请在 PR 中说明本机实际验证过的平台，其余平台交由 CI 矩阵检查。

### 文档与多语言同步

- **中英文手册同步**：修改命令行标志、配置字段或用户可见行为时，请在同一提交中同步更新中文（`src/`）和英文（`en/`）mdBook 文档。
- **章节增删与重定向**：新增、移动或删除文档章节时，必须同时更新中英文两份 `SUMMARY.md`，并在对应 `book.toml` 的 `[output.html.redirect]` 中保留旧页面地址重定向（见[文档站](documentation.md)）。
- **前端多语言文案**：新增或修改 `web/` 前端文案后，请运行 `pnpm i18n:extract` 提取词条，并补齐 `web/src/locales/en/messages.po` 中的英文翻译（见[落地页](web.md#多语言翻译与样式)）。

## 提交信息规范

每次提交前，请先查看最近的提交历史，沿用仓库当前的提交信息风格：

```bash
git log --oneline -20
```

提交标题采用 `type(scope): description` 格式，其中 `(scope)` 可省略，`description` 使用简洁的英文祈使句：

```text
fix(component): enforce ownership across file edits
refactor(toolchain): centralize host target associations
docs: simplify project toolchain setup
```

请按逻辑职责拆分提交，保持每个 commit 独立清晰。

## 发起 Pull Request

请在 Pull Request 描述中清晰说明以下内容：

- **背景与触发条件**：什么命令、配置或环境条件触发了该问题。
- **改动方案与结果**：修改了哪个模块，以及修改后 `cjv` 的行为表现。
- **验证情况**：在本地运行了哪些测试套件和平台（文档改动请运行 `npx markdownlint-cli2` 和 `python3 scripts/build-docs.py`）。
- **关联 Issue**：附上相关的 GitHub Issue 链接。

推送分支后请检查 GitHub Actions 运行结果；如果有检查失败，请查看日志并在同一分支补充修复提交。架构文档只记录稳定的模块职责与不变量约束，具体实现细节请留在代码注释和测试中。
