import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { BlogLayout } from '../components/blog-layout'
import { BLOG_POSTS, postForPath, translatedPath } from './blog'
import { renderArticle } from './blog.server'
import { pageSEO } from './seo'

describe('localized blog publishing', () => {
  it('normalizes aliases and preserves the article when switching language', () => {
    expect(translatedPath('/en/index.html', 'zh')).toBe('/')
    expect(translatedPath('/blog', 'en')).toBe('/en/blog/')
    expect(postForPath('/blog/missing/')).toBeUndefined()
    for (const post of BLOG_POSTS) {
      for (const lang of ['zh', 'en'] as const) {
        const path = `${lang === 'en' ? '/en' : ''}/blog/${post.slug}/`
        const metadata = pageSEO(`${path}index.html`)
        expect(metadata.path).toBe(path)
        expect(metadata.title).toContain(post[lang].title)
        expect(metadata.structuredData['@type']).toBe('BlogPosting')
        expect(metadata.zhPath).toBe(`/blog/${post.slug}/`)
        expect(metadata.enPath).toBe(`/en/blog/${post.slug}/`)
        const article = renderArticle(post.slug, lang)
        expect(article.html).toContain('<pre><code')
        expect(article.headings.length).toBeGreaterThan(2)
        expect(new Set(article.headings.map(heading => heading.id)).size).toBe(article.headings.length)
      }
    }
  })

  it('distinguishes blog collections from installation pages', () => {
    for (const path of ['/blog/', '/en/blog/index.html']) {
      expect(pageSEO(path).structuredData['@type']).toBe('Blog')
    }
    expect(pageSEO('/').structuredData['@type']).toBe('SoftwareApplication')
    expect(pageSEO('/en/index.html').path).toBe('/en/')
  })

  it('rejects unpublished articles with a real not-found response', () => {
    try {
      renderArticle('missing', 'zh')
      expect.unreachable('unknown articles must not render')
    } catch (error) {
      expect(error).toBeInstanceOf(Response)
      expect((error as Response).status).toBe(404)
    }
  })

  it.each(['zh', 'en'] as const)('links %s readers to the matching translation and canonical manual', lang => {
    const slug = BLOG_POSTS[0].slug
    render(<MemoryRouter initialEntries={[`${lang === 'en' ? '/en' : ''}/blog/${slug}/`]}><BlogLayout><main id="main">Article</main></BlogLayout></MemoryRouter>)
    expect(screen.getByRole('link', { name: lang === 'zh' ? 'English' : '中文' }).getAttribute('href')).toBe(`${lang === 'zh' ? '/en' : ''}/blog/${slug}/`)
    expect(screen.getByRole('link', { name: lang === 'zh' ? '用户手册' : 'User guide' }).getAttribute('href')).toBe(`https://cjv.zxilly.com/book/user-guide/${lang === 'zh' ? 'zh-CN' : 'en'}/`)
    expect(screen.getByRole('link', { name: lang === 'zh' ? '开发实践' : 'Blog' }).hasAttribute('aria-current')).toBe(false)
  })
})
