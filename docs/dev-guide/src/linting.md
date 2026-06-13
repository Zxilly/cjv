# 代码检查与格式化

cjv 在三种语言上做检查：Go 代码、Markdown 文档、落地页的 TypeScript。每一类都有 CI job 守门(见 [ci.md](ci.md))，本章说明这些检查在跑什么，以及怎么在本地提前跑一遍，避免推上去才发现红叉。

## Go 格式化

Go 代码统一用 `gofmt` 格式化。提交前格式化整棵树：

```bash
gofmt -w .
```

想先看看哪些文件没格式化、又不直接改动，用 `-l` 只列文件名：

```bash
gofmt -l .
```

没有输出就说明全部已经格式化。CI 没有专门的 gofmt 差异检查 job，格式化靠开发者自觉和编辑器的保存即格式化。绝大多数 Go 编辑器集成都默认在保存时跑 `gofmt`(或等价的 `goimports`)，开着就基本不用手动管。

## go vet

`go vet` 是 Go 自带的静态检查，能发现 `Printf` 格式串对不上参数、复制了带锁的结构体之类编译器不报但很可能是 bug 的写法。本地跑：

```bash
go vet ./...
```

cjv 有一个 `mirror` 构建 tag(给国内镜像源用的变体)，那部分代码只有带 tag 才会被编译，所以也要单独 vet 一遍：

```bash
go vet -tags=mirror ./...
```

这两条对应 CI 里 `go-test` job 的 `Run go vet` 和 `Run go vet (mirror)` 两步。

## golangci-lint

更全面的 Go 检查交给 `golangci-lint`。配置在 [`.golangci.yml`](https://github.com/Zxilly/cjv/blob/master/.golangci.yml)，用的是 v2 配置格式。本地安装见 [官方文档](https://golangci-lint.run/welcome/install/)，装好后在仓库根目录跑：

```bash
golangci-lint run
```

配置里 `default: standard` 启用 golangci-lint 的标准 linter 集合：`errcheck`、`govet`、`ineffassign`、`staticcheck`、`unused`。在此之上额外开了三个：

- `errorlint`：检查 Go 1.13 错误包装的用法，比如该用 `errors.Is` 的地方用了 `==`。
- `copyloopvar`：检查循环变量被复制的多余写法(Go 1.22 改了循环变量语义后这类代码可以简化)。
- `errname`：要求哨兵错误以 `Err` 前缀命名、错误类型以 `Error` 后缀命名。

有两处例外写在配置里。`errcheck` 放过 `progressbar` 进度条的 `Add64` 和 `Finish`，这两个的返回错误只是装饰性的，处理了也没意义。测试文件(`_test.go`)整体放宽 `errcheck`，测试里大量 `defer f.Close()` 之类不检查错误是常态。

`golangci-lint` 能自动修一部分问题：

```bash
golangci-lint run --fix
```

CI 的 `lint` job 用 `golangci-lint-action` 跑，版本固定为 `latest`。本地装的版本如果偏旧，可能少报或多报几条，以 CI 结果为准。

## Markdown

仓库里的 Markdown(两个 README 和 mdBook 手册)用 `markdownlint-cli2` 检查。配置在仓库根目录的 [`.markdownlint-cli2.jsonc`](https://github.com/Zxilly/cjv/blob/master/.markdownlint-cli2.jsonc)。在仓库根目录跑：

```bash
npx markdownlint-cli2
```

不用先装，`npx` 会临时拉一份。检查范围由配置里的 `globs` 定义，是 `README.md`、`README.EN.md` 和 `docs/**/*.md`。两份 `SUMMARY.md` 被 `ignores` 排除，因为 mdBook 用多个一级标题当分组标题，会触发 MD025(单文档只允许一个一级标题)，那是结构需要，不是写作问题。

配置里关掉和调整了几条规则，都是为中文文档服务的：

- `MD013`(行长度)关掉。中文整段不换行，行长度限制对 CJK 正文、表格、长 URL 没意义。
- `MD060`(表格列对齐)关掉。这条要求竖线按字符数对齐，但中文是全角字符、占两个字符宽，按字符数对齐的话视觉上反而不齐。
- `MD024`(重复标题)设为 `siblings_only`，允许不同父标题下出现同名小节，比如多个章节里都有「示例」。

CI 的 `markdown-lint` job 跑 `npx --yes markdownlint-cli2@0.22.1`，版本号是钉死的。本地直接 `npx markdownlint-cli2` 拿到的是最新版，规则集偶尔会有出入，要完全对齐 CI 就显式带上版本：

```bash
npx --yes markdownlint-cli2@0.22.1
```

## 落地页 TypeScript

落地页(`web/`，见 [web.md](web.md))的类型检查由 TypeScript 编译器 `tsc` 完成，它被串在构建里。`web/package.json` 的 `build` 脚本是 `tsc -b && vite build`，所以一次构建会先做类型检查再打包，类型不过编译就直接失败。在 `web/` 目录里跑：

```bash
cd web
pnpm install
pnpm build
```

CI 的 `web-build` job 就是这么做的：`pnpm install --frozen-lockfile` 之后 `pnpm build`。落地页没有单独的 ESLint job，类型安全靠 `tsc`，行为正确性靠测试(见 [testing.md](testing.md))。

## 依赖检查

两件和 Go 模块相关的检查也在 CI 里，本地能复现。

`go mod tidy` 用来确保 `go.mod` 和 `go.sum` 干净，没有多余或缺失的依赖。CI 的 `mod-check` job 跑完 `go mod tidy` 后用 `git diff --exit-code` 校验文件没被改动，改动了就说明提交的不是 tidy 状态。本地照着跑：

```bash
go mod tidy
git diff --exit-code go.mod go.sum
```

同一个 job 还跑 `go mod verify`，校验本地模块缓存里的依赖和它们下载时的校验和一致，没被篡改：

```bash
go mod verify
```

## 漏洞扫描

`govulncheck` 扫描代码实际调用到的依赖里有没有已知漏洞。它只报真正会走到的调用路径，所以噪音很低。CI 的 `vuln-check` job 先 `go install` 再扫描。本地装一次：

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

之后扫描整棵树：

```bash
govulncheck ./...
```

## 本地一键自查

推送前想把上面这些一次跑完，在仓库根目录依次执行：

```bash
gofmt -l .
go vet ./...
go vet -tags=mirror ./...
golangci-lint run
go mod tidy && git diff --exit-code go.mod go.sum
go mod verify
govulncheck ./...
npx --yes markdownlint-cli2@0.22.1
```

落地页相关的改动再补上一段：

```bash
cd web && pnpm install && pnpm build
```

这串命令对应 CI 里 `go-test`、`lint`、`mod-check`、`vuln-check`、`markdown-lint`、`web-build` 几个 job 的检查部分。全绿基本就能保证这些 job 不会因为检查或格式化挂掉，剩下的留给测试(见 [testing.md](testing.md))。
