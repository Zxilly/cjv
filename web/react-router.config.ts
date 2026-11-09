import type { Config } from '@react-router/dev/config'

export default {
  appDirectory: 'src',
  buildDirectory: 'dist',
  ssr: false,
  prerender: ['/', '/en/'],
  // Static hosting has no runtime manifest endpoint.
  routeDiscovery: { mode: 'initial' },
} satisfies Config
