# 网站部署

`cjv.zxilly.dev` 使用 GitHub Pages，并由 Cloudflare Worker 处理首页；`cjv.zxilly.com` 使用腾讯云 EdgeOne Makers。两站共用一次 Actions 构建的网站产物，包括首页、双语手册、安装脚本和 `/dl/` 安装器。

## 自动部署

`.github/workflows/pages.yml` 在网站、文档或相关部署文件变化时运行，也接受手动触发和发布流程调用。它先发布 GitHub Pages，再把同一份 `github-pages` artifact 转为 EdgeOne 预构建目录并部署。两个部署任务都必须成功。

Cloudflare Worker 由 `.github/workflows/cloudflare.yml` 单独部署。该工作流只在 Worker、共享响应处理、构建工具、验证工具或自身工作流变化时运行。普通网站内容变化不需要重新部署 Worker，因为它每次从 Pages 获取当前首页。

触发条件直接使用工作流的 `paths`；Go 源码变化不触发网站部署。发布工作流调用 Pages 时仍会更新下载文件。修改共享主题或文档构建脚本时，也需要触发网站部署。

| 类型 | 名称 | 用途 |
| --- | --- | --- |
| Repository variable | `EDGEONE_PROJECT_NAME` | 已创建的 Makers 生产项目 |
| Repository secret | `EDGEONE_API_TOKEN` | 腾讯云中国站 Makers API Token |
| Repository variable | `CLOUDFLARE_ACCOUNT_ID` | Worker 所属账户 |
| Repository secret | `CLOUDFLARE_API_TOKEN` | Worker 部署凭据 |

凭据权限和边缘代码的构建命令以 [deployment/README.md](../deployment/README.md) 为准。缺少 EdgeOne 项目或 Token 会使部署明确失败。不要通过删除凭据暂停部署；需要暂停时调整工作流。

## EdgeOne 手动部署

先从成功的网站工作流下载 `github-pages` artifact，解开外层 ZIP 和其中的 `artifact.tar`。解出的目录必须包含 `index.html`、`book/`、`dl/`、`install.sh`、`install.ps1` 和 `404.html`。

在仓库根目录执行，`./site` 是解出的完整网站目录：

```sh
node deployment/build-edge.mjs edgeone ./site ./edgeone-output/.edgeone
npx --yes edgeone@1.6.44 login --site china --token "$EDGEONE_API_TOKEN"
npx --yes edgeone@1.6.44 makers deploy ./edgeone-output/.edgeone \
  --name "$EDGEONE_PROJECT_NAME" --env production --skip-ai-gateway-sync
node deployment/verify-site.mjs https://cjv.zxilly.dev/ https://cjv.zxilly.com/
```

必须上传名为 `.edgeone` 的预构建目录。直接上传 `./site` 会绕过边缘函数；把预构建目录改成其他名称会让 CLI 将其视为普通静态上传。构建工具统一生成首页函数和路由，首页按请求国家设置初始镜像选项，其他路径通过 origin hook 返回静态文件或 404。

`--site china` 选择腾讯云中国站账户入口，与项目加速区域不同。先创建项目并确认区域，再配置 Actions 变量，避免 CLI 因项目不存在而意外创建新项目。`--skip-ai-gateway-sync` 跳过静态网站不使用的 AI Gateway 同步。

## 域名与地区跳转

Makers 自定义域名绑定生产环境，CNAME 使用控制台给出的目标，在 Cloudflare 中设置为仅 DNS。项目的加速区域在创建时选择；包含中国大陆的区域要求域名完成备案。调整区域时创建新项目并迁移域名，按控制台的域名释放状态决定切换时间。

`cjv.zxilly.dev` 保留 GitHub Pages 自定义域名与 HTTPS 配置，并在 Cloudflare 开启代理。为该主机单独使用 Full (strict) 回源，避免 Flexible 与 Pages 的 HTTPS 跳转产生循环；不要为此修改整个 DNS 区域的 SSL 模式。

中国大陆跳转规则位于 `.dev` 的 Cloudflare 入口，表达式为：

```text
(http.host eq "cjv.zxilly.dev" and ip.src.country eq "CN" and
 (http.request.uri.path in {"/" "/index.html"} or starts_with(http.request.uri.path, "/book/")))
```

目标为 `concat("https://cjv.zxilly.com", http.request.uri.path)`，状态码 `302`，保留查询参数。规则按来源 IP 判断，不按浏览器语言判断。`.com` 不配置反向地区跳转。

仅匹配首页和文档，保留 `.dev/install.sh`、`.dev/install.ps1` 和 `.dev/dl/` 直接返回内容。已发布的 `curl -sSf ... | sh` 不跟随重定向。文档使用 `.com`；从 `.dev` 首页访问时，页面仍显示 `.dev` 的安装命令。

## 验证与回退

`deployment/verify-site.mjs` 检查首页、静态资源、文档、安装脚本、下载入口和 404。下载内容还应与同次发布产物核对；部署成功不代表路由或文件内容正确。

调整地区规则后，从中国大陆和境外出口分别检查 `.dev` 首页和深层文档的状态码、`Location` 与查询参数；两种出口访问安装脚本都应直接返回脚本。客户端自设国家请求头不能替代真实出口验证。

回退时先禁用地区重定向，确认用户可以通过 `.dev` 访问。若入口代理有问题，再把 `.dev` CNAME 改回仅 DNS。重新部署历史 artifact 时仍使用当前边缘构建步骤，避免恢复已经修复的上传布局问题。

## 2026-10-20 迁移记录

以下是当日操作记录，不代表当前最新部署：

- 初始项目 `cjv-web`（`makers-9riya3l0zoze`），首次部署 `dpqt1qkz6d2l`，使用 v0.4.1 安装器；后续部署 `dpjy0k9ak9gn` 当时保留用于回退。
- 迁移后的项目为 `cjv-web-global`（`makers-nbvih0bn28ch`），当时验证的部署为 `dp8y8ezpjjzq`，区域为全球可用区（含中国大陆）。
- 当日自定义域名 CNAME 目标为 `cjv.zxilly.com.pages.dnsoe4.com`；域名迁移后 HTTPS、四份文档、两份脚本、14 个安装器与缺失路径的 404 均已验证。
- 当日 Cloudflare Trace 验证了中国大陆规则的 302，本机通过境外出口验证了 `.dev` 页面和脚本；没有从真实中国大陆出口测量响应。

当前生产项目和部署状态应以 Actions、Makers 控制台及公开站点验证为准。

## 官方文档

- [Makers GitHub Actions 部署](https://pages.edgeone.ai/document/using-github-actions)
- [Makers CLI](https://pages.edgeone.ai/document/edgeone-cli)
- [Makers 构建输出与路由](https://pages.edgeone.ai/document/building-output-configuration)
- [Makers 域名及加速区域](https://cloud.tencent.com/document/product/1552/127403)
- [Cloudflare 按地区重定向](https://developers.cloudflare.com/rules/url-forwarding/examples/redirect-country-subdomains/)
- [Cloudflare 按请求配置 SSL](https://developers.cloudflare.com/rules/configuration-rules/settings/)
