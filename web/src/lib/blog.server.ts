import { Marked } from 'marked'
import { BLOG_POSTS } from './blog'
import type { Lang } from './i18n'

const sources = import.meta.glob('../content/blog/*/*.md', { query: '?raw', import: 'default', eager: true }) as Record<string, string>

export function renderArticle(slug: string, lang: Lang) {
  if (!BLOG_POSTS.some(post => post.slug === slug)) throw new Response('Not found', { status: 404 })
  const source = sources[`../content/blog/${lang}/${slug}.md`]
  if (!source) throw new Error(`Missing ${lang} translation for ${slug}`)
  const headings: { id: string; text: string }[] = []
  const parser = new Marked({
    renderer: {
      heading({ tokens, depth }) {
        if (depth !== 2 && depth !== 3) throw new Error('Article headings must use H2 or H3')
        const text = this.parser.parseInline(tokens)
        const id = `section-${headings.length + 1}`
        headings.push({ id, text: text.replace(/<[^>]*>/g, '') })
        return `<h${depth} id="${id}">${text}</h${depth}>\n`
      },
    },
  })
  // These are reviewed, repository-owned Markdown files, never visitor input.
  const html = parser.parse(source, { async: false })
  return { html, headings }
}
