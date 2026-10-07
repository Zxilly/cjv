# 贡献指南

从最新 `master` 创建分支，提交一个范围明确的改动。外部贡献者可以先 fork [Zxilly/cjv](https://github.com/Zxilly/cjv)，再向主仓库的 `master` 提交 PR。

```bash
git fetch upstream
git switch -c fix-windows-path upstream/master
```

## 改动与验证

先复现问题，再在负责该行为的模块中修复。测试通过生产入口验证结果；测试位置和命令见[测试与检查](testing.md)。涉及平台差异时说明实际验证的平台，其他平台交由 CI 检查。

文档的中英文内容在同一提交中同步，增删章节需同时更新目录与旧页面重定向。前端文案使用 Lingui 并补齐英文翻译。

## 提交信息

每次提交前查看最近历史，沿用仓库当前风格：

```bash
git log --oneline -20
```

当前使用 `type(scope): description`，scope 可省略，描述用简洁英文，例如：

```text
fix(component): enforce ownership across file edits
refactor(toolchain): centralize host target associations
docs: simplify project toolchain setup
```

按职责拆分提交，避免把无关格式化或依赖升级混入修复。

## Pull request

PR 描述说明触发条件、改动后的行为和验证结果；有关联 issue 时附上引用。推送后检查 CI，失败时查看日志并补充修复。Review 修改继续提交到原分支。

涉及命令行为或配置格式的变化，应同时更新用户手册。架构文档只记录稳定的职责和约束，具体实现留在代码和测试中。
