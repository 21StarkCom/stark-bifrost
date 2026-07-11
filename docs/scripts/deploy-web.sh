#!/usr/bin/env bash
set -euo pipefail

# deploy-web.sh — manual, local deploy of the web registry origin.
#
# Replaces the disabled `.github/workflows/web-deploy.yml` (disabled to keep
# bifrost zero-cost: it rebuilt a Docker image + redeployed Cloud Run on every
# push to main, growing GAR storage). Run this ONLY when you actually want to
# publish site changes — the native CC marketplace reads the public repo
# directly and does NOT need this site.
#
# Auth: uses your LOCAL gcloud ADC + docker (not WIF). You must be
# authenticated to the ev-infra-group project with rights to push to GAR and
# `gcloud run deploy` the stark-marketplace service (run.developer). The
# allUsers run.invoker binding + LB/DNS stay Terraform-owned in ev-infra-group.
#
# Usage:
#   docs/scripts/deploy-web.sh            # build + push + deploy
#   DRY_RUN=1 docs/scripts/deploy-web.sh  # print what would run, do nothing

GCP_PROJECT_ID="ev-infra-group"
GCP_REGION="us-central1"
GAR_REPO="stark-marketplace"
CR_SERVICE="stark-marketplace"
RUNTIME_SA="stark-marketplace-run@ev-infra-group.iam.gserviceaccount.com"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

SHA="$(git rev-parse HEAD)"
IMAGE="${GCP_REGION}-docker.pkg.dev/${GCP_PROJECT_ID}/${GAR_REPO}/${CR_SERVICE}"

run() {
  echo "+ $*"
  if [ "${DRY_RUN:-0}" != "1" ]; then "$@"; fi
}

echo "==> Building SPA (web/)"
run bash -c 'cd web && npm ci && npm run lint && npm run typecheck && npm test && npm run build'

echo "==> Staging index.json + bundles/ into web/dist (atomic content-hashed unit)"
run bash -c 'mkdir -p web/dist/bundles && cp index.json web/dist/index.json && (cp bundles/*.json web/dist/bundles/ 2>/dev/null || true)'

echo "==> Building + pushing image ${IMAGE}:${SHA}"
run gcloud auth configure-docker "${GCP_REGION}-docker.pkg.dev" --quiet
run docker build -t "${IMAGE}:${SHA}" -t "${IMAGE}:latest" .
run docker push "${IMAGE}:${SHA}"
run docker push "${IMAGE}:latest"

echo "==> Deploying Cloud Run ${CR_SERVICE} (behind platform LB)"
run gcloud run deploy "${CR_SERVICE}" \
  --project "${GCP_PROJECT_ID}" \
  --region "${GCP_REGION}" \
  --image "${IMAGE}:${SHA}" \
  --service-account "${RUNTIME_SA}" \
  --ingress internal-and-cloud-load-balancing \
  --port 8080 \
  --cpu 1 --memory 256Mi \
  --min-instances 0 --max-instances 4 --concurrency 80 \
  --quiet

echo "==> Done. Verify: curl -sI https://marketplace.21stark.com/ | head -1"
echo "    Prune old GAR images afterwards to keep storage near-zero:"
echo "    gcloud artifacts docker images list ${IMAGE} --include-tags"
