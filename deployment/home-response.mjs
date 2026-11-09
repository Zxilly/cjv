// This self-contained function is embedded in both providers' deployment bundles.
export function homeResponse(html, country, method, hostname) {
  if (method !== 'GET' && method !== 'HEAD') {
    return new Response(null, { status: 405, headers: { Allow: 'GET, HEAD' } })
  }
  // Keep installer commands usable on the selected mirror domain, while canonical
  // and hreflang URLs consistently identify the primary .com site.
  const localized = hostname === 'cjv.zxilly.dev'
    ? html.replaceAll('https://cjv.zxilly.com/install.', 'https://cjv.zxilly.dev/install.')
    : html
  const body = localized.replace('<html ', `<html data-default-mirror="${country === 'CN'}" `)
  return new Response(method === 'HEAD' ? null : body, {
    headers: {
      'Content-Type': 'text/html; charset=utf-8',
      // The shared static assets stay cacheable; this HTML varies by visitor.
      'Cache-Control': 'private, no-store',
    },
  })
}
