# stark-marketplace — Web registry hosting

> **Provisioned target.** Host = `marketplace.evinced-infra.group`, served on
> Cloud Run behind the `ev-infra-group` shared platform LB. Infra primitives
> (SAs/IAM/GAR/NEG/host-rule/DNS) live in Terraform
> (`ev-infra-group/infra/stark-marketplace.tf`) — no ad-hoc `gcloud`. The Cloud
> Run service itself is deployed by this repo's CI via WIF
> (`.github/workflows/web-deploy.yml`).

## Pattern: public static origin behind the platform LB

- **Origin:** the atomic content-hashed `web/dist` bundle (SPA shell + hashed assets +
  `index.json` + `bundles/*.json`), produced by `.github/workflows/web-deploy.yml`.
- **Public edge:** the shared external HTTPS LB serves
  `marketplace.evinced-infra.group`; there is no IAP block on the marketplace
  backend.
- **Origin posture:** Cloud Run deploys with
  `--ingress internal-and-cloud-load-balancing`, so the public `*.run.app` URL is
  blocked and traffic enters through the LB.

## Routing (no server rewrite needed)

The SPA uses **hash routing** (`HashRouter`): every route lives under `/#/…` on the single root
`index.html`, with assets referenced by a relative base. A deep link like
`…/#/bundle/stark-gh` therefore loads the root document and its hashed assets directly — the
dumb static origin needs **no SPA-fallback rewrite rule**, and a shared/refreshed deep link
never 404s its assets. (If you ever switch to history routing, the origin MUST rewrite unknown
paths to `/index.html` and serve a root-anchored asset base — call that out here first.)

## Caching (atomic unit)

- `assets/*` (content-hashed): `Cache-Control: public, max-age=31536000, immutable`.
- `index.html`, `index.json`, `bundles/*.json`: `Cache-Control: no-cache` (the
  cache-busted pointers). Because assets are hashed, a deploy flips atomically — a client
  never pairs a new `index.html` with a stale asset, nor an old SPA with a new index.

## SPA fetches

The SPA calls `fetch(..., { credentials: 'same-origin' })`. The origin serves
`index.json` and `bundles/*.json` as public static files from the same host. If
the index is unreadable, the data layer returns a degraded result instead of
throwing (`src/data/registry.ts`).

## Schema skew safety net

If the index `schemaVersion` is newer than the deployed SPA understands (or unreadable),
the SPA shows the degraded view rather than blank-failing (`src/data/schema.ts`,
`src/pages/DegradedPage.tsx`). N-1 schema versions still render.

## Provisioned implementation

```
browser → marketplace.evinced-infra.group
        → platform HTTPS LB (host rule, *.evinced-infra.group wildcard cert)
        → serverless NEG → Cloud Run `stark-marketplace` (server/)
```

| Piece | Owner | What |
|-------|-------|------|
| `server/` | this repo | stdlib Go static fileserver (SPA + `index.json` + `bundles/`, cache headers, `/healthz`) |
| `Dockerfile` | this repo | `web/dist` (built in CI) + the Go binary → distroless image |
| `.github/workflows/web-deploy.yml` | this repo | build → push to GAR → `gcloud run deploy` (WIF, keyless) |
| runtime SA, CI SA + WIF, GAR, NEG, backend, host rule, DNS, registry slot 2 | `ev-infra-group` (`infra/stark-marketplace.tf`) | all infra primitives |

**Cloud Run posture:** `--ingress internal-and-cloud-load-balancing` blocks the
public `*.run.app` URL (only the external LB reaches the service). The LB →
Cloud Run hop on a serverless NEG carries no Cloud Run IAM token, so the service
needs an `allUsers` `run.invoker` binding — **owned by Terraform** (gated), so the
deploy SA (`run.developer`) never calls `setIamPolicy`. Safe because ingress
restricts the path to the LB.

**Required repo variables** (Settings → Secrets and variables → Actions → Variables),
set from the Terraform outputs after the infra PR applies:

| Variable | Terraform output |
|----------|------------------|
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | `wif_provider_name` |
| `GCP_DEPLOY_SA` | `marketplace_ci_sa_email` |

Project (`ev-infra-group`), region (`us-central1`), GAR repo, service name, and
runtime SA are fixed in the workflow `env:` block.

**Bootstrap order (first time only):**
1. `ev-infra-group` apply with `marketplace_lb_enabled=false` creates identity and GAR.
2. Set the two repo variables from the TF outputs.
3. Merge this repo to `main` → CI builds + deploys the Cloud Run service.
4. Flip `marketplace_lb_enabled=true` in `ev-infra-group` and apply again to wire DNS/LB/alerts.
5. Visit `https://marketplace.evinced-infra.group`.

Until step 4 lands, the custom host is intentionally not wired.
