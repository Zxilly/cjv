import { route, type RouteConfig } from '@react-router/dev/routes'

export default [
  // Static hosts also expose index.html; use the same route IDs for hydration.
  route('index.html?', 'routes/home.tsx', { id: 'home-zh' }),
  route('en/index.html?', 'routes/home.tsx', { id: 'home-en' }),
] satisfies RouteConfig
