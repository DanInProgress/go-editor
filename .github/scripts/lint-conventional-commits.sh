#!/usr/bin/env bash
# Checks that the pull request title and every commit in the pull request
# follow the Conventional Commits format documented in CONTRIBUTING.md.
set -euo pipefail

# Keep this list in sync with CONTRIBUTING.md and with renovate.json's
# semanticCommitType.
readonly TYPES='build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test'
readonly PATTERN="^(${TYPES})(\([a-z0-9._/-]+\))?!?: .+"
readonly MAX_SUBJECT=100

failed=0

check() {
  local what="$1" subject="$2"

  if [[ "$subject" =~ ^(Revert|Merge)\  ]]; then
    echo "skip: $what is a revert or merge commit: $subject"
    return
  fi
  if [[ ! "$subject" =~ $PATTERN ]]; then
    echo "::error::$what does not follow Conventional Commits: $subject"
    failed=1
    return
  fi
  if (( ${#subject} > MAX_SUBJECT )); then
    echo "::error::$what is ${#subject} characters, over the ${MAX_SUBJECT} character limit: $subject"
    failed=1
    return
  fi
  echo "ok: $what: $subject"
}

check "pull request title" "$PR_TITLE"

while IFS= read -r line; do
  [ -n "$line" ] || continue
  sha="${line%% *}"
  check "commit ${sha:0:8}" "${line#* }"
done < <(git log --no-merges --format='%H %s' "${BASE_SHA}..${HEAD_SHA}")

if (( failed )); then
  cat <<'MSG'

Expected format: <type>[optional scope][!]: <description>

  feat: add Editor.LaunchTempFile suffix handling
  fix(term): restore terminal state when /dev/tty is unavailable
  refactor!: drop the klog dependency

See CONTRIBUTING.md for the full list of types.
MSG
  exit 1
fi
