import { route, type RouteConfig } from '@react-router/dev/routes'

export default [
  // Static hosts also expose index.html; use the same route IDs for hydration.
  route('index.html?', 'routes/home.tsx', { id: 'home-zh' }),
  route('en/index.html?', 'routes/home.tsx', { id: 'home-en' }),
  route('blog/index.html?', 'routes/blog.tsx', { id: 'blog-zh' }),
  route('en/blog/index.html?', 'routes/blog.tsx', { id: 'blog-en' }),
  route('blog/:slug/index.html?', 'routes/blog-post.tsx', { id: 'post-zh' }),
  route('en/blog/:slug/index.html?', 'routes/blog-post.tsx', { id: 'post-en' }),
] satisfies RouteConfig
