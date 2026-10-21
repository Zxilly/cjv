import { homeResponse } from '../home-response.mjs'

export default {
  async fetch(request) {
    // A route Worker fetches the underlying origin without invoking itself.
    // Drop validators so a 304 cannot reuse HTML personalized on another visit.
    const originRequest = new Request(request)
    originRequest.headers.delete('if-none-match')
    originRequest.headers.delete('if-modified-since')
    const response = await fetch(originRequest)
    if (response.status !== 200 || !response.headers.get('content-type')?.includes('text/html')) {
      return response
    }
    const injected = homeResponse(await response.text(), request.cf?.country, request.method)
    const headers = new Headers(response.headers)
    for (const [key, value] of injected.headers) headers.set(key, value)
    for (const key of ['content-length', 'content-encoding', 'etag', 'last-modified']) headers.delete(key)
    return new Response(injected.body, { status: injected.status, headers })
  },
}
