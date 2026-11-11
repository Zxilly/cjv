import { Link, useLoaderData, type LoaderFunctionArgs } from 'react-router'
import { BlogLayout } from '../components/blog-layout'
import { BLOG_COPY, BLOG_POSTS, blogPath } from '../lib/blog'
import { renderArticle } from '../lib/blog.server'
import { languageForPath } from '../lib/i18n'

export function loader({ request, params }: LoaderFunctionArgs) {
  const lang = languageForPath(new URL(request.url).pathname)
  const post = BLOG_POSTS.find(entry => entry.slug === params.slug)
  if (!post) throw new Response('Not found', { status: 404 })
  return { post, lang, ...renderArticle(post.slug, lang) }
}

export default function BlogPost() {
  const { post, lang, html, headings } = useLoaderData<typeof loader>()
  const copy = BLOG_COPY[lang]
  return <BlogLayout>
    <main id="main" className="blog-article">
      <Link className="blog-back" to={blogPath(lang)}>{copy.back}</Link>
      <article>
        <header className="blog-article-header">
          <h1>{post[lang].title}</h1>
          <p className="blog-intro">{post[lang].description}</p>
          <div className="blog-meta"><a href="https://github.com/Zxilly">Zxilly</a><span>{copy.published} <time dateTime={post.date}>{post.date}</time></span><span>{post[lang].category}</span></div>
        </header>
        <nav className="blog-contents" aria-label={copy.contents}>
          <h2>{copy.contents}</h2>
          <ol>{headings.map(heading => <li key={heading.id}><a href={`#${heading.id}`}>{heading.text}</a></li>)}</ol>
        </nav>
        <div className="blog-prose" dangerouslySetInnerHTML={{ __html: html }} />
      </article>
      <aside className="blog-related" aria-labelledby="related-title">
        <h2 id="related-title">{copy.related}</h2>
        <ul>{BLOG_POSTS.filter(other => other.slug !== post.slug).map(other => <li key={other.slug}><Link to={blogPath(lang, other.slug)}>{other[lang].title}</Link></li>)}</ul>
      </aside>
    </main>
  </BlogLayout>
}
