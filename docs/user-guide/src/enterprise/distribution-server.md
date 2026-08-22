# 内部分发源

`dist_server` 是工具链制品根地址。假设配置为：

```toml
dist_server = "https://artifacts.corp.example/cjv/dist"
```

cjv 会读取：

```text
https://artifacts.corp.example/cjv/dist/versions.json
```

相对 SDK 与组件 URL 会以 `dist_server` 为基准解析；绝对 URL 则完全按 manifest 中的值使用，可以指向同一制品库的其他路径，也可以指向其他主机。cjv 把 manifest 视为管理员发布的可信配置，不替企业实施网络访问策略。

## 推荐布局

```text
企业制品库
└── cjv/
    ├── dist/                         # dist_server 指向这里
    │   ├── versions.json
    │   ├── sdk/                      # LTS / STS SDK
    │   ├── components/               # LTS / STS 组件
    │   └── nightly/                  # nightly SDK 与组件
    └── releases/                     # cjv 本体归档和 checksums.txt
```

分发端点应允许受管终端通过 HTTPS GET 读取。需要访问控制时，建议通过网络白名单、设备身份或企业反向代理提供机器可访问的只读端点，不要依赖交互式登录页。

## Manifest 契约

`versions.json` 必须包含 `lts` 与 `sts`；要在严格内网中使用 nightly，还必须包含 `nightly`。每个 SDK 条目都需要 `name`、`url` 和 `sha256`。组件条目支持可选 `sha256`，企业镜像应尽量提供。

```jsonc
{
  "channels": {
    "lts": {
      "latest": "1.0.5",
      "versions": {
        "1.0.5": {
          "linux-x64": {
            "name": "cangjie-sdk-linux-x64-1.0.5.tar.gz",
            "url": "sdk/cangjie-sdk-linux-x64-1.0.5.tar.gz",
            "sha256": "<64 位十六进制 SHA-256>"
          }
        }
      }
    },
    "sts": {
      "latest": "1.1.0",
      "versions": {
        "1.1.0": {
          "linux-x64": {
            "name": "cangjie-sdk-linux-x64-1.1.0.tar.gz",
            "url": "sdk/cangjie-sdk-linux-x64-1.1.0.tar.gz",
            "sha256": "<64 位十六进制 SHA-256>"
          }
        }
      }
    },
    "nightly": {
      "latest": "1.2.0-alpha.20260822010101",
      "versions": {
        "1.2.0-alpha.20260822010101": {
          "linux-x64": {
            "name": "cangjie-sdk-linux-x64-1.2.0-alpha.20260822010101.tar.gz",
            "url": "nightly/20260822/cangjie-sdk-linux-x64-1.2.0-alpha.20260822010101.tar.gz",
            "sha256": "<64 位十六进制 SHA-256>",
            "release_tag": "1.1.0-alpha.20260822010101"
          }
        }
      },
      "components": {
        "1.2.0-alpha.20260822010101": {
          "docs": {
            "name": "cangjie-docs-html-1.2.0-alpha.20260822010101.tar.gz",
            "url": "nightly/20260822/cangjie-docs-html-1.2.0-alpha.20260822010101.tar.gz",
            "sha256": "<64 位十六进制 SHA-256>"
          }
        }
      }
    }
  }
}
```

`release_tag` 是可选字段。只有上游 Release tag 与 SDK 资产版本不同时才需要填写；工具链名称仍使用版本字段，下载地址以 manifest 中的 URL 为准。

组件结构与普通 manifest 相同：`docs`、`stdx-docs` 各是一项，`stdx` 按制品平台键组织。完整字段可参考企业实际使用的 `versions.json`，并保留所有批准平台。

## Nightly 的处理方式

统一分发源下，nightly 与 LTS/STS 使用同一份 manifest：

- `cjv install nightly` 安装 `channels.nightly.latest`。
- `cjv install nightly-<version>` 安装 manifest 中的确切版本。
- `cjv check`、`cjv update nightly` 和 `cjv toolchain list-remote --channel nightly` 读取同一来源。
- nightly SDK 与组件都使用 manifest 中的 URL 和校验和，不需要 GitCode API 令牌。
- manifest 缺少 nightly、目标平台或组件时直接失败，不会访问公网补齐。

cjv 不会像 rustup 那样在最新 nightly 缺组件时逐日回退。企业发布流程应先上传同一 nightly 的全部批准平台和组件，验证可下载后再更新 `latest`。项目应固定确切 nightly 版本，避免构建结果随 `latest` 漂移。

## 发布顺序

1. 镜像并校验批准版本的 SDK 与组件归档。
2. 计算 SDK SHA-256；组件也建议计算并写入 manifest。
3. 先发布所有归档，并用普通终端账户验证 HTTPS GET。
4. 生成 `versions.json`。内网部署通常使用相对路径或企业批准的内部绝对 URL。
5. 最后原子替换 manifest；只有制品齐全时才推进各通道的 `latest`。
6. 保留仍被项目工具链文件引用的精确版本，不要只保留最新版本。

`CJV_DIST_SERVER` 会覆盖设置文件中的 `dist_server`，适合 CI 临时选择测试源。若两者均未设置，才使用旧的 `manifest_url` / GitCode nightly 兼容模式。
