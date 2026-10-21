# How 0deepresearch.com is published

The site is served by GitHub Pages from the `gh-pages` branch. GitHub Actions cannot run for this account (billing lock), so the
branch is updated by a Woodpecker pipeline (`.woodpecker.yml`) on every push to `main`, including the CMS "Create Posts" commits:

1. build with Hugo (version pinned in the image tag, same as `.github/workflows/deploy-hugo.yml`);
2. `scripts/publish-gh-pages.sh` syncs the build into a clone of `gh-pages` (keeping `CNAME` and `.nojekyll`), commits and pushes with a
   fast-forward only. It refuses a build with fewer than 50 pages or a `gh-pages` without the `CNAME`.

## One-time setup (done by a person)

- Woodpecker (ci.0exec.com): activate `baditaflorin/0deepresearch`.
- Woodpecker, repository secrets: `gh_deploy_key` = the private half of the deploy key `woodpecker-0deepresearch` (read-write, registered
  under the repository's Settings, Deploy keys), available to the **push** event only.

## Publish by hand (fallback)

```bash
git clone --recurse-submodules https://github.com/baditaflorin/0deepresearch.git && cd 0deepresearch
REPO_URL=https://github.com/baditaflorin/0deepresearch.git scripts/publish-gh-pages.sh      # DRY_RUN=1 shows the change first
```

The build runs `hugo --gc`, which can delete tracked files under `resources/_gen` in your working tree; `git checkout -- resources` restores them. Do not commit those deletions.

## Rotate the deploy key

Generate a new key pair, add its public half as a deploy key (read-write), replace the Woodpecker secret, delete the old deploy key.

## Roll back

`git revert` the last commit on `gh-pages` and push, or push the previous tree as a new commit; the branch only ever moves forward.
