#!/usr/bin/env bash
# Level 1, fast development checks (docs/TESTING.md): what your working tree
# changed since BASE, tested by what it reaches. Runs in your checkout and
# leaves it as it was; nothing is built into it.
#
#   scripts/test-fast.sh [BASE]          BASE defaults to main
#   scripts/test-fast.sh --all [BASE]    every package, as if everything changed
#
# It runs: Go vet and a compile of both modules; the tests of every Go
# package the change reaches (scripts/lib/classify.mjs follows the import
# graph, test imports included), in parallel processes; the frontend tests
# and a production frontend build to a temporary directory; the gate's own
# tests and the test scripts' tests when those changed. It does not run the
# race detector, regenerate fixtures or goldens, build the Wails app or drive
# the window: scripts/test-acceptance.sh does, and scripts/gate.sh before a
# release. Exit 0 only if everything passed, the checkout is unchanged and
# the real save folders are unchanged.
#
# Settings: FAST_JOBS (parallel test processes; default CPUs - 1),
# FAST_STAGE_TIMEOUT (seconds a stage may run; default 900).

source "$(dirname "$0")/lib/common.sh"

ALL=""
[ "${1:-}" = "--all" ] && { ALL=1; shift; }
case "${1:-}" in -h|--help) sed -n '2,22p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;; -*) echo "usage: scripts/test-fast.sh [--all] [BASE]"; exit 2 ;; esac
BASE=${1:-main}
tiers_init fast
JOBS=${FAST_JOBS:-$(( $(sysctl -n hw.ncpu 2>/dev/null || nproc 2>/dev/null || echo 4) - 1 ))}
LIMIT=${FAST_STAGE_TIMEOUT:-900}
ROOT=$TIERS_ROOT
git -C "$ROOT" rev-parse --verify --quiet "$BASE^{commit}" >/dev/null || die_pre "BASE '$BASE' does not name a commit"

say "== Istary fast checks: the working tree against $BASE ($(git -C "$ROOT" rev-parse --short "$BASE"))"
PLAN_FILE="$WORK/plan.json"
if [ -n "$ALL" ]; then
  echo "go.mod" >"$WORK/changed"   # Go dependencies: the classifier's "everything"
else
  changed_files "$BASE" >"$WORK/changed" || die_pre "cannot list the changed files"
fi
node "$ROOT/scripts/lib/classify.mjs" "$ROOT" <"$WORK/changed" >"$PLAN_FILE" || die_pre "cannot classify the changes"
print_plan

# The checkout as it was: a level-1 run must leave it so.
tree_state() { { git -C "$ROOT" status --porcelain --untracked-files=all && git -C "$ROOT" diff; } | shasum; }
TREE_BEFORE=$(tree_state)
check "real saves: recorded" saves_before

ROOT_PKGS=$(plan_get 'p.goPackages.filter((x) => x.module === ".").map((x) => x.pkg)')
GUI_PKGS=$(plan_get 'p.goPackages.filter((x) => x.module === "gui").map((x) => x.pkg)')
if [ -n "$ROOT_PKGS$GUI_PKGS" ]; then
  stage "go: compile both modules" "$ROOT" "$LIMIT" bash -c 'go build ./... && cd gui && go build -o /dev/null .'
  [ -n "$ROOT_PKGS" ] && stage "go: vet (root)" "$ROOT" "$LIMIT" go vet $ROOT_PKGS
  [ -n "$GUI_PKGS" ] && stage "go: vet (gui)" "$ROOT/gui" "$LIMIT" go vet $GUI_PKGS
  targets=()
  for p in $ROOT_PKGS; do targets+=(".:$p"); done
  for p in $GUI_PKGS; do targets+=("gui:$p"); done
  stage "go: tests ($(echo $ROOT_PKGS $GUI_PKGS | wc -w | tr -d ' ') packages, parallel)" "$ROOT" "$LIMIT" \
    node scripts/lib/gotest.mjs --jobs "$JOBS" --work "$WORK/gotest" --timings "$TIMINGS_CACHE" --baseline "$TIMINGS_BASELINE" "${targets[@]}"
fi
if [ "$(plan_get p.makeBuild)" = true ]; then
  stage "terminal game: build" "$ROOT" "$LIMIT" go build -o "$WORK/istary" ./cmd/istary
fi
if [ "$(plan_get p.frontend)" = true ]; then
  [ -d "$ROOT/gui/frontend/node_modules" ] || stage "frontend: dependencies (npm ci)" "$ROOT/gui/frontend" "$LIMIT" npm ci --prefer-offline --no-audit --no-fund
  stage "frontend: tests" "$ROOT/gui/frontend" "$LIMIT" npm test
  if [ "$(plan_get p.frontendBuild)" = true ]; then
    stage "frontend: production build (to a temporary directory)" "$ROOT/gui/frontend" "$LIMIT" npx vite build --outDir "$WORK/dist" --emptyOutDir --logLevel warn
  fi
fi
[ "$(plan_get p.gateScripts)" = true ] && stage "gate: its own tests" "$ROOT" "$LIMIT" bash scripts/gate_test.sh
[ "$(plan_get p.testScripts)" = true ] && stage "test scripts: their own tests" "$ROOT" "$LIMIT" bash scripts/test-scripts_test.sh

check "checkout unchanged" eval '[ "$(tree_state)" = "$TREE_BEFORE" ] || { say "   the checkout changed:"; git -C "$ROOT" status --short | sed "s/^/   | /"; false; }'
check "real saves unchanged" saves_unchanged

[ "$(plan_get p.wailsBuild)" = true ] && note "the Wails app is not built here: scripts/test-acceptance.sh builds it"
if [ "$(plan_get 'p.race.length')" != 0 ]; then note "race tests are due ($(plan_get 'p.race.map((x) => x.pkg.split("/").pop()).join(", ")')): scripts/test-acceptance.sh runs them"; fi
if [ "$(plan_get p.live.length)" != 0 ]; then note "the window is not driven here: scripts/test-acceptance.sh runs the live checks ($(plan_get 'p.live.join(", ")'))"; fi
[ "$(plan_get p.gateRequired)" = true ] && note "the full regression gate is due before release: $(plan_get 'p.gateReasons.join("; ")')"
[ -z "$ROOT_PKGS$GUI_PKGS" ] && [ "$(plan_get p.frontend)" != true ] && note "no Go package or frontend code changed: nothing to test"
exit 0
