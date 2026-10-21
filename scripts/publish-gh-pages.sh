#!/usr/bin/env bash
# Build the Hugo site from the current checkout and publish it to the gh-pages branch.
#
# Publishes by fast-forward push only: it never force-pushes, so a racing run is
# rejected instead of overwriting the other one. It keeps CNAME and .nojekyll,
# which the build does not produce but GitHub Pages needs.
#
# Run it from the repo root with the theme submodule checked out
# (git clone --recurse-submodules). DRY_RUN=1 builds and shows what would change
# without pushing.
#
# Environment:
#   REPO_URL   where to clone gh-pages from and push to
#              (default git@github.com:baditaflorin/0deepresearch.git)
#   MIN_PAGES  refuse to publish a build with fewer pages than this (default 50)
#   DRY_RUN=1  do not push
set -euo pipefail

REPO_URL="${REPO_URL:-git@github.com:baditaflorin/0deepresearch.git}"
BRANCH="gh-pages"
DOMAIN="0deepresearch.com"
MIN_PAGES="${MIN_PAGES:-50}"

src="$(git rev-parse --show-toplevel)"
src_sha="$(git -C "$src" rev-parse --short HEAD)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

[ -f "$src/themes/ananke/theme.toml" ] || {
  echo "theme submodule missing: clone with --recurse-submodules" >&2
  exit 1
}

# Build outside the repo so a tracked public/ directory is never touched.
hugo --gc --minify --source "$src" --destination "$work/out" >/dev/null
hugo_version="$(hugo version | sed -E 's/^hugo v([0-9.]+).*/\1/')"

pages="$(find "$work/out" -name index.html | wc -l | tr -d ' ')"
if [ ! -f "$work/out/index.html" ] || [ "$pages" -lt "$MIN_PAGES" ]; then
  echo "refusing to publish: build has $pages pages (minimum $MIN_PAGES)" >&2
  exit 1
fi

git clone --quiet --branch "$BRANCH" --single-branch "$REPO_URL" "$work/pages"
[ "$(cat "$work/pages/CNAME" 2>/dev/null)" = "$DOMAIN" ] || {
  echo "refusing to publish: $BRANCH has no CNAME for $DOMAIN" >&2
  exit 1
}

rsync -a --delete --exclude .git --exclude CNAME --exclude .nojekyll "$work/out/" "$work/pages/"

cd "$work/pages"
git add -A
if git diff --cached --quiet; then
  echo "$BRANCH is already up to date with main@$src_sha"
  exit 0
fi
git diff --cached --stat | tail -1

if [ "${DRY_RUN:-0}" = "1" ]; then
  echo "DRY_RUN: not pushing"
  exit 0
fi

git -c user.name="${GIT_AUTHOR_NAME:-0DeepResearch publisher}" \
  -c user.email="${GIT_AUTHOR_EMAIL:-publisher@0deepresearch.com}" \
  commit --quiet -m "Publish 0DeepResearch static site from main@$src_sha

Built with Hugo $hugo_version ($pages pages)."
git push --quiet origin "$BRANCH"
echo "published main@$src_sha to $BRANCH"
