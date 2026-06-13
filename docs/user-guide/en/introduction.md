# Introduction

cjv (Cangjie Version Manager) is the toolchain manager for the [Cangjie](https://cangjie-lang.cn/) programming language SDK. It manages multiple Cangjie SDK installations on one machine, handles version switching, and transparently proxies execution of SDK tools such as `cjc` and `cjpm`.

## What problems cjv solves

When you install the Cangjie SDK directly, the system usually keeps only one version, and switching versions means downloading, extracting, and changing `PATH` again. This gets tedious when you have several projects that each need a different SDK.

cjv installs LTS, STS, nightly, and specific-version SDKs side by side under a single home directory, without interference. You can set one default toolchain for the whole machine, and you can also pin a toolchain for a single directory or a single command. cjv resolves which SDK to use each time by the priority of environment variable, directory override, toolchain file, and default.

When you run commands such as `cjc` or `cjpm` directly, cjv proxies the call to the resolved toolchain and injects the environment variables required by the runtime and components (such as `CANGJIE_STDX_PATH_DYNAMIC`). You do not need to change `PATH` or export variables by hand.

On top of this, cjv also supports cross-compilation target SDKs, extension components such as `stdx`, offline documentation, and configuring the runtime environment to run compiled artifacts directly.

## Who it's for

cjv is for Cangjie developers who need to switch back and forth between LTS, STS, and nightly, and for those maintaining multiple projects on one machine where each project is locked to a different SDK version. If you do cross-compilation (such as for HarmonyOS OHOS or Android) and need to manage target SDKs alongside the host SDK, cjv can help here too. Teams can use a toolchain file to make version switching reproducible and travel with the project.

## Where to start

- [Installing cjv](installation/index.md): obtain and install cjv itself.
- [Basic usage](basic-usage.md): install your first toolchain, set a default, and run commands.
- [Concepts](concepts/index.md): learn the terms toolchain, channel, component, proxy, and override, and understand how cjv works.

cjv is open source under the Apache-2.0 license; the source is at <https://github.com/Zxilly/cjv>.
