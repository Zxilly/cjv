# Website edge functions

Both providers set `data-default-mirror="true"` on the homepage HTML when the
visitor's country code is `CN`. Other countries and missing location data use
`false`. React reads this value on its first render; the visitor can still change
the switch manually. The browser does not request a geolocation API.

Personalized homepage responses use `Cache-Control: private, no-store`. Assets,
downloads, documentation, and missing-file responses retain their static routes.

## EdgeOne

The website deployment workflow generates a prebuilt EdgeOne directory from the
shared GitHub Pages artifact, including an edge function for `/` and `/index.html`.
The homepage route precedes static-file lookup so it also runs when `index.html`
exists. The function uses `request.eo.geo.countryCodeAlpha2`.

```sh
node deployment/build-edge.mjs edgeone web/dist /path/to/edgeone-output/.edgeone
edgeone makers deploy /path/to/edgeone-output/.edgeone --name cjv-web-global --env production
```

The output directory must be named `.edgeone`. The CLI treats other directory
names as plain static uploads, leaving functions inactive and assets under the
wrong URL prefix. The workflow also checks the public homepage and static routes
after deployment.

## Cloudflare

The route Worker wraps GitHub Pages on `cjv.zxilly.dev` and uses
`request.cf.country`. It fetches the current origin HTML on each request, so
ordinary website deployments do not require redeploying the Worker. Deploy it
again when the shared response logic, Worker, or route configuration changes.

The `Deploy Cloudflare Worker` GitHub Actions workflow watches the Cloudflare
adapter, shared response logic, builder, verifier, and its workflow file. Each
matching push builds and deploys the Worker. Manual workflow runs also build and
deploy it. Unrelated file changes do not trigger this workflow.

The workflow uses the repository secret `CLOUDFLARE_API_TOKEN` and variable
`CLOUDFLARE_ACCOUNT_ID`. The token needs Workers Scripts Write for the account,
plus Workers Routes Write and Zone Read for `zxilly.dev`. The current CI token
expires on October 21, 2027; rotate it before that date.

The website workflow watches only its build inputs and deployment configuration.
Go-only changes do not redeploy either website. Release workflows can still call
it to publish new download binaries. EdgeOne publishes its static files and
embedded homepage function together, so a new homepage also updates that bundle.

Build the output and deploy with the npm `cf` CLI, version `1.0.0-beta.6`:

```sh
node deployment/build-edge.mjs cloudflare - .cloudflare/output/v0
cf deploy --prebuilt --dry-run
cf deploy --prebuilt
```

The CLI needs an authenticated profile with Worker script and route permissions.
The domain's DNS record must be proxied through Cloudflare. Only the homepage
routes in `cloudflare/worker.config.json` are attached to this Worker.

## Validation

```sh
node --test deployment/edge.test.mjs
```

These tests execute both generated bundles with CN, non-CN, and missing country
metadata, and check cache policy, manual-route boundaries, origin errors, HEAD
responses, and preservation of static assets.
