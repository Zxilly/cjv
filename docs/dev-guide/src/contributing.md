# 贡献指南

欢迎参与 cjv 的开发。无论是修一个 typo、补一段文档,还是实现一个新子命令,流程都一样:fork 仓库、开分支、本地跑通检查、提 PR。这一章把这套流程讲清楚,让你第一次提交就能顺利合并。

## Fork 与分支

cjv 托管在 [github.com/Zxilly/cjv](https://github.com/Zxilly/cjv),主分支是 `master`。在 GitHub 上 fork 到自己的账号,然后克隆下来:

```bash
git clone https://github.com/<your-username>/cjv.git
cd cjv
git remote add upstream https://github.com/Zxilly/cjv.git
```

不要直接在 `master` 上改。每个改动开一个独立分支,从最新的 `upstream/master` 切出来:

```bash
git fetch upstream
git switch -c fix-windows-path upstream/master
```

分支名没有强制规范,起个能说明意图的短名字就行。一个 PR 只做一件事,这样 review 快、回滚也干净。

构建和运行环境见 [building.md](building.md),代码的整体结构见 [architecture.md](architecture.md)。

## 改动前先跑检查

提交之前,在本地把 CI 会跑的几道检查跑一遍,省去来回等流水线。CI 的完整定义在 `.github/workflows/ci.yml`,本地最常用的是下面几条。

Go 代码先构建、跑测试、过 vet:

```bash
go build ./...
go test -count=1 ./...
go vet ./...
```

cjv 有一个 `mirror` 构建标签(给中国大陆镜像源用),改动若涉及下载或自更新逻辑,顺手把带标签的那一份也构建一遍:

```bash
go build -tags=mirror ./...
```

代码风格用 `golangci-lint`,配置在 `.golangci.yml`:

```bash
golangci-lint run
```

测试怎么组织、集成测试怎么跑,见 [testing.md](testing.md);lint 与格式化的细节见 [linting.md](linting.md)。

改了 `go.mod` 或依赖,跑一下 `go mod tidy`,确认没有多余改动;CI 的 `mod-check` job 会用 `git diff --exit-code go.mod go.sum` 卡这一点:

```bash
go mod tidy
git diff --exit-code go.mod go.sum
```

改了 Markdown(包括文档和 README),过一遍 markdownlint,规则在 `.markdownlint-cli2.jsonc`:

```bash
npx --yes markdownlint-cli2
```

落地页(`web/` 目录)的改动有单独的构建和测试,见 [web.md](web.md)。

## 提交信息约定

仓库用 [Conventional Commits](https://www.conventionalcommits.org/) 风格,格式是 `type(scope): 描述`。`scope` 可选。看几条历史提交就能找到手感:

```text
feat(toolchain): support installing from a URL via `toolchain link`
fix(env): derive Windows env-script bin dir from script location
docs: add a bilingual user manual and slim the README
refactor(install): extract lifecycle orchestration
test(windows): ignore registry key close errors
ci: lint Markdown with markdownlint-cli2
chore(deps): bump golang.org/x/crypto to v0.52.0
```

仓库里实际用到的 `type`:

- `feat`:新功能
- `fix`:修 bug
- `docs`:只改文档
- `refactor`:不改外部行为的重构
- `test`:加或改测试
- `ci`:改 CI 配置
- `chore`:杂项(依赖升级、清理等)

`scope` 用来标出改动落在哪个子系统,常见的有 `cli`、`web`、`install`、`toolchain`、`env`、`config`、`ci` 等;一个改动跨两个范围时也会写成 `fix(proxy,cli): ...` 这样。`scope` 不是固定枚举,贴合改动就行,不确定就省略。

描述用祈使句、英文小写开头、不加句号,概括「这个提交做了什么」。提交信息没有强制的 commit hook 校验,但请遵照上面的风格,保持历史可读。

## 提交 PR

把分支推到你的 fork,然后开 PR,目标分支是 `Zxilly/cjv` 的 `master`:

```bash
git push -u origin fix-windows-path
```

也可以用 `gh` 命令行直接开:

```bash
gh pr create --repo Zxilly/cjv --base master
```

PR 描述里说清楚改了什么、为什么改;如果对应某个 issue,带上引用。开 PR 后 CI 会自动跑全套检查:Go 在 Linux、macOS、Windows 的 amd64 与 arm64 上构建并测试(含 `-race`),`golangci-lint`、markdownlint、`go mod tidy` 校验、`govulncheck` 漏洞扫描、跨平台交叉构建,以及落地页的构建与浏览器集成测试。安装脚本还有一组覆盖多种 shell 的集成测试。这些 job 的细节见 [ci.md](ci.md)。

所有检查都得是绿的才会合并。如果某个 job 红了,点进去看日志,在本地复现并修掉,再推一次;同一个分支后续的 push 会自动触发重跑,不用重开 PR。Review 提出的修改意见,直接在分支上追加提交即可。

## 文档改动要中英同步

文档(`docs/user-guide` 和 `docs/dev-guide`)每本都维护两份独立源:`src/` 是简体中文(源语言),`en/` 是英文。两边是各自独立的 Markdown 文件,没有自动翻译,改了中文要手动把英文同步上。

只改错别字、调整措辞这类小改动,英文未必受影响;但凡增删段落、改了完整句子,就把 `en/` 下同名文件的对应部分也改掉,否则英文站会停在旧内容。新增或删除一章时,记得在 `src/` 和 `en/` 两边的章节文件与各自的 `SUMMARY.md` 都改。

代码块在两份源里各写各的:中文版 `src/` 的注释和示例输出用中文,英文版 `en/` 用英文;命令、路径、标识符两边保持一致。改完本地各跑一遍中英两版,确认都能正常渲染(英文用 `MDBOOK_BOOK__SRC=en` 把源目录切到 `en/`):

```bash
cd docs/dev-guide
mdbook build                                              # 中文(src/)
MDBOOK_BOOK__SRC=en MDBOOK_BOOK__LANGUAGE=en mdbook build  # 英文(en/)
```

文档站的目录布局、语言切换器和 CI 构建细节见 [documentation.md](documentation.md) 和 `.github/workflows/pages.yml`。中文源和英文改动放在同一个 PR 里提交,reviewer 能对照着看,英文站也跟着这次合并一起更新。

只想读用户视角的功能说明时,用户手册在 [https://cjv.zxilly.dev/book/user-guide/zh-CN/](https://cjv.zxilly.dev/book/user-guide/zh-CN/)。

## 遇到问题

不确定某个改动该怎么做、或者想先讨论方向,欢迎先开一个 issue 聊聊,再动手写代码。小修小补可以直接提 PR。无论哪种,我们都很乐意帮你把改动推进到合并。
