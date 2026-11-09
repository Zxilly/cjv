import type { ReactNode } from 'react'
import { Links, Outlet, Scripts, ScrollRestoration, useLocation } from 'react-router'
import { LanguageProvider } from './lib/language-provider'
import { languageForPath } from './lib/i18n'
import { PAGE_META, SITE_URL } from './lib/seo'
import './style.css'

export function Layout({ children }: { children: ReactNode }) {
  const { pathname } = useLocation()
  const lang = languageForPath(pathname)
  const meta = PAGE_META[lang]
  // Edge workers add this per-request attribute to the static document.
  const defaultMirror = typeof document === 'undefined' ? undefined : document.documentElement.dataset.defaultMirror

  return (
    <html lang={meta.language} data-prerendered="true" data-page-lang={lang} data-default-mirror={defaultMirror}>
      <head>
        <meta charSet="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <title>{meta.title}</title>
        <meta name="description" content={meta.description} />
        <link rel="canonical" href={SITE_URL + meta.path} />
        <link rel="alternate" hrefLang="zh-CN" href={SITE_URL + '/'} />
        <link rel="alternate" hrefLang="en" href={SITE_URL + '/en/'} />
        <link rel="alternate" hrefLang="x-default" href={SITE_URL + '/'} />
        <link rel="describedby" href="/llms.txt" />
        <meta property="og:type" content="website" />
        <meta property="og:site_name" content="cjv" />
        <meta property="og:title" content={meta.title} />
        <meta property="og:description" content={meta.description} />
        <meta property="og:url" content={SITE_URL + meta.path} />
        <meta property="og:locale" content={meta.locale} />
        <meta name="twitter:card" content="summary" />
        <meta name="twitter:title" content={meta.title} />
        <meta name="twitter:description" content={meta.description} />
        <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify({
          '@context': 'https://schema.org', '@type': 'SoftwareApplication', name: 'cjv',
          url: SITE_URL, description: meta.description, applicationCategory: 'DeveloperApplication',
          operatingSystem: 'Windows, macOS, Linux, HarmonyOS',
          license: 'https://www.apache.org/licenses/LICENSE-2.0',
          sameAs: ['https://github.com/Zxilly/cjv', 'https://gitcode.com/Zxilly/cjv'],
        }).replaceAll('<', '\\u003c') }} />
        <Links />
        <script defer src="https://trail.learningman.top/script.js" data-website-id="0e5c6fa1-6dd8-452d-acf3-70917b9fb499" />
      </head>
      <body className="bg-white dark:bg-gray-950 text-gray-800 dark:text-gray-200 antialiased min-h-screen flex items-center-safe">
        <div id="app" className="w-full">{children}</div>
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  )
}

export default function Root() {
  return <LanguageProvider><Outlet /></LanguageProvider>
}
