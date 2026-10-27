# 贡献指南

从最新的 `master` 创建工作分支。外部贡献者可先 fork [Zxilly/cjv](https://github.com/Zxilly/cjv)，将源仓库配置为 `upstream`，再向源仓库的 `master` 提交 Pull Request：

```bash
git fetch upstream
git switch -c fix-windows-path upstream/master
```

## 改动与本地验证

1. **先复现再修复** — 修复问题前先复现故障，并在[代码架构](architecture.md)中定位负责该行为的模块。
2. **通过生产入口测试** — 在负责该行为的模块中通过真实生产入口补充或更新回归测试；测试位置和验证命令见[测试与检查](testing.md#回归测试放在哪里)。
3. **说明平台覆盖情况** — 改动涉及平台差异（如 Windows 文件锁、Unix shell 配置文件或鸿蒙自签名）时，请在 PR 中说明本机实际验证过的平台，其余平台交由 CI 矩阵检查。
4. **同步中英文文档与前端翻译** — 文档的中英文内容（`src/` 与 `en/`）请在同一提交中同步更新；新增、移动或删除章节时需同时更新两份 `SUMMARY.md` 并在 `book.toml` 中保留旧页面重定向（见[文档站](documentation.md)）。`web/` 前端文案使用 Lingui（`pnpm i18n:extract`），修改后需补齐 `web/src/locales/en/messages.po` 中的英文翻译（见[落地页](web.md#翻译与样式)）。

## 原子提交与提交信息

每个提交应包含一个独立的逻辑改动。无关的格式化、重构和依赖升级应分别提交。

每次提交前，请先查看最近的提交历史并沿用仓库当前风格：

```bash
git log --oneline -20
```

提交标题采用 `type(scope): description` 格式，其中 `scope` 可省略，描述使用简洁的英文祈使句，例如：

```text
fix(component): enforce ownership across file edits
refactor(toolchain): centralize host target associations
docs: simplify project toolchain setup
```

## Pull request

发起 Pull Request 时，请在描述中说明：

- **触发条件与背景** — 什么命令、配置或边界条件触发了该问题。
- **改动后的行为** — 修改了哪些模块以及现在的预期行为。
- **验证结果** — 在本地运行了哪些测试套件和平台。
- **关联 Issue** — 附上相关的 Issue 编号或链接。

推送分支后请留意 CI 运行状态；若出现失败，查看日志并在原分支继续补充修复提交。

涉及命令行为、参数选项或配置格式的变化时，请同时更新[用户手册](https://cjv.zxilly.com/book/user-guide/zh-CN/)。开发指南中的[代码架构](architecture.md)只记录稳定的模块职责和约束，具体实现细节留在代码注释和测试中。
