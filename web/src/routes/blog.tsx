import { Link, useLocation } from 'react-router'
import { BlogLayout } from '../components/blog-layout'
import { BLOG_COPY, BLOG_POSTS, blogPath } from '../lib/blog'
import { languageForPath } from '../lib/i18n'

export default function Blog() {
  const lang = languageForPath(useLocation().pathname)
  const copy = BLOG_COPY[lang]
  return <BlogLayout>
    <main id="main" className="blog-index">
      <h1>{copy.title}</h1>
      <p className="blog-intro">{copy.description}</p>
      <ol className="blog-post-list">
        {BLOG_POSTS.map(post => <li key={post.slug}>
          <div className="blog-meta"><span>{post[lang].category}</span><time dateTime={post.date}>{post.date}</time></div>
          <h2><Link to={blogPath(lang, post.slug)}>{post[lang].title}</Link></h2>
          <p>{post[lang].description}</p>
        </li>)}
      </ol>
    </main>
  </BlogLayout>
}
