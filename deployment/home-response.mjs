// This self-contained function is embedded in both providers' deployment bundles.
export function homeResponse(html, country, method) {
  if (method !== 'GET' && method !== 'HEAD') {
    return new Response(null, { status: 405, headers: { Allow: 'GET, HEAD' } })
  }
  const body = html.replace('<html ', `<html data-default-mirror="${country === 'CN'}" `)
  return new Response(method === 'HEAD' ? null : body, {
    headers: {
      'Content-Type': 'text/html; charset=utf-8',
      // The shared static assets stay cacheable; this HTML varies by visitor.
      'Cache-Control': 'private, no-store',
    },
  })
}
