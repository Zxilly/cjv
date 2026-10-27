# 网站部署与边缘路由

`cjv` 官方网站同时部署在两个公开域名上：

- **`cjv.zxilly.dev`** — 托管于 GitHub Pages，并由 Cloudflare Worker 处理首页的地区镜像默认开关与中国大陆跳转。
- **`cjv.zxilly.com`** — 托管于腾讯云 EdgeOne Makers，为境内外提供静态访问与边缘函数支持。

两个站点共用同一次 GitHub Actions 构建出的网站产物，包含落地页、中英文双语手册、安装脚本（`install.sh`、`install.ps1`）以及 `/dl/` 独立安装器。边缘函数的源码实现与本地单元测试说明见 [deployment/README.md](../deployment/README.md)。

## 自动化部署流程

### 网站与文档部署（`pages.yml`）

`.github/workflows/pages.yml` 在落地页、文档或相关部署文件发生变化时自动运行，同时也支持手动触发（`workflow_dispatch`）和发布工作流（`release.yml`）调用。

该工作流先在单次构建任务中生成完整的 `github-pages` 制品，随后先发布到 GitHub Pages（`cjv.zxilly.dev`），再将同一份制品通过 `deployment/build-edge.mjs` 转换为 EdgeOne 预构建目录并部署到腾讯云 EdgeOne（`cjv.zxilly.com`）。两个部署任务都必须成功，工作流才算通过。

触发条件直接由工作流的 `paths` 规则控制：单独修改 Go 源码不会触发网站重部署，但发布新版本时 `release.yml` 会主动调用 `pages.yml` 以更新网站上的 `/dl/` 安装器。修改 `docs/theme/` 共享主题或 `scripts/build-docs.py` 构建脚本时，同样会触发网站部署。

### Cloudflare Worker 部署（`cloudflare.yml`）

Cloudflare Worker 由 `.github/workflows/cloudflare.yml` 独立部署。该工作流仅在 Worker 适配器、共享响应处理逻辑、边缘构建工具、站点验证工具或其自身工作流文件发生变化时运行。

由于 Cloudflare Worker 在每次收到请求时都会实时从 GitHub Pages 回源获取当前首页 HTML，普通的网站文案或文档更新不需要重新部署 Worker。

### 仓库变量与凭据要求

| 类型 | 名称 | 用途说明 |
| --- | --- | --- |
| Repository variable | `EDGEONE_PROJECT_NAME` | 已预先创建的 EdgeOne Makers 生产项目名称 |
| Repository secret | `EDGEONE_API_TOKEN` | 腾讯云中国站 Makers API Token |
| Repository variable | `CLOUDFLARE_ACCOUNT_ID` | Cloudflare Worker 所属账户 ID |
| Repository secret | `CLOUDFLARE_API_TOKEN` | Cloudflare Worker 部署凭据（权限要求见 [deployment/README.md](../deployment/README.md)） |

缺少 EdgeOne 项目名称或 API Token 会导致部署失败。暂停部署应调整工作流文件。

## EdgeOne 手动部署

如果需要在本地手动执行 EdgeOne 部署，请先从最近一次成功的网站工作流中下载 `github-pages` artifact，依次解开外层 ZIP 和其中的 `artifact.tar`。解压出的完整网站目录必须包含 `index.html`、`book/`、`dl/`、`install.sh`、`install.ps1` 和 `404.html`。

假设 `./site` 是解压出的完整网站目录，在仓库根目录执行以下命令：

```sh
node deployment/build-edge.mjs edgeone ./site ./edgeone-output/.edgeone
npx --yes edgeone@1.6.44 login --site china --token "$EDGEONE_API_TOKEN"
npx --yes edgeone@1.6.44 makers deploy ./edgeone-output/.edgeone \
  --name "$EDGEONE_PROJECT_NAME" --env production --skip-ai-gateway-sync
node deployment/verify-site.mjs https://cjv.zxilly.dev/ https://cjv.zxilly.com/
```

执行手动部署时请注意以下细节：

- **预构建目录必须命名为 `.edgeone`**：直接上传 `./site` 会完全绕过边缘函数；而如果将预构建目录改名为其他名称，EdgeOne CLI 会将其视为普通静态目录上传，导致边缘函数不生效且静态资源路径前缀错位。`deployment/build-edge.mjs` 会统一生成首页边缘函数与路由配置，使首页按请求所在国家设置初始镜像开关，其余路径通过 origin hook 返回静态文件或 `404.html`。
- **账户站点与项目加速区域**：`--site china` 用于选择腾讯云中国站账户登录入口，它与 Makers 项目本身的加速区域是两个独立概念。请先在控制台创建好生产项目并确认加速区域，再配置 GitHub Actions 变量，避免 CLI 因找不到同名项目而意外创建新项目。
- **跳过无关同步**：`--skip-ai-gateway-sync` 用于跳过静态网站不需要的 AI Gateway 配置同步。

## 域名配置与地区跳转

### 域名绑定与回源 SSL

- **`cjv.zxilly.com`（EdgeOne Makers）**：在 Makers 控制台中将自定义域名绑定到生产环境，按照控制台给出的目标地址在 Cloudflare DNS 中添加 CNAME 记录，并将代理状态设置为**仅 DNS（灰色云朵）**。项目的加速区域在创建项目时选定；包含中国大陆的加速区域要求域名已完成 ICP 备案。如果后续需要调整加速区域，应创建新项目并迁移自定义域名，根据控制台显示的域名释放状态决定切换时间。
- **`cjv.zxilly.dev`（GitHub Pages + Cloudflare）**：保留 GitHub Pages 的自定义域名与 HTTPS 配置，并在 Cloudflare DNS 中开启代理（橙色云朵）。请通过 Cloudflare Configuration Rules 为 `cjv.zxilly.dev` 这一主机单独设置 **Full (strict)** 回源 SSL 模式，避免 `Flexible` 模式与 GitHub Pages 的 HTTPS 强制跳转产生循环重定向；此规则不应修改整个 `zxilly.dev` DNS 区域的全局 SSL 模式。

### 中国大陆访问重定向规则

`cjv.zxilly.dev` 的 Cloudflare 入口将中国大陆访问者的首页和文档请求重定向到 `.com`，匹配表达式如下：

```text
(http.host eq "cjv.zxilly.dev" and ip.src.country eq "CN" and
 (http.request.uri.path in {"/" "/index.html"} or starts_with(http.request.uri.path, "/book/")))
```

重定向目标设为 `concat("https://cjv.zxilly.com", http.request.uri.path)`，状态码为 `302`，并开启保留查询参数（Preserve query string）。该规则按来源 IP 所属国家判断，不按浏览器语言判断；`cjv.zxilly.com` 不配置反向地区跳转。

这条规则只匹配首页和 `/book/` 文档路径，保留了 `.dev/install.sh`、`.dev/install.ps1` 和 `.dev/dl/` 直接返回内容。这是因为许多已发布的安装命令 `curl -sSf ... | sh` 默认不带 `-L` 参数，不会跟随 `302` 重定向。公开文档统一使用 `.com` 链接；而从境外直接访问 `.dev` 首页时，页面仍显示 `.dev` 域名下的安装命令。

## 验证与故障回退

`deployment/verify-site.mjs` 会自动检查两个域名的首页、静态资源、四份双语文档、安装脚本、`/dl/` 下载入口以及缺失路径的 `404` 响应。除自动化检查外，发布新版本后还应将下载到的安装器与同次发布的制品校验和进行核对——部署任务成功并不代表路由或文件内容一定正确。

每次调整地区跳转规则后，应分别从中国大陆出口和境外出口实测 `.dev` 首页与深层文档链接的状态码、`Location` 响应头及查询参数保留情况，并确认两种出口访问 `install.sh` / `install.ps1` 均能直接拿到脚本内容（客户端自行伪造国家请求头不能替代真实出口验证）。

如果线上需要回退：

1. 先在 Cloudflare 中禁用中国大陆地区重定向规则，确保所有用户都能通过 `cjv.zxilly.dev` 正常访问。
2. 若 Cloudflare 入口代理本身存在问题，再将 `cjv.zxilly.dev` 的 CNAME 改回**仅 DNS**。
3. 如需重新部署历史 `github-pages` artifact，仍应使用当前最新的 `deployment/build-edge.mjs` 边缘构建步骤，避免重新引入早期已修复的上传目录结构问题。

## 2026-10-20 迁移记录

以下为 2026-10-20 当日的操作记录，仅供历史追溯，不代表当前最新部署状态：

- 初始项目为 `cjv-web`（`makers-9riya3l0zoze`），首次部署 ID 为 `dpqt1qkz6d2l`（使用 `v0.4.1` 安装器）；后续部署 `dpjy0k9ak9gn` 当时保留用于回退。
- 迁移后的项目为 `cjv-web-global`（`makers-nbvih0bn28ch`），当时验证通过的部署 ID 为 `dp8y8ezpjjzq`，加速区域为全球可用区（含中国大陆）。
- 当日自定义域名 CNAME 目标为 `cjv.zxilly.com.pages.dnsoe4.com`；域名迁移后 HTTPS、四份双语文档、两份安装脚本、14 个平台安装器及缺失路径的 `404` 均已验证通过。
- 当日通过 Cloudflare Trace 验证了中国大陆规则的 `302` 跳转，本机通过境外出口验证了 `.dev` 页面和脚本；未从真实中国大陆出口测量响应。

当前生产项目和最新部署状态请以 GitHub Actions、Makers 控制台及公开站点实时验证为准。

## 官方参考文档

- [Makers GitHub Actions 部署](https://pages.edgeone.ai/document/using-github-actions)
- [Makers CLI](https://pages.edgeone.ai/document/edgeone-cli)
- [Makers 构建输出与路由配置](https://pages.edgeone.ai/document/building-output-configuration)
- [Makers 域名及加速区域](https://cloud.tencent.com/document/product/1552/127403)
- [Cloudflare 按地区重定向](https://developers.cloudflare.com/rules/url-forwarding/examples/redirect-country-subdomains/)
- [Cloudflare 按请求配置 SSL](https://developers.cloudflare.com/rules/configuration-rules/settings/)
