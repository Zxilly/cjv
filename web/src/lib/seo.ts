import { BLOG_COPY, BLOG_POSTS, blogPath, postForPath, translatedPath } from './blog'
import { languageForPath } from './i18n'

export const SITE_URL = 'https://cjv.zxilly.com'

export const PAGE_META = {
  zh: {
    path: '/', language: 'zh-CN', locale: 'zh_CN',
    title: 'cjv：仓颉 SDK 安装与版本管理器',
    description: '使用 cjv 安装和切换仓颉编程语言 SDK，为项目固定工具链版本，管理 stdx 组件，让本地开发与 CI 使用一致的环境。支持 Windows、macOS、Linux 和鸿蒙。',
  },
  en: {
    path: '/en/', language: 'en', locale: 'en_US',
    title: 'cjv — Cangjie SDK installer and version manager',
    description: 'Install and switch Cangjie SDKs, pin toolchain versions per project, and manage stdx components with cjv. Keep local development and CI consistent on Windows, macOS, Linux and HarmonyOS.',
  },
} as const

export function pageSEO(pathname: string) {
  const lang = languageForPath(pathname)
  const home = PAGE_META[lang]
  const post = postForPath(pathname)
  const isBlog = /^\/(en\/)?blog(?:\/|$)/.test(pathname)
  const path = isBlog ? blogPath(lang, post?.slug) : home.path
  const title = post ? `${post[lang].title} | cjv` : isBlog ? `${BLOG_COPY[lang].title} | cjv Blog` : home.title
  const description = post ? post[lang].description : isBlog ? BLOG_COPY[lang].description : home.description
  const url = SITE_URL + path
  const data = post ? {
    '@type': 'BlogPosting', headline: post[lang].title, description, url,
    mainEntityOfPage: { '@type': 'WebPage', '@id': url },
    inLanguage: home.language, datePublished: post.date,
    author: { '@type': 'Person', name: 'Zxilly', url: 'https://github.com/Zxilly' },
    publisher: { '@type': 'Organization', name: 'cjv', url: SITE_URL },
    isPartOf: { '@type': 'Blog', '@id': SITE_URL + blogPath(lang) },
  } : isBlog ? {
    '@type': 'Blog', name: BLOG_COPY[lang].title, description, url, inLanguage: home.language,
    blogPost: BLOG_POSTS.map(entry => ({ '@type': 'BlogPosting', headline: entry[lang].title, url: SITE_URL + blogPath(lang, entry.slug) })),
  } : {
    '@type': 'SoftwareApplication', name: 'cjv', url: SITE_URL,
    description, applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Windows, macOS, Linux, HarmonyOS',
    license: 'https://www.apache.org/licenses/LICENSE-2.0',
    sameAs: ['https://github.com/Zxilly/cjv', 'https://gitcode.com/Zxilly/cjv'],
  }
  return {
    ...home, path, title, description, type: post ? 'article' : 'website', published: post?.date,
    zhPath: translatedPath(path, 'zh'), enPath: translatedPath(path, 'en'),
    structuredData: { '@context': 'https://schema.org', ...data },
  }
}
