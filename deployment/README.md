# Website edge functions

The `deployment/` directory contains the edge function builders, adapters, and verification scripts for the two public `cjv` website domains:

- **`cjv.zxilly.dev`**: hosted on GitHub Pages and fronted by a Cloudflare Worker route.
- **`cjv.zxilly.com`**: hosted on Tencent Cloud EdgeOne Makers.

## Homepage geo-personalization

Both edge providers inspect the visitor's two-letter country code on requests to `/` and `/index.html`:

- When the visitor's country code is `CN`, the edge function sets `data-default-mirror="true"` on the homepage HTML response.
- For all other countries (or when geolocation metadata is absent), the edge function sets `data-default-mirror="false"`.
- On initial render, the React app in `web/` reads `data-default-mirror` to set the default state of the GitCode mirror switch. The visitor can still toggle the switch manually at any time, and the browser never requests the HTML5 Geolocation API.
- Personalized homepage responses set `Cache-Control: private, no-store`. All other routes (static assets, `/dl/` binaries, `/book/` documentation, `install.sh`, `install.ps1`, and `404.html`) use their normal static caching rules.

## Tencent Cloud EdgeOne (`cjv.zxilly.com`)

During `.github/workflows/pages.yml`, the deployment job converts the shared `github-pages` artifact into a prebuilt EdgeOne directory that includes an edge function for `/` and `/index.html`. The homepage function route runs before static file lookup and reads `request.eo.geo.countryCodeAlpha2`.

To build and deploy the EdgeOne bundle manually from `web/dist/`:

```sh
node deployment/build-edge.mjs edgeone web/dist /path/to/edgeone-output/.edgeone
edgeone makers deploy /path/to/edgeone-output/.edgeone --name cjv-web-global --env production
```

> **CAUTION:** You **must** name the output directory `.edgeone` when you run `edgeone makers deploy`. If you pass a directory with any other name, the EdgeOne CLI treats it as a plain static upload, skips the edge function, and serves assets under the wrong URL prefix.

After deployment, the workflow runs `deployment/verify-site.mjs` to check the live homepage, documentation, installer scripts, and download routes.

## Cloudflare Worker (`cjv.zxilly.dev`)

### How the route Worker operates

On `cjv.zxilly.dev`, a Cloudflare Worker wraps the GitHub Pages origin for the homepage routes defined in `deployment/cloudflare/worker.config.json`. It reads `request.cf.country` and fetches the current homepage HTML from the GitHub Pages origin on each request.

Because the Worker fetches live HTML from GitHub Pages, ordinary website or documentation updates do not need a Worker redeployment. You only need to redeploy the Worker when you change the shared response handler, the Cloudflare adapter, the edge builder, or `worker.config.json`.

### Automated and manual Cloudflare deployment

The `.github/workflows/cloudflare.yml` (`Deploy Cloudflare Worker`) workflow triggers automatically when files under `deployment/` or `.github/workflows/cloudflare.yml` change on `master`, or on manual `workflow_dispatch`.

The workflow uses two GitHub repository settings:

| Setting type | Name | Needed permissions |
| --- | --- | --- |
| Repository variable | `CLOUDFLARE_ACCOUNT_ID` | Target Cloudflare account ID |
| Repository secret | `CLOUDFLARE_API_TOKEN` | Account `Workers Scripts:Write`, plus `zxilly.dev` zone `Workers Routes:Write` and `Zone:Read` (current token expires on October 21, 2027; rotate before expiry) |

To build and deploy the Cloudflare Worker locally with the npm `cf` CLI (`1.0.0-beta.6`):

```sh
node deployment/build-edge.mjs cloudflare - .cloudflare/output/v0
cf deploy --prebuilt --dry-run
cf deploy --prebuilt
```

Enable the Cloudflare proxy (orange cloud) on the `cjv.zxilly.dev` DNS record so the Worker route intercepts homepage requests.

## Local validation

Run the unit and integration test suite for both edge bundles with Node.js:

```sh
node --test deployment/edge.test.mjs
```

`deployment/edge.test.mjs` runs both the generated Cloudflare and EdgeOne bundles against simulated `CN`, non-`CN`, and missing-country requests. It checks `data-default-mirror`, `Cache-Control` headers, route boundaries, `HEAD` requests, origin error handling, and static asset pass-through.
