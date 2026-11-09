import { renderToString } from 'react-dom/server'
import App from './App'
import { activateLang, type Lang } from './lib/i18n'
export { PAGE_META, SITE_URL } from './lib/seo'

// The build renders locales sequentially; no request-time shared locale state.
export function render(lang: Lang) {
  activateLang(lang)
  return renderToString(<App />)
}
