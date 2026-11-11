import type { ReactNode } from 'react'
import { Link, useLocation } from 'react-router'
import { BLOG_COPY, blogPath, translatedPath } from '../lib/blog'
import { languageForPath } from '../lib/i18n'
import { SITE_URL } from '../lib/seo'

export function BlogLayout({ children }: { children: ReactNode }) {
  const { pathname } = useLocation()
  const lang = languageForPath(pathname)
  const copy = BLOG_COPY[lang]
  return (
    <div className="blog-shell">
      <a className="blog-skip" href="#main">{lang === 'zh' ? '跳至正文' : 'Skip to content'}</a>
      <header className="blog-header">
        <Link className="blog-brand" to={lang === 'en' ? '/en/' : '/'} reloadDocument aria-label={copy.home}>cjv</Link>
        <nav aria-label={lang === 'zh' ? '主导航' : 'Main navigation'}>
          <Link to={blogPath(lang)} aria-current={translatedPath(pathname, lang) === blogPath(lang) ? 'page' : undefined}>{copy.blog}</Link>
          <a href={`${SITE_URL}/book/user-guide/${lang === 'en' ? 'en' : 'zh-CN'}/`}>{copy.docs}</a>
          <Link to={translatedPath(pathname, lang === 'zh' ? 'en' : 'zh')} hrefLang={lang === 'zh' ? 'en' : 'zh-CN'} lang={lang === 'zh' ? 'en' : 'zh-CN'}>{lang === 'zh' ? 'English' : '中文'}</Link>
        </nav>
      </header>
      {children}
      <footer className="blog-footer">
        <span>cjv · {copy.title}</span>
        <a href="https://github.com/Zxilly/cjv">GitHub</a>
      </footer>
    </div>
  )
}
