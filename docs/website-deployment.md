# 网站部署

目标部署方式：`cjv.zxilly.dev` 由 GitHub Pages 提供服务，`cjv.zxilly.com` 由腾讯云 EdgeOne Makers 提供服务。两个站点使用同一次 GitHub Actions 构建产物，包含首页、双语用户及开发指南、安装脚本和 `/dl/` 下的安装器。

## EdgeOne Makers

2026-10-20 首次部署记录：项目 `cjv-web`（`makers-9riya3l0zoze`），生产部署 `dpqt1qkz6d2l`。首次上传包由本地网站构建、mdBook 0.5.3 生成的四份文档和通过官方校验和验证的 v0.4.1 安装器组成。自定义域名的 CNAME 目标为 `cjv.zxilly.com.pages.dnsoe4.com`，DNS 已添加，自动验证的免费证书已部署。正式 `.com` 首页及四份文档均返回 200；线上两份安装脚本和 14 个安装器逐一下载后，SHA-256 均与上传产物一致。

区域调整后使用项目 `cjv-web-global`（`makers-nbvih0bn28ch`），最新生产部署为 `dp8y8ezpjjzq`，代码、脚本及双语文档统一使用 `.com`。GitHub 仓库主页也已更新为 `https://cjv.zxilly.com/`。控制台确认区域为 **全球可用区（含中国大陆）**。原中国大陆项目 `cjv-web` 及部署 `dpjy0k9ak9gn` 保留用于回退。迁移自定义域名需要先解除原项目绑定；平台释放域名资源约需 5 分钟，等待期间暂时关闭 Cloudflare 地区跳转，绑定完成且 HTTPS 验证通过后再恢复。

全球项目预览环境的两份安装脚本和 14 个安装器已逐一验证 HTTP 200 和 SHA-256；四份文档返回 200，缺失页面及下载路径返回 404。

迁移后自定义域名的 CNAME 目标未变。公共 DNS 返回的两个全球节点均已通过正式域名 TLS 校验并返回首页 200；四份文档、两份脚本和 404 行为复验通过，脚本校验和与上传产物一致。Cloudflare 地区规则已恢复启用，官方 Trace 确认中国大陆文档请求返回 302。本机解析器曾缓存迁移期间的 NXDOMAIN，因此使用公共 DNS 实际返回的 IP 配合 `curl --resolve` 完成校验，未跳过 TLS 验证。

控制台最终确认自定义域名“已生效”、HTTPS 证书“已部署”。本机常规 DNS 解析随后也已恢复，直接请求 `https://cjv.zxilly.com/` 返回 200。GitHub Actions 使用 `EDGEONE_PROJECT_NAME` 和加密 Secret `EDGEONE_API_TOKEN` 自动部署全球项目。

使用 `web/public/404.html` 和 `deployment/edgeone/routes.json` 明确静态文件优先及 404 回退后，缺失页面和安装器路径均已验证返回 404。Cloudflare 官方 Trace 的 8 组检查确认中国大陆首页（包括 `/index.html`）、文档返回 302，境外请求、安装脚本、下载路径和其他主机均不匹配。实际境外出口验证 `.dev` 首页与脚本正常返回 200；本次未从真实中国大陆出口测量响应。

在腾讯云控制台选择 **EdgeOne → Makers → 创建项目 → 直接上传**，当前生产项目名为 `cjv-web-global`。使用直接上传而非重新在 Makers 构建，可以复用 Actions 下载的发布归档和生成的 mdBook 文档。

加速区域需要在创建项目时选择：

本项目选用 **全球可用区（含中国大陆）**；域名备案状态由域名持有人确认。现有项目的控制台设置和官方 SDK 更新接口未提供区域修改选项；区域调整使用新项目承载相同产物，再迁移自定义域名。

| 区域 | 域名要求 |
| --- | --- |
| 中国大陆可用区 | 域名已完成工信部备案 |
| 全球可用区（含中国大陆） | 域名已完成工信部备案 |
| 全球可用区（不含中国大陆） | 不要求备案，使用境外节点 |

首次上传时，从成功的 Pages 工作流下载 `github-pages` artifact，解开外层 ZIP 后，再解开其中的 `artifact.tar`。将解出的文件打包上传，确保 `index.html` 在上传包根目录，与 `book/`、`dl/`、`install.sh`、`install.ps1` 同级。不要上传源码目录或只有 Vite 首页的本地产物。

在新项目的 **域名管理 → 添加自定义域名** 中填写 `cjv.zxilly.com`，关联生产环境。按照控制台实际给出的目标配置 DNS：

| DNS 区域 | 类型 | 名称 | 内容 | Cloudflare 代理 |
| --- | --- | --- | --- | --- |
| `zxilly.com` | CNAME | `cjv` | Makers 为自定义域名分配的 CNAME | 关闭，仅 DNS |

如控制台另行要求 TXT 所有权验证，按其提示添加。随后在 Makers 完成 HTTPS 配置，确认域名和证书均已生效。Makers 的临时预览地址不能代替生产域名：中国大陆访问系统预览链接有时效限制。

在 GitHub 仓库的 **Settings → Secrets and variables → Actions** 中配置：

| 类型 | 名称 | 内容 |
| --- | --- | --- |
| Repository variable | `EDGEONE_PROJECT_NAME` | `cjv-web-global` |
| Repository secret | `EDGEONE_API_TOKEN` | 腾讯云中国站 Makers API Token |

Token 保存到 GitHub Actions 加密 Secret，不写入源码或文档。先创建项目并确认区域，再配置变量；CLI 在项目不存在时可能自动创建项目。

`.github/workflows/pages.yml` 在推送 `master` 或手动触发时发布 GitHub Pages，然后使用同一份 `github-pages` artifact 部署 Makers。两处都会包含首页、双语文档、安装脚本和安装器。腾讯部署为必需任务，缺少项目变量或 Token 时会明确失败。发布版本时，`release.yml` 同样向该复用工作流传递 Token。

工作流固定使用 `edgeone@1.6.44`，命令为：

```sh
npx --yes edgeone@1.6.44 login --site china --token "$EDGEONE_API_TOKEN"
npx --yes edgeone@1.6.44 makers deploy ./site \
  --name "$EDGEONE_PROJECT_NAME" --env production --skip-ai-gateway-sync
```

登录命令的 `--site china` 表示腾讯云中国站账户入口，与项目选择的加速区域不同。部署命令中的同名选项主要用于匿名部署，CI 应先显式登录中国站，再部署已创建的项目。`--skip-ai-gateway-sync` 跳过此静态网站不需要的 AI Gateway 配置同步。这里的 `./site` 为解包后的完整网站目录。

此静态站点不添加全局 SPA rewrite。`web/public/404.html` 会复制到产物根目录；Makers 部署任务另将 `deployment/edgeone/routes.json` 复制为产物中的 `.edgeone/routes.json`，明确先匹配静态文件，再以 404 返回错误页。实测仅添加 `404.html` 仍会被自动识别为 SPA 并回退到首页，因此需要显式路由。此配置仅加入 Makers 上传包，GitHub Pages 继续使用原始网站 artifact。

## 中国大陆访客重定向

重定向配置在 `zxilly.dev` 的 Cloudflare 入口。先确认 `.com` 完整站点可用，再将 `cjv.zxilly.dev → zxilly.github.io` 的 CNAME 开启 Cloudflare 代理。原有 GitHub Pages 自定义域名和 HTTPS 配置继续保留。

2026-10-20 检查时，`zxilly.dev` 全局 SSL 模式为 Flexible。为避免影响其他子域名，仅为 `http.host eq "cjv.zxilly.dev"` 新建 Configuration Rule，将 SSL 设为 Full (strict)。API 阶段为 `http_config_settings`，action 为 `set_config`，`action_parameters` 为 `{"ssl":"strict"}`。不要直接修改整区 SSL 模式；开启代理前必须保证该主机使用 Strict，以免 GitHub Pages 的 HTTPS 跳转与 Flexible 回源产生循环。

在 **Rules → Overview → Create rule → Redirect Rule** 中新建规则：

- 名称：`cjv China website`
- 匹配方式：Custom filter expression
- 表达式：

```text
(http.host eq "cjv.zxilly.dev" and ip.src.country eq "CN" and
 (http.request.uri.path in {"/" "/index.html"} or starts_with(http.request.uri.path, "/book/")))
```

- URL redirect 类型：Dynamic
- 目标表达式：

```text
concat("https://cjv.zxilly.com", http.request.uri.path)
```

- 状态码：`302`
- Preserve query string：启用

规则按请求来源 IP 判断中国大陆，不按浏览器语言或国籍判断。浏览器自身会保留通常的 URL fragment（如文档章节锚点）；片段不发送到服务端。

此规则只匹配首页和文档路径，保留旧的 `/install.sh`、`/install.ps1`、`/dl/` 地址直接返回内容。已发布的 `curl -sSf ... | sh` 命令不会跟随重定向，因此不应给整个 `.dev` 站点直接套用域名跳转。安装脚本示例及双语文档统一使用 `cjv.zxilly.com`；从 `cjv.zxilly.dev` 访问首页时显示 `.dev` 安装命令，正式 `.com`、本地及预览环境显示 `.com` 安装命令。

`.com` 不配置返回 `.dev` 的地区跳转，避免形成循环。上线后使用临时重定向，便于调整分流或回退。

## 验证与回退

启用规则前，核对 `.com` 的首页、四份文档入口、两个安装脚本和各平台 `/dl/` 文件；下载文件应与同次 GitHub Pages artifact 中的内容一致。确认不存在的文件返回合理的 404，而不是安装命令会误执行的首页 HTML。

启用后，从中国大陆和境外出口分别请求 `.dev` 首页及深层文档，检查状态码、`Location`、查询参数和浏览器锚点。中国大陆请求应跳到同路径 `.com`；境外请求仍由 `.dev` 提供服务。两种出口访问 `.dev/install.sh` 都应直接返回脚本。不要用客户端自设的国家请求头代替真实出口验证。

如需回退，先禁用 Cloudflare 重定向规则；如入口代理也有问题，再把 `.dev` CNAME 恢复为仅 DNS。如需暂停腾讯发布，应调整工作流中的 `edgeone` 任务；删除项目变量或 Token 会使该任务失败。

## 官方文档

- [Makers GitHub Actions 部署](https://pages.edgeone.ai/document/using-github-actions)
- [Makers CLI](https://pages.edgeone.ai/document/edgeone-cli)
- [Makers 直接上传](https://pages.edgeone.ai/document/direct-upload)
- [Makers 构建输出与路由](https://pages.edgeone.ai/document/building-output-configuration)
- [Makers 域名及加速区域](https://cloud.tencent.com/document/product/1552/127403)
- [Cloudflare 按地区重定向示例](https://developers.cloudflare.com/rules/url-forwarding/examples/redirect-country-subdomains/)
- [Cloudflare 创建重定向规则](https://developers.cloudflare.com/rules/url-forwarding/single-redirects/create-dashboard/)
- [Cloudflare 按请求配置 SSL](https://developers.cloudflare.com/rules/configuration-rules/settings/)
