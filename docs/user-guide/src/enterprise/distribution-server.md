# 内部分发源

[配置](../configuration.md)中的 `dist_server` 字段（或 `CJV_DIST_SERVER` [环境变量](../environment-variables.md)）指定工具链制品与版本清单的根地址。假设配置为：

```toml
dist_server = "https://artifacts.corp.example/cjv/dist"
```

`cjv` 会按通道在该根地址下读取两个独立的清单文件：

```text
https://artifacts.corp.example/cjv/dist/versions.json  # LTS / STS
https://artifacts.corp.example/cjv/dist/nightly.json   # nightly
```

- LTS 和 STS 操作只读取 `versions.json`；nightly 操作只读取 `nightly.json`。
- `cjv check` 与 `cjv update` 仅根据本机已安装的通道按需加载对应文件。
- 清单中相对路径形式的 SDK 与组件 `url` 以分发根（`dist_server`）为基准解析，绝对 URL 则完全按清单中的原值请求。

分发源优先级依次为：`CJV_DIST_SERVER` > 合并设置中的 `dist_server` > `manifest_url`。CI 流水线可以使用 `CJV_DIST_SERVER` 环境变量临时选择测试分发源。

## 文件布局

建议在企业制品库中按以下结构组织工具链分发目录（`dist/`）与 `cjv` 本体发布目录（`releases/`）：

```text
企业制品库
└── cjv/
    ├── dist/                         # dist_server 指向这里
    │   ├── versions.json             # LTS / STS 及其组件
    │   ├── nightly.json              # nightly 及其组件
    │   ├── sdk/
    │   ├── components/
    │   └── nightly/
    └── releases/                     # cjv 本体归档和 checksums.txt
```

分发端点只需通过 `HTTPS GET` 向受管终端提供机器可读的只读访问；网络白名单、设备证书或企业反向代理均可用于实施访问控制。

## `versions.json`

`versions.json` 的顶层是 `channels` 对象，包含 `lts` 与 `sts` 两个通道：

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
    }
  }
}
```

## `nightly.json`

`nightly.json` 直接在顶层表示一个通道，包含 `latest`、`versions` 和可选的 `components`（请**不要**将 nightly 内容包裹在 `channels.nightly` 下）：

```jsonc
{
  "latest": "1.2.0-alpha.20260822010101",
  "versions": {
    "1.2.0-alpha.20260822010101": {
      "linux-x64": {
        "name": "cangjie-sdk-linux-x64-1.2.0-alpha.20260822010101.tar.gz",
        "url": "nightly/20260822/cangjie-sdk-linux-x64-1.2.0-alpha.20260822010101.tar.gz",
        "sha256": "<64 位十六进制 SHA-256>"
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
```

每个 SDK 条目包含 `name`、`url` 和 `sha256`。上游 `nightly` 的 `sha256` 在刚发布时可能暂时为空，此时 `cjv` 会自动请求 `<url>.sha256`；而在企业内网镜像中，建议直接计算并填入完整的 64 位校验和。组件条目同样支持可选的 `sha256` 字段，建议一并填写以便客户端校验完整性并写入可复用离线缓存。

`versions` 中的版本键决定工具链名称，`url` 则精确定位下载归档。`nightly` 安装以 `latest` 为起点，并根据所需的交叉目标和组件在清单历史中查找兼容版本（默认不降级）；固定完整版本名时则直接选用对应版本。

## 组件格式

每个通道对象的 `components` 字段按版本号索引。`docs` 与 `stdx-docs` 在每个版本下各对应一个归档对象；`stdx` 则进一步按组件平台标识索引：

```jsonc
{
  "components": {
    "1.0.5": {
      "stdx": {
        "linux-x64": {
          "name": "stdx-linux-x64.tar.gz",
          "url": "components/stdx-linux-x64.tar.gz",
          "sha256": "<64 位十六进制 SHA-256>"
        }
      },
      "docs": { "name": "docs.tar.gz", "url": "components/docs.tar.gz" },
      "stdx-docs": { "name": "stdx-docs.tar.gz", "url": "components/stdx-docs.tar.gz" }
    }
  }
}
```

以上是通道对象的片段，实际文件中需与同一通道的 `latest`、`versions` 合并。[交叉编译](../cross-compilation.md) SDK 位于 `versions[版本][平台]` 下（例如 `linux-x64-ohos`），其对应的 `stdx` 则使用组件平台标识（例如 `ohos-aarch64`）。

## 发布顺序

向内部分发源发布新版本时，请按以下顺序操作，确保客户端不会读到指向缺失文件的清单：

1. 镜像并校验所有批准版本的宿主 SDK、交叉 SDK 与组件归档。
2. 计算每个 SDK 的 SHA-256；组件归档也建议计算并写入清单。
3. 先上传发布所有归档文件，并用普通终端账户验证 `HTTPS GET` 可正常访问。
4. 分别生成 `versions.json` 与 `nightly.json`，并保留项目工具链文件仍在引用的全部精确版本。
5. 原子替换对应的清单文件；确认该版本的所有制品齐全后再推进清单中的 `latest`。

客户端验收检查项见[企业与离线部署](index.md#发布和验收)。
