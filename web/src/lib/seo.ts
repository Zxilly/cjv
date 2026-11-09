export const SITE_URL = 'https://cjv.zxilly.com'

export const PAGE_META = {
  zh: {
    path: '/', language: 'zh-CN', locale: 'zh_CN',
    title: 'cjv：仓颉 SDK 安装与版本管理器',
    heading: '仓颉 SDK 版本管理器',
    description: '使用 cjv 安装和切换仓颉编程语言 SDK，为项目固定工具链版本，管理 stdx 组件，让本地开发与 CI 使用一致的环境。支持 Windows、macOS、Linux 和鸿蒙。',
    intro: '安装和切换仓颉 SDK，为每个项目固定工具链，让本地开发与 CI 使用一致的版本。',
  },
  en: {
    path: '/en/', language: 'en', locale: 'en_US',
    title: 'cjv — Cangjie SDK installer and version manager',
    heading: 'Cangjie SDK version manager',
    description: 'Install and switch Cangjie SDKs, pin toolchain versions per project, and manage stdx components with cjv. Keep local development and CI consistent on Windows, macOS, Linux and HarmonyOS.',
    intro: 'Install and switch Cangjie SDKs. Pin a toolchain for each project and keep local development and CI on the same version.',
  },
} as const
