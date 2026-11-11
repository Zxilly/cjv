import type { Config } from '@react-router/dev/config'
import posts from './src/content/blog/posts.json'

export default {
  appDirectory: 'src',
  buildDirectory: 'dist',
  ssr: false,
  prerender: ['/', '/en/', '/blog/', '/en/blog/', ...posts.flatMap(post => [`/blog/${post.slug}/`, `/en/blog/${post.slug}/`])],
  // Static hosting has no runtime manifest endpoint.
  routeDiscovery: { mode: 'initial' },
} satisfies Config
