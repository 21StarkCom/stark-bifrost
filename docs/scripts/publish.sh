#!/usr/bin/env bash
set -euo pipefail

# publish.sh — regenerate the marketplace from stark-skills, the RIGHT way.
#
# Since bifrost's GitHub Actions are disabled for $0 spend, the old auto-publish
# `marketplace-sync` PR no longer fires — regeneration is manual. This chains the
# error-prone two-step regen (sync -> golden -> build --fix -> build --check) that
# the CLAUDE.md gotcha warns about, so you can't commit a `sync` without a `build`
# and trip the drift gate.
#
# It can also add a skill to a bundle (edits bundle.yaml membership + bumps the
# bundle version) before regenerating.
#
# Usage:
#   docs/scripts/publish.sh                              # sync + build + check
#   docs/scripts/publish.sh --add-skill stark-gha-cost --bundle stark-ops
#   docs/scripts/publish.sh --ci                         # also run ci-local.sh
#   docs/scripts/publish.sh --add-skill X --bundle Y --ci --deploy
#
# Env:
#   STARK_SKILLS   path to the stark-skills checkout (default: ../../stark-skills
#                  relative to this repo, i.e. a sibling clone)
#
# It does NOT commit, push, or open a PR — that stays a deliberate human step
# (the script prints the next commands). --deploy runs deploy-web.sh at the end.

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

STARK_SKILLS="${STARK_SKILLS:-$REPO_ROOT/../stark-skills}"
ADD_SKILL="" BUNDLE="" RUN_CI=0 RUN_DEPLOY=0

while [ $# -gt 0 ]; do
  case "$1" in
    --add-skill) ADD_SKILL="${2:?}"; shift 2 ;;
    --bundle)    BUNDLE="${2:?}"; shift 2 ;;
    --ci)        RUN_CI=1; shift ;;
    --deploy)    RUN_DEPLOY=1; shift ;;
    -h|--help)   sed -n '3,30p' "${BASH_SOURCE[0]}"; exit 0 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

[ -d "$STARK_SKILLS/skill" ] || { echo "stark-skills not found at $STARK_SKILLS (set STARK_SKILLS)" >&2; exit 1; }
echo "→ stark-skills: $STARK_SKILLS"

# --- optional: add a skill to a bundle's membership + bump the bundle version ---
if [ -n "$ADD_SKILL" ]; then
  [ -n "$BUNDLE" ] || { echo "--add-skill requires --bundle" >&2; exit 2; }
  BFILE="catalog/$BUNDLE/bundle.yaml"
  [ -f "$BFILE" ] || { echo "no such bundle: $BFILE" >&2; exit 1; }
  [ -d "$STARK_SKILLS/skill/$ADD_SKILL" ] || { echo "skill '$ADD_SKILL' not in stark-skills" >&2; exit 1; }

  if grep -qE "^[[:space:]]*-[[:space:]]+$ADD_SKILL[[:space:]]*$" "$BFILE"; then
    echo "→ $ADD_SKILL already in $BUNDLE — skipping membership edit"
  else
    # append to the `skills:` block (env vars must be exported for $ENV in perl)
    ADD_SKILL="$ADD_SKILL" perl -0pi -e '
      my $s=$ENV{ADD_SKILL};
      s/(^skills:\n(?:[ \t]*-[ \t]*\S+\n)+)/"$1  - $s\n"/me;
    ' "$BFILE"
    grep -qE "^[[:space:]]*-[[:space:]]+$ADD_SKILL[[:space:]]*$" "$BFILE" \
      || { echo "ERROR: failed to insert $ADD_SKILL into $BFILE (no skills: block?)" >&2; exit 1; }
    # bump patch version (version: X.Y.Z)
    perl -i -pe 's/^(version:\s*\d+\.\d+\.)(\d+)\s*$/$1.($2+1)."\n"/e' "$BFILE"
    echo "→ added $ADD_SKILL to $BUNDLE, bumped $(grep -m1 '^version:' "$BFILE")"
  fi
fi

bump_bundle() {  # $1 = bundle name; bump its patch version (path absolute — cwd varies)
  perl -i -pe 's/^(version:\s*\d+\.\d+\.)(\d+)\s*$/$1.($2+1)."\n"/e' "$REPO_ROOT/catalog/$1/bundle.yaml"
}

cd engine

echo "→ stark sync (regenerate catalog + vendor from stark-skills)"
go run ./cmd/stark sync --from "$STARK_SKILLS" ../catalog

# Auto-bump every bundle whose generated content changed. A cross-cutting change
# in stark-skills (e.g. a frontmatter flip on many skills) touches several
# bundles; check-bumps content-locks each artifact to its bundle version, so each
# affected bundle must bump. Loop: detect → bump → re-sync until the gate is clean.
for round in 1 2 3 4 5; do
  # check-bumps exits non-zero when the gate fails — capture output without
  # tripping `set -e`/`pipefail`, then extract the affected bundle names.
  cb_out="$(go run ./cmd/stark check-bumps ../catalog 2>&1 || true)"
  changed=$(printf '%s\n' "$cb_out" \
    | sed -nE 's#^[[:space:]]*-[[:space:]]+([a-z0-9-]+)/.*#\1#p' | sort -u)
  [ -z "$changed" ] && { echo "→ version-bump gate clean"; break; }
  echo "→ content changed in: $(echo "$changed" | tr '\n' ' ') — bumping + re-syncing (round $round)"
  for b in $changed; do bump_bundle "$b"; done
  go run ./cmd/stark sync --from "$STARK_SKILLS" ../catalog >/dev/null
  [ "$round" = 5 ] && { echo "ERROR: check-bumps still failing after 5 rounds" >&2; exit 1; }
done

echo "→ refresh adapter golden (harmless if unchanged)"
UPDATE_GOLDEN=1 go test ./internal/adapter/... >/dev/null 2>&1 || true

echo "→ stark build --fix (regenerate dist / bundles / index)"
go run ./cmd/stark build --fix ../catalog

echo "→ stark build --check (drift gate)"
go run ./cmd/stark build --check ../catalog

echo "→ stark check-bumps (final gate)"
go run ./cmd/stark check-bumps ../catalog

cd "$REPO_ROOT"

if [ "$RUN_CI" -eq 1 ]; then
  echo "→ ci-local.sh (full local gate)"
  docs/scripts/ci-local.sh
fi

echo
echo "✅ regeneration complete + drift-clean."
echo "   Review + commit EVERYTHING (catalog + vendor + dist + golden):"
echo "     git add -A && git commit && git push && gh pr create ..."
if [ "$RUN_DEPLOY" -eq 1 ]; then
  echo "→ deploy-web.sh"
  docs/scripts/deploy-web.sh
else
  echo "   Then deploy the web origin when ready:  docs/scripts/deploy-web.sh"
fi
