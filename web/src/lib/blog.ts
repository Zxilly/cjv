import posts from '../content/blog/posts.json'
import type { Lang } from './i18n'

export const BLOG_POSTS = posts
export const BLOG_COPY = {
  zh: {
    title: '仓颉开发实践',
    description: '用 cjv 配置仓颉开发环境、管理项目 SDK 版本并接入持续集成。每篇文章围绕一个具体任务，提供操作步骤与排查方法。',
    home: '安装 cjv', docs: '用户手册', blog: '开发实践',
    contents: '本文内容', related: '继续阅读', back: '全部文章', published: '发布于',
  },
  en: {
    title: 'Cangjie in practice',
    description: 'Set up a Cangjie development environment, manage project SDK versions and build in CI with cjv. Practical steps and troubleshooting for one task at a time.',
    home: 'Install cjv', docs: 'User guide', blog: 'Blog',
    contents: 'On this page', related: 'Keep reading', back: 'All articles', published: 'Published',
  },
} as const

export function blogPath(lang: Lang, slug?: string) {
  return `${lang === 'en' ? '/en' : ''}/blog/${slug ? `${slug}/` : ''}`
}

export function translatedPath(pathname: string, lang: Lang) {
  const path = pathname.replace(/^\/en(?=\/|$)/, '').replace(/index\.html$/, '') || '/'
  return `${lang === 'en' ? '/en' : ''}${path.endsWith('/') ? path : path + '/'}`
}

export function postForPath(pathname: string) {
  const path = pathname.replace(/^\/en(?=\/)/, '').replace(/index\.html$/, '').replace(/\/$/, '')
  return BLOG_POSTS.find(post => path === `/blog/${post.slug}`)
}
