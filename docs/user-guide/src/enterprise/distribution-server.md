# 内部分发源

`dist_server` 是工具链制品根地址。假设配置为：

```toml
dist_server = "https://artifacts.corp.example/cjv/dist"
```

cjv 按通道读取两个独立文件：

```text
https://artifacts.corp.example/cjv/dist/versions.json  # LTS / STS
https://artifacts.corp.example/cjv/dist/nightly.json   # nightly
```

LTS/STS 操作只读取 `versions.json`；nightly 操作只读取 `nightly.json`。`cjv check` 与 `cjv update` 根据已安装通道加载所需文件。相对 SDK 与组件 URL 以分发根为基准解析，绝对 URL 完全按 manifest 中的值使用。

## 推荐布局

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

分发端点通过 HTTPS GET 向受管终端提供机器可读的只读访问。网络白名单、设备身份或企业反向代理可以实施访问控制。

## `versions.json` 契约

`versions.json` 的顶层是 `channels`，包含 `lts` 与 `sts`：

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

## `nightly.json` 契约

`nightly.json` 直接表示一个通道，包含 `latest`、`versions` 和可选 `components`：

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

每个 SDK 条目包含 `name`、`url` 和 `sha256`。nightly 的 `sha256` 可以暂为空，此时 cjv 读取 `<url>.sha256`；企业镜像宜直接填入校验和。组件条目支持可选 `sha256`。版本键决定工具链名称，URL 精确定位下载资产。

`cjv install nightly` 安装 `nightly.json` 的 `latest`；带版本的 nightly 安装、`check`、`update`、`list-remote` 和组件安装都读取同一文件。项目固定确切 nightly 版本可获得可复现构建。

## 发布顺序

1. 镜像并校验批准版本的 SDK 与组件归档。
2. 计算 SDK SHA-256；组件也建议计算并写入 manifest。
3. 先发布所有归档，并用普通终端账户验证 HTTPS GET。
4. 分别生成 `versions.json` 与 `nightly.json`。
5. 原子替换对应文件；制品齐全后推进该文件中的 `latest`。
6. 保留项目工具链文件引用的全部精确版本。

来源优先级为 `CJV_DIST_SERVER`、设置文件中的 `dist_server`、`manifest_url`。CI 可用环境变量临时选择测试源。
