# 目标与覆盖

每次执行被代理的 SDK 工具(如 `cjc`、`cjpm`)时,cjv 都要回答两个问题:用哪个工具链,以及这个工具链上要带哪些交叉编译目标。前者由一套带优先级的解析链决定,后者由「目标」(targets)这一附加安装维度决定。

## 工具链解析优先级

cjv 按以下优先级从高到低解析当前活跃的工具链,取第一个命中的来源:

1. `CJV_TOOLCHAIN` 环境变量
2. 目录覆盖(通过 `cjv override set` 设置)
3. 工具链文件(当前目录或某级父目录中的 `cangjie-sdk.toml`)
4. 默认工具链(通过 `cjv default` 设置)

代理执行、`cjv exec`、`cjv envsetup` 以及大多数接受 `[toolchain]` 参数的命令都使用同一套解析链。在任意目录下会用到哪个工具链,答案都是一致且可预测的。

### 1. `CJV_TOOLCHAIN` 环境变量

设置 `CJV_TOOLCHAIN` 会无条件覆盖其余所有来源,适合 CI、容器或临时验证场景:

```bash
CJV_TOOLCHAIN=nightly cjc --version
```

它优先级最高,目录覆盖、工具链文件、默认工具链都会被忽略。详见 [环境变量](../environment-variables.md)。

### 2. 目录覆盖

目录覆盖把某个具体目录(及其子目录)绑定到一个工具链,信息存放在全局 `settings.toml` 中,而不是项目里。它适合你不想或不能在项目里放 `cangjie-sdk.toml` 的情况,例如对方仓库不属于你,或你只想本地临时切换。

### 3. 工具链文件 `cangjie-sdk.toml`

cjv 从当前目录向上递归查找 `cangjie-sdk.toml`,用找到的第一个文件作为项目的工具链声明。这是随仓库提交、对协作者生效的项目级工具链锚点。完整字段说明见 [工具链文件](../toolchain-file.md)。

### 4. 默认工具链

当以上三者都没有命中时,回退到通过 `cjv default` 设置的全局默认工具链:

```bash
cjv default lts
```

若连默认工具链都未设置,cjv 会报错提示尚未配置任何工具链。

### 覆盖与工具链文件如何在目录树上交错

第 2 级和第 3 级并不是先扫完所有覆盖再扫所有工具链文件,而是沿目录树逐级向上,在每一级同时检查目录覆盖和 `cangjie-sdk.toml`:

- 在同一级目录上,目录覆盖优先于该级的 `cangjie-sdk.toml`;
- 更靠近当前目录的工具链文件,优先于更靠上的目录覆盖。

也就是说,优先级既看来源类型也看距离。若在 `~/work` 上设置了目录覆盖,而 `~/work/proj` 里有 `cangjie-sdk.toml`,那么在 `~/work/proj` 下工作时,更近的工具链文件胜出;只有当某一级既无更近的文件也命中了覆盖时,覆盖才生效。这样项目自带的声明就不会被祖先目录上一个宽泛的覆盖意外盖掉。

> 提示:`cangjie-sdk.toml` 存在但 `channel` 为空时,cjv 不会静默回退,而是直接报错,提醒你补全声明。只有文件根本不存在时才会继续向上查找。

## 管理目录覆盖

### 设置覆盖

```bash
# 为当前目录设置覆盖
cjv override set nightly

# 为指定目录设置(无需先 cd 进去)
cjv override set lts --path /path/to/project
```

`cjv override set` 会先校验工具链名。标准名称(如 `lts`、`sts`、`nightly`、具体版本)会被规范化后存储,自定义工具链名按原样接受。目录路径在写入前会被规范化为绝对路径,并解析符号链接、在 Windows 上统一盘符大小写,因此同一目录的不同写法不会产生重复条目。

### 移除覆盖

```bash
# 移除当前目录的覆盖
cjv override unset

# 移除指定目录的覆盖
cjv override unset --path /path/to/project

# 清理所有指向「已不存在目录」的覆盖
cjv override unset --nonexistent
```

`--nonexistent` 适合定期清理。项目目录被删除后,其覆盖条目会残留在 `settings.toml` 里,这条命令会把所有目标目录已不存在的覆盖一次性删掉。

### 列出覆盖

```bash
cjv override list
```

输出按目录路径排序,每行形如 `目录 → 工具链`。没有任何覆盖时会给出相应提示。

## 交叉编译目标(targets)

「目标」是与工具链解析正交的另一个维度。它回答的不是用哪个工具链,而是这个工具链要额外带哪些交叉编译 SDK。

目标 SDK 是宿主工具链之上的附加安装项,不会改变当前活跃的工具链。代理执行 `cjc`、`cjpm` 时,仍然使用宿主 SDK。安装一个目标只是把对应的交叉编译 SDK 备好,供你在需要时为该目标产出二进制。

### 只填后缀

无论在命令行还是在 `cangjie-sdk.toml` 中,`targets` 都只填目标后缀,例如 `ohos`、`android`、`ohos-arm32`,不要写完整平台 key(如 `linux-x64-ohos`)。cjv 会把后缀拼接到当前宿主元组上,自动得到完整目标。后缀必须匹配 `^[a-z0-9]+(?:-[a-z0-9]+)*$`,且不能本身就是一个完整平台元组。

### 在安装时附加目标

```bash
# 安装宿主 STS SDK,并额外装上当前宿主对应的 OHOS 交叉 SDK
cjv install sts -t ohos

# target 支持重复或逗号分隔
cjv install sts -t ohos -t android
cjv install sts --target ohos,android
```

### 在项目里声明目标

项目可在 `cangjie-sdk.toml` 中声明附加 `targets`。开启 `auto_install` 时,代理执行会自动补齐缺失的目标 SDK:

```toml
[toolchain]
channel = "sts"
targets = ["ohos", "android", "ohos-arm32"]
```

重复的后缀会自动去重。`targets` 与工具链解析无关,它只在你选定的工具链上叠加交叉编译能力。关于如何驱动交叉编译构建、以及目标 SDK 的运行时环境,见 [交叉编译](../cross-compilation.md)。
