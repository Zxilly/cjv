import { BLOG_COPY, BLOG_POSTS, blogPath, postForPath, translatedPath } from './blog'
import { languageForPath } from './i18n'

export const SITE_URL = 'https://cjv.zxilly.com'

export const PAGE_META = {
  zh: {
    path: '/', language: 'zh-CN', locale: 'zh_CN',
    title: 'cjv：仓颉 SDK 安装与版本管理器',
    description: 'cjv 已转入维护模式，不再开发新功能，后续仅处理安全修复。',
  },
  en: {
    path: '/en/', language: 'en', locale: 'en_US',
    title: 'cjv — Cangjie SDK installer and version manager',
    description: 'cjv is now in maintenance mode. No new features will be developed; future changes will be limited to security fixes.',
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
