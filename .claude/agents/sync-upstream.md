# Sync Upstream

Sync code from the upstream llm-d org to a user's fork and open a PR to opendatahub-io.

## Input

Ask the user: **"Sync to upstream/main HEAD or a specific commit SHA?"**

- If the user provides a **commit SHA**, use that as the target commit
- If no SHA is given, default to `upstream/main` HEAD

## Workflow

1. **Pre-flight checks**: Verify `origin` remote points to the user's fork (not upstream or opendatahub)
2. **Fetch remotes**: Add/update `upstream` and `opendatahub` remotes, fetch `upstream/main` and `opendatahub/main_2`
3. **Resolve target commit**: Use the user-provided SHA, or default to `upstream/main` HEAD. Verify it exists on `upstream/main`
4. **Check for duplicates**: If a sync branch or PR for this SHA already exists, inform the user and stop
5. **Create sync branch**: Create `sync/upstream-<short_sha>` based on `opendatahub/main_2`
6. **Merge upstream**: Merge the target upstream commit into the sync branch. Resolve conflicts if needed
7. **Push branch**: Push to `origin`
8. **Confirm PR creation**: Ask the user whether to open a PR or stop with the branch pushed only
9. **Open PR** (if confirmed): Open a PR to `opendatahub-io/llm-d-inference-scheduler` targeting `main_2`

## Commands

```bash
# 0. Save current branch to restore later
ORIGINAL_BRANCH=$(git rev-parse --abbrev-ref HEAD)

# 1. Pre-flight: verify origin is the user's fork
ORIGIN_URL=$(git remote get-url origin)
if echo "${ORIGIN_URL}" | grep -qE '(llm-d/llm-d-inference-scheduler|opendatahub-io/llm-d-inference-scheduler)'; then
  echo "Error: origin remote points to upstream or opendatahub, not your fork"
  exit 1
fi

# 2. Set up remotes and fetch
git remote add upstream https://github.com/llm-d/llm-d-inference-scheduler.git 2>/dev/null || true
git remote set-url upstream https://github.com/llm-d/llm-d-inference-scheduler.git
git remote add opendatahub https://github.com/opendatahub-io/llm-d-inference-scheduler.git 2>/dev/null || true
git remote set-url opendatahub https://github.com/opendatahub-io/llm-d-inference-scheduler.git
git fetch upstream main
git fetch opendatahub main_2

# 3. Resolve target commit
TARGET_COMMIT="${USER_SHA:-upstream/main}"
FULL_SHA=$(git rev-parse "${TARGET_COMMIT}")
SHORT_SHA=$(git rev-parse --short "${TARGET_COMMIT}")
git merge-base --is-ancestor "${FULL_SHA}" upstream/main || { echo "Error: commit not on upstream/main"; exit 1; }

# 4. Check for duplicates
BRANCH="sync/upstream-${SHORT_SHA}"
if git show-ref --verify --quiet "refs/heads/${BRANCH}" || git show-ref --verify --quiet "refs/remotes/origin/${BRANCH}"; then
  echo "Warning: branch ${BRANCH} already exists"
  # Ask user whether to force-update or skip
fi

# 5. Create branch from opendatahub/main_2
git checkout -b "${BRANCH}" opendatahub/main_2

# 6. Merge upstream commit into the sync branch
git merge --no-ff "${FULL_SHA}" --no-edit -m "Sync upstream llm-d/llm-d-inference-scheduler ${SHORT_SHA}"
```

## Conflict Resolution

If step 6 produces merge conflicts:

1. List conflicted files with `git diff --name-only --diff-filter=U`
2. Show the conflicts to the user
3. Attempt to resolve trivial conflicts automatically (whitespace, import order)
4. For non-trivial conflicts, show the diff and ask the user how to resolve each file
5. After all conflicts are resolved:
   ```bash
   git add -u
   git commit --no-edit
   ```

## Push and Open PR

```bash
# 7. Push to origin (user's fork)
git push -u origin "${BRANCH}"
```

**Before creating the PR, ask the user whether they want to open a PR to opendatahub-io or skip.**

If the user chooses to open the PR:

```bash
# 8. Open PR to opendatahub-io
FORK_OWNER=$(git remote get-url origin | sed -E 's|.*[:/]([^/]+)/[^/]+(.git)?$|\1|')

gh pr create \
  --repo opendatahub-io/llm-d-inference-scheduler \
  --base main_2 \
  --head "${FORK_OWNER}:${BRANCH}" \
  --title "[sync] upstream llm-d main branch ${SHORT_SHA} [$(date -u +%Y-%m-%d)]" \
  --body "Syncs llm-d/llm-d-inference-scheduler main branch into ODH main_2 branch.

Upstream commit: https://github.com/llm-d/llm-d-inference-scheduler/commit/${FULL_SHA}"
```

Regardless of whether the PR was created or skipped:

```bash
# 9. Return to the original branch
git checkout "${ORIGINAL_BRANCH}"
```

If `gh pr create` fails, inform the user that the branch has been pushed to `origin/${BRANCH}` and ask them to create the PR manually at `https://github.com/opendatahub-io/llm-d-inference-scheduler/compare/main_2...${FORK_OWNER}:${BRANCH}`.

## Error Handling

- If the user-provided SHA does not exist on `upstream/main`, report the error and ask for a valid SHA
- If conflicts cannot be resolved, abort the merge (`git merge --abort`), clean up (`git checkout "${ORIGINAL_BRANCH}" && git branch -D "${BRANCH}"`), and inform the user
- If the branch already exists, ask the user whether to force-update or skip
- On any failure after branch creation, clean up with `git checkout "${ORIGINAL_BRANCH}" && git branch -D "${BRANCH}"`
- Always return the PR URL on success
