# How 0deepresearch.com is published

The site is served by GitHub Pages from the `gh-pages` branch. GitHub Actions cannot run for this account (billing lock), so the
branch is updated by a Woodpecker pipeline (`.woodpecker.yml`) on every push to `main`, including the CMS "Create Posts" commits:

1. build with Hugo (version pinned in the image tag, same as `.github/workflows/deploy-hugo.yml`);
2. `scripts/publish-gh-pages.sh` syncs the build into a clone of `gh-pages` (keeping `CNAME` and `.nojekyll`), commits and pushes with a
   fast-forward only. It refuses a build with fewer than 50 pages or a `gh-pages` without the `CNAME`.

## One-time setup (done 2026-10-21, through the Woodpecker API)

- Activated: `POST /api/repos?forge_remote_id=974527902` on ci.0exec.com (repository id 438 there; Woodpecker also created the GitHub webhook).
- Secret: `POST /api/repos/438/secrets` with name `gh_deploy_key`, the private half of the deploy key `woodpecker-0deepresearch` (read-write, under the
  repository's Settings, Deploy keys), for the `push` event only. Do not add an image filter: Woodpecker then lets only plugin steps read the secret, and
  this pipeline's step is a normal `commands` step (the first run failed with exactly that message).
- Both calls went through `fleet-api` (the token is read from the Builder over ssh and never printed). To redo it, or to rotate the key: generate a
  new pair, add its public half as a deploy key, `PATCH /api/repos/438/secrets/gh_deploy_key` with the new value, then delete the old deploy key.

## Publish by hand (fallback)

```bash
git clone --recurse-submodules https://github.com/baditaflorin/0deepresearch.git && cd 0deepresearch
REPO_URL=https://github.com/baditaflorin/0deepresearch.git scripts/publish-gh-pages.sh      # DRY_RUN=1 shows the change first
```

The build runs `hugo --gc`, which can delete tracked files under `resources/_gen` in your working tree; `git checkout -- resources` restores them. Do not commit those deletions.

## Rotate the deploy key

Generate a new key pair, add its public half as a deploy key (read-write), replace the Woodpecker secret, delete the old deploy key.

## What a run looks like

The first run (2026-10-21, run 3) waited about 14 minutes in the queue behind other repositories' jobs (the agent takes one workflow at a time, 66 were waiting)
and took about a minute to run. The first publish from Hugo 0.147.2 changed 66 files (the previous builds came from 0.149); later runs with no site change say
"gh-pages is already up to date".

## Roll back

`git revert` the last commit on `gh-pages` and push, or push the previous tree as a new commit; the branch only ever moves forward.
