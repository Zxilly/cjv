# Website edge functions

`deployment/` contains edge function builders, provider adapters, and verification scripts for the two cjv website domains:

- **`cjv.zxilly.dev`** — Hosted on GitHub Pages with a Cloudflare Worker route in front of the homepage.
- **`cjv.zxilly.com`** — Hosted on Tencent Cloud EdgeOne Makers.

Both domains serve the same static site artifact produced by GitHub Actions, while adapting the initial mirror toggle on the homepage to the visitor's region.

## How homepage geo-personalization works

When a visitor requests `/` or `/index.html`, both edge providers inspect the two-letter country code attached to the incoming request by the CDN:

- If the visitor's country code is `CN`, the edge function rewrites the homepage HTML to set `data-default-mirror="true"`.
- For all other countries—or when geolocation metadata is unavailable—the edge function sets `data-default-mirror="false"`.

When the React application in `web/` mounts for the first time, it reads `data-default-mirror` from the document to initialize the GitCode mirror switch. Visitors can change the switch manually. Region selection uses CDN metadata and does not request browser geolocation permission.

Because the homepage HTML varies by visitor location, personalized responses carry `Cache-Control: private, no-store`. All other routes—including hashed frontend assets, `/dl/` standalone installers, `/book/` documentation pages, `install.sh`, `install.ps1`, and `404.html`—continue to use their standard static routes and cache policies.

## Tencent Cloud EdgeOne (`cjv.zxilly.com`)

During the website deployment workflow (`.github/workflows/pages.yml`), `deployment/build-edge.mjs` transforms the shared `github-pages` artifact into a prebuilt EdgeOne output directory. This bundle includes an edge function bound to `/` and `/index.html` that reads `request.eo.geo.countryCodeAlpha2`. The homepage route is ordered ahead of static-file lookup so that the edge function executes even though a static `index.html` file is present in the bundle.

To build and deploy the EdgeOne bundle manually from a built `web/dist` directory:

```sh
node deployment/build-edge.mjs edgeone web/dist /path/to/edgeone-output/.edgeone
edgeone makers deploy /path/to/edgeone-output/.edgeone --name cjv-web-global --env production
```

The output directory passed to `edgeone makers deploy` must be named `.edgeone`. If you pass a directory with any other name, the EdgeOne CLI treats it as a plain static directory upload, leaving the edge function inactive and placing static assets under the wrong URL prefix.

After deploying, the CI workflow automatically runs `deployment/verify-site.mjs` against both public domains to confirm that the homepage, documentation books, installer scripts, and download routes respond as expected.

## Cloudflare Worker (`cjv.zxilly.dev`)

### Architecture and deployment triggers

On `cjv.zxilly.dev`, a Cloudflare Worker wraps the GitHub Pages origin for the homepage routes declared in `deployment/cloudflare/worker.config.json` and reads `request.cf.country`. Because the Worker fetches the current homepage HTML from the GitHub Pages origin on each request, ordinary website or documentation updates do not require redeploying the Worker. You only need to redeploy it when you modify the shared response logic, the Cloudflare adapter, the edge builder, or `worker.config.json`.

The `.github/workflows/cloudflare.yml` (`Deploy Cloudflare Worker`) workflow watches `deployment/` and its own workflow file on `master` (and also supports manual `workflow_dispatch` runs). Unrelated Go or documentation changes do not trigger a Worker deployment; similarly, Go-only changes do not redeploy either website unless called by the release workflow to publish new `/dl/` binaries.

### Credentials and manual deployment

The Cloudflare workflow uses two GitHub repository settings:

- **`CLOUDFLARE_ACCOUNT_ID`** (repository variable) — The target Cloudflare account ID.
- **`CLOUDFLARE_API_TOKEN`** (repository secret) — An API token with account `Workers Scripts:Write` permission, plus `Workers Routes:Write` and `Zone:Read` for the `zxilly.dev` zone. Check the token expiry in Cloudflare and rotate it before it expires.

To build and deploy the Cloudflare Worker locally with the npm `cf` CLI (`1.0.0-beta.6`):

```sh
node deployment/build-edge.mjs cloudflare - .cloudflare/output/v0
cf deploy --prebuilt --dry-run
cf deploy --prebuilt
```

The CLI requires an authenticated Cloudflare profile with Worker script and route permissions, and the `cjv.zxilly.dev` DNS record must have Cloudflare proxying (orange cloud) enabled so that the Worker route intercepts homepage traffic.

## Validation

Run the tests for both edge bundles locally with Node.js:

```sh
node --test deployment/edge.test.mjs
```

These tests execute both the generated Cloudflare and EdgeOne bundles against simulated `CN`, non-`CN`, and missing-country requests, verifying `data-default-mirror` injection, `Cache-Control` headers, manual-route boundaries, origin error handling, `HEAD` responses, and preservation of static assets.
