# 内部分发源

`dist_server` 是工具链制品根地址。假设配置为：

```toml
dist_server = "https://artifacts.corp.example/cjv/dist"
```

cjv 会读取：

```text
https://artifacts.corp.example/cjv/dist/versions.json
```

相对 SDK 与组件 URL 会以 `dist_server` 为基准解析；绝对 URL 则完全按 manifest 中的值使用，可以指向同一制品库的其他路径，也可以指向其他主机。

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

分发端点通过 HTTPS GET 向受管终端提供机器可读的只读访问。网络白名单、设备身份或企业反向代理可以实施访问控制。

## Manifest 契约

`versions.json` 包含 `lts` 与 `sts`；启用企业 nightly 时再加入 `nightly`。每个 SDK 条目包含 `name`、`url` 和 `sha256`。组件条目支持可选 `sha256`，企业镜像宜一并提供。

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

上游 Release tag 与 SDK 资产版本存在差异时填写可选字段 `release_tag`；工具链名称使用版本字段，下载地址以 manifest 中的 URL 为准。

组件结构与普通 manifest 相同：`docs`、`stdx-docs` 各是一项，`stdx` 按制品平台键组织。完整字段可参考企业实际使用的 `versions.json`，并保留所有批准平台。

## Nightly 的处理方式

统一分发源下，nightly 与 LTS/STS 使用同一份 manifest：

- `cjv install nightly` 安装 `channels.nightly.latest`。
- `cjv install nightly-<version>` 安装 manifest 中的确切版本。
- `cjv check`、`cjv update nightly` 和 `cjv toolchain list-remote --channel nightly` 读取同一来源。
- nightly SDK 与组件使用 manifest 中的 URL 和校验和，版本解析直接由企业 manifest 完成。
- manifest 缺少 nightly、目标平台或组件时返回对应的缺失错误。

`channels.nightly.latest` 精确选择一个版本，目标与组件完整性是该版本的发布条件。企业发布流程先上传同一 nightly 的全部批准平台和组件，验证可下载后再更新 `latest`。项目固定确切 nightly 版本可获得可复现构建。

## 发布顺序

1. 镜像并校验批准版本的 SDK 与组件归档。
2. 计算 SDK SHA-256；组件也建议计算并写入 manifest。
3. 先发布所有归档，并用普通终端账户验证 HTTPS GET。
4. 生成 `versions.json`。内网部署通常使用相对路径或企业批准的内部绝对 URL。
5. 最后原子替换 manifest；只有制品齐全时才推进各通道的 `latest`。
6. 保留项目工具链文件引用的全部精确版本。

来源优先级为 `CJV_DIST_SERVER`、设置文件中的 `dist_server`、`manifest_url` / GitCode nightly 兼容模式。CI 可用环境变量临时选择测试源。
