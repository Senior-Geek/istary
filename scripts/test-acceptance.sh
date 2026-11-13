#!/usr/bin/env bash
# Level 2, milestone acceptance (docs/TESTING.md): a commit, in a temporary
# worktree outside your checkout, checked as a milestone is accepted.
#
#   scripts/test-acceptance.sh [options] [BASE [CANDIDATE]]
#     BASE defaults to main, CANDIDATE to HEAD
#   --race            race tests on every package with concurrent code
#                     (engine, cmd/istary, gui), whatever changed
#   --no-live         do not drive the window
#   --sizes "WxH..."  window sizes for the live checks (default "980x692 1440x900")
#   --suites a,b      the live suites (default: those the change calls for)
#
# It runs: vet; every Go test of both modules (in parallel processes) and
# the frontend tests; the test scripts' own tests (and the gate's, if it
# changed); `make build`, the production frontend and the Wails app, with
# codesign; race tests on the packages with concurrent code when the change
# reaches concurrency or saving (or with --race); and the live checks, which
# drive the window on a seeded scratch library at each size. When CANDIDATE
# is BASE's merge base itself (nothing changed), it checks as if everything
# had. It does not run the root race tests (the app package alone takes over
# an hour under -race), shuffled runs or fixture regeneration: those are the
# full gate's (scripts/gate.sh).
#
# Exit 0 only if every stage passed, the checkout is as it was, no process it
# started is left, and the real save folders are unchanged. Run it from a
# clean checkout on macOS (Apple silicon), with Google Chrome for the live
# checks.
#
# Settings: ACCEPT_JOBS (parallel test processes), ACCEPT_STAGE_TIMEOUT
# (seconds a stage may run; default 1200), WAILS, CHROME.

source "$(dirname "$0")/lib/common.sh"

RACE_ALL="" LIVE=1 SIZES="980x692 1440x900" SUITES=""
while [ $# -gt 0 ]; do
  case $1 in
    --race) RACE_ALL=1 ;;
    --no-live) LIVE="" ;;
    --sizes) SIZES=${2:-}; shift ;;
    --suites) SUITES=${2:-}; shift ;;
    -h|--help) sed -n '2,31p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    -*) echo "usage: scripts/test-acceptance.sh [--race] [--no-live] [--sizes \"WxH ...\"] [--suites a,b] [BASE [CANDIDATE]]"; exit 2 ;;
    *) break ;;
  esac
  shift
done
BASE=${1:-main}
CANDIDATE=${2:-HEAD}
tiers_init acceptance
ROOT=$TIERS_ROOT
JOBS=${ACCEPT_JOBS:-$(( $(sysctl -n hw.ncpu 2>/dev/null || echo 4) - 1 ))}
LIMIT=${ACCEPT_STAGE_TIMEOUT:-1200}

[ "$(uname -s)/$(uname -m)" = Darwin/arm64 ] || die_pre "acceptance builds the macOS app (darwin/arm64) and checks it with codesign; this is $(uname -s)/$(uname -m). Use scripts/test-fast.sh here."
need codesign "verifying the Wails build"
need make "make build"
if [ -z "${WAILS:-}" ]; then WAILS=$(command -v wails 2>/dev/null || echo "$(go env GOPATH)/bin/wails"); fi
[ -x "$WAILS" ] || die_pre "the Wails CLI was not found (set WAILS=/path/to/wails)"
CHROME=${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}
if [ -n "$LIVE" ] && [ ! -x "$CHROME" ]; then die_pre "Google Chrome is needed for the live checks (set CHROME=..., or --no-live)"; fi
dirty=$(git -C "$ROOT" status --porcelain --untracked-files=normal)
[ -z "$dirty" ] || die_pre "the working tree is not clean; acceptance tests a commit, so commit or set aside your changes first (it never stashes or resets):
$dirty"
BASE_SHA=$(git -C "$ROOT" rev-parse --verify --quiet "$BASE^{commit}") || die_pre "BASE '$BASE' does not name a commit"
CAND_SHA=$(git -C "$ROOT" rev-parse --verify --quiet "$CANDIDATE^{commit}") || die_pre "CANDIDATE '$CANDIDATE' does not name a commit"
START_HEAD=$(git -C "$ROOT" rev-parse HEAD)
START_BRANCH=$(git -C "$ROOT" symbolic-ref -q HEAD || echo "(detached)")

say "== Istary acceptance: $CANDIDATE ($(git -C "$ROOT" rev-parse --short "$CAND_SHA")) against $BASE ($(git -C "$ROOT" rev-parse --short "$BASE_SHA"))"
PLAN_FILE="$WORK/plan.json"
changed_files "$BASE_SHA" "$CAND_SHA" >"$WORK/changed" || die_pre "cannot list the changed files"
if [ ! -s "$WORK/changed" ]; then
  echo "go.mod" >"$WORK/changed"
  note "nothing changed between $BASE and $CANDIDATE: checked as if everything had"
fi
node "$ROOT/scripts/lib/classify.mjs" "$ROOT" <"$WORK/changed" >"$PLAN_FILE" || die_pre "cannot classify the changes"
print_plan
check "real saves: recorded" saves_before

# The candidate in a worktree of its own.
WT="$WORK/wt"
add_worktree() { git -C "$ROOT" worktree add --detach "$WT" "$CAND_SHA" >/dev/null 2>&1; }
CLEANUPS+=('[ ! -d "$WT" ] || git -C "$ROOT" worktree remove --force "$WT"')
check "worktree of $(git -C "$ROOT" rev-parse --short "$CAND_SHA")" add_worktree

stage "frontend: dependencies (npm ci)" "$WT/gui/frontend" "$LIMIT" npm ci --prefer-offline --no-audit --no-fund
stage "go: vet (root)" "$WT" "$LIMIT" go vet ./...
stage "go: vet (gui)" "$WT/gui" "$LIMIT" go vet ./...
stage "go: every test, both modules (parallel)" "$WT" "$LIMIT" \
  node scripts/lib/gotest.mjs --jobs "$JOBS" --work "$WORK/gotest" --timings "$TIMINGS_CACHE" --baseline "$TIMINGS_BASELINE" \
  .:./... gui:./...
stage "frontend: tests" "$WT/gui/frontend" "$LIMIT" npm test
stage "test scripts: their own tests" "$WT" "$LIMIT" bash scripts/test-scripts_test.sh
[ "$(plan_get p.gateScripts)" = true ] && stage "gate: its own tests" "$WT" "$LIMIT" bash scripts/gate_test.sh
stage "terminal game: make build" "$WT" "$LIMIT" make build

RACE=$(plan_get 'p.race.map((x) => (x.module === "gui" ? "gui:" : ".:") + x.pkg)')
[ -n "$RACE_ALL" ] && RACE=$'.:github.com/Senior-Geek/istary/engine\n.:github.com/Senior-Geek/istary/cmd/istary\ngui:github.com/Senior-Geek/istary/gui'
if [ -n "$RACE" ]; then
  stage "go: race tests ($(echo $RACE | sed 's#[^ ]*/##g' | tr '\n' ' '))" "$WT" "$LIMIT" \
    node scripts/lib/gotest.mjs --race --jobs "$JOBS" --work "$WORK/gotest-race" --timings "$TIMINGS_CACHE" --baseline "$TIMINGS_BASELINE" $RACE
else
  note "no race tests: the change reaches no concurrent code or saving (--race runs them anyway)"
fi

stage "wails build (production frontend and app)" "$WT/gui" "$LIMIT" "$WAILS" build -platform darwin/arm64
stage "codesign verify" "$WT/gui" "$LIMIT" codesign --verify --verbose build/bin/Istary.app

if [ -n "$LIVE" ]; then
  [ -n "$SUITES" ] || SUITES=$(plan_get 'p.live.join(",")')
  if [ -z "$SUITES" ]; then
    note "no live checks: the change does not reach the window"
  else
    for size in $SIZES; do
      lib="$WORK/library-$size"
      stage "live: seed a scratch library ($size)" "$WT/gui" "$LIMIT" env ISTARY_ACCEPTANCE_LIBRARY="$lib" go test . -run '^TestWriteAcceptanceLibrary$' -count=1
      stage "live: $SUITES at $size" "$WT" "$LIMIT" node scripts/acceptance/live.mjs --gui "$WT/gui" --library "$lib" \
        --out "$LOG_DIR/live-$size" --size "$size" --suites "$SUITES" --wails "$WAILS" --chrome "$CHROME"
    done
  fi
else
  note "live checks skipped (--no-live)"
fi

# Nothing of ours still running: every process whose command names this
# run's directory (its worktree, library, profile) is ours, and none may be left.
ours_left() { pgrep -f "$WORK/" >/dev/null 2>&1 && { say "   still running: $(pgrep -fl "$WORK/" | head -5)"; return 1; }; return 0; }
check "no process of this run left" ours_left
remove_wt() { git -C "$ROOT" worktree remove --force "$WT" && [ ! -d "$WT" ]; }
check "worktree removed" remove_wt
repository_unchanged() {
  [ "$(git -C "$ROOT" rev-parse HEAD)" = "$START_HEAD" ] && [ "$(git -C "$ROOT" symbolic-ref -q HEAD || echo "(detached)")" = "$START_BRANCH" ] &&
    [ -z "$(git -C "$ROOT" status --porcelain --untracked-files=normal)" ] && ! git -C "$ROOT" worktree list --porcelain | grep -qF "worktree $WORK/"
}
check "checkout unchanged" repository_unchanged
check "real saves unchanged" saves_unchanged
[ "$(plan_get p.gateRequired)" = true ] && note "the full regression gate is due before release: $(plan_get 'p.gateReasons.join("; ")')"
note "the full gate (scripts/gate.sh) is mandatory before any release tag, whatever this says"
exit 0
