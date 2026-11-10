#!/usr/bin/env bash
# The regression gate: every commit after BASE alone (vet, Go tests, npm),
# then the tip in full (vet, tests, shuffled order, race, the build, the
# GUI module, the frontend, fixture regeneration and the Wails build).
#
#   scripts/gate.sh BASE [BRANCH]    e.g. scripts/gate.sh v0.3.0-inn temple
#
# Run it from a clean working tree; it fails on the first failing step.
set -e
BASE=${1:?usage: scripts/gate.sh BASE [BRANCH]}
BRANCH=${2:-HEAD}
ROOT=$(git rev-parse --show-toplevel)
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
cd "$ROOT"
echo "== head"; git log --oneline -1; git status --short
for c in $(git rev-list --reverse "$BASE..$BRANCH"); do
  echo "== commit $(git log --oneline -1 "$c" | cut -c1-70) alone: root + gui + npm"
  git worktree add -q --detach "$WORK/wt-$c" "$c"
  (cd "$WORK/wt-$c" && go vet ./... && go test ./engine ./cmd/istary ./app -count=1 && cd gui && go vet ./... && go test ./... -count=1 && cd frontend && node --test src/lib/*.test.js 2>&1 | grep -E "^# (pass|fail)")
  git worktree remove --force "$WORK/wt-$c"
done
echo "== root vet"; go vet ./...
echo "== root test"; go test ./... -count=1
echo "== root shuffle"; go test ./... -count=1 -shuffle=on
echo "== make build"; make build >/dev/null && echo ok
# Under the race detector the app package alone takes over an hour (about
# 64 minutes at v0.4.0-temple), so 60 minutes is too tight.
echo "== root race"; go test -race ./... -count=1 -timeout 150m
cd gui
echo "== gui vet"; go vet ./...
echo "== gui test"; go test ./... -count=1
echo "== gui shuffle"; go test ./... -count=1 -shuffle=on
echo "== gui race"; go test -race ./... -count=1 -timeout 30m
echo "== npm"; (cd frontend && npm test 2>&1 | grep -E "^# (pass|fail)")
echo "== fixtures regenerate"
B=$(git -C "$ROOT" status --short; git -C "$ROOT" diff | shasum)
ISTARY_UPDATE_FIXTURES=1 go test . -count=1 >/dev/null
(cd .. && go test ./cmd/istary -count=1 -update >/dev/null)
A=$(git -C "$ROOT" status --short; git -C "$ROOT" diff | shasum)
[ "$A" = "$B" ] && echo "fixtures unchanged" || { echo "FIXTURES CHANGED"; git -C "$ROOT" status --short; exit 1; }
echo "== wails build"; ~/go/bin/wails build -platform darwin/arm64 >/dev/null 2>&1 && codesign --verify build/bin/*.app && echo ok
git -C "$ROOT" checkout -- gui/frontend/wailsjs/runtime 2>/dev/null || true
echo "== final status"; git -C "$ROOT" status --short; echo "(status end)"
echo "== ALL DONE"
