# 内部分发源

[配置](../configuration.md)中的 `dist_server` 字段（或 `CJV_DIST_SERVER` [环境变量](../environment-variables.md)）用于指定工具链清单与归档文件的根地址。本章定义内部分发源的推荐目录布局、`versions.json` 与 `nightly.json` 的格式契约，以及安全的制品发布顺序。

## 清单读取与 URL 解析规则

假设在 `settings.toml` 中配置了如下 `dist_server`：

```toml
dist_server = "https://artifacts.corp.example/cjv/dist"
```

cjv 会按通道在该根地址下读取两个独立的清单文件：

```text
https://artifacts.corp.example/cjv/dist/versions.json  # LTS 与 STS 通道
https://artifacts.corp.example/cjv/dist/nightly.json   # nightly 通道
```

- LTS 和 STS 相关操作只读取 `versions.json`。
- nightly 相关操作只读取 `nightly.json`。
- `cjv check` 与 `cjv update` 仅根据本机已安装的通道按需加载对应文件。
- 清单中相对路径形式的 SDK 和组件 `url` 会以 `dist_server` 为基准解析；绝对路径形式的 `https://` URL 则完全按原样请求。

分发源地址的优先级依次为：`CJV_DIST_SERVER` > 合并设置中的 `dist_server` > `manifest_url`。CI 流水线可通过设置 `CJV_DIST_SERVER` 临时指向测试分发源进行验证。

## 推荐目录布局

建议在企业制品库中将工具链分发目录 `dist/` 与 cjv 本体发布目录 `releases/` 并列存放：

```text
企业制品库
└── cjv/
    ├── dist/                         # dist_server 指向此目录
    │   ├── versions.json             # LTS / STS 通道及其组件清单
    │   ├── nightly.json              # nightly 通道及其组件清单
    │   ├── sdk/                      # 稳定通道 SDK 归档
    │   ├── components/               # 组件归档（stdx、docs、stdx-docs）
    │   └── nightly/                  # nightly SDK 与组件归档
    └── releases/                     # cjv 本体归档与 checksums.txt
```

分发服务端应通过 HTTPS GET 向受管终端提供只读访问，并可结合网络白名单、设备证书或企业反向代理实施访问控制。

## `versions.json` 格式契约

`versions.json` 的顶层为 `channels` 对象，包含 `lts` 和 `sts` 两个通道：

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

## `nightly.json` 格式契约

`nightly.json` 直接在顶层表示单个通道，包含 `latest`、`versions` 和可选的 `components`。请注意**不要**将 `nightly.json` 的内容包裹在 `channels.nightly` 下：

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

每个 SDK 条目均包含 `name`、`url` 和 `sha256`。上游 nightly 构建刚发布时，`sha256` 字段可能暂时为空，此时 cjv 会尝试请求 `<url>.sha256`；而在企业内部镜像中，建议直接计算并填入完整的 64 位 `sha256` 校验和。

`versions` 中的版本键决定工具链名称，`url` 则定位具体的下载文件。安装 `nightly` 通道时，cjv 从 `latest` 开始，结合所需的交叉目标和组件在历史版本中查找兼容版本，且默认不降级；固定完整版本名时则直接安装指定版本。

## 组件与交叉编译目标格式

每个通道对象（`channels.lts`、`channels.sts` 或 `nightly.json` 顶层）中的 `components` 字段按版本号索引组件归档：

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

- `docs` 与 `stdx-docs` 不区分平台，每个版本直接对应一个归档对象。
- `stdx` 进一步按组件平台标识索引（例如宿主平台 `linux-x64`，或交叉目标平台 `ohos-aarch64`）。
- [交叉编译](../cross-compilation.md) SDK 存放在 `versions[版本号][平台标识]` 下，使用 `linux-x64-ohos` 这样的“宿主-目标”组合键名。
- 组件条目支持可选的 `sha256` 字段；在企业内网镜像中建议为所有组件填入 `sha256`，以便客户端校验完整性并写入离线可复用缓存。

## 安全发布顺序

每次向内部分发源发布新版本时，请严格遵循以下顺序：

1. 镜像并校验所有批准版本的宿主 SDK、交叉 SDK 和组件归档。
2. 计算每个 SDK 与组件归档的 64 位十六进制 SHA-256 摘要。
3. 先将所有归档文件上传发布到目标路径，并使用普通终端账户验证 `HTTPS GET` 可正常下载。
4. 独立生成更新后的 `versions.json` 与 `nightly.json`，并保留现有项目工具链文件引用的全部精确历史版本。
5. 原子替换对应的 JSON 清单文件；确认所有归档就绪后再推进清单中的 `latest` 字段。

发布后的客户端验收项见[企业与离线部署](index.md#发布与验收清单)。
