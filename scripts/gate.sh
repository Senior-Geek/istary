#!/usr/bin/env bash
# The regression gate (docs/REGRESSION-GATE.md): every commit after BASE up
# to CANDIDATE alone (root vet and tests, GUI vet and tests, frontend
# tests), then CANDIDATE in full (vet, tests, shuffled order, the build,
# race, the GUI module, the frontend, fixture regeneration and the Wails
# build). Each commit is checked out in its own temporary worktree, so the
# checkout the gate is run from is never built in or changed.
#
#   scripts/gate.sh BASE [CANDIDATE]     e.g. scripts/gate.sh main g1-regression-gate
#
# CANDIDATE defaults to HEAD. BASE must be a proper ancestor of CANDIDATE.
# Run it from a clean checkout of Istary on macOS (Apple silicon). The exit
# status is 0 only when every stage passed and every worktree the gate made
# was removed: 1 a stage failed, 2 the gate could not start (usage, platform,
# dependencies, repository state), 4 cleanup failed, 124 a stage timed out,
# 130/143/129 interrupted (INT/TERM/HUP).
#
# Settings (environment):
#   GATE_LOG_DIR             where the logs go (default: a new temporary directory)
#   GATE_STAGE_TIMEOUT       seconds any one stage may run (default 3600)
#   GATE_ROOT_RACE_TIMEOUT   seconds for the root race tests (default 9000: the
#                            app package alone takes about 65 minutes under -race)
#   WAILS                    the Wails CLI (default: wails on PATH, else $GOPATH/bin/wails)

set -u -o pipefail

usage() { echo "usage: scripts/gate.sh BASE [CANDIDATE]   (see docs/REGRESSION-GATE.md)"; }

# --- Results -----------------------------------------------------------------

EXIT_FAIL=1 EXIT_PRECONDITION=2 EXIT_CLEANUP=4 EXIT_TIMEOUT=124
STAGES=0           # stages run
FAILED_STAGE=""    # the first stage that did not pass
FAILED_RESULT=""   # FAIL or TIMEOUT
STATUS=0           # the gate's exit status so far
CLEANUP="not needed"
INTERRUPTED=""
COMMITS_TESTED=0 COMMITS_TOTAL=0 TIP_DONE="no"
BASE_SHA="" CAND_SHA=""
LOG_DIR="" WT_DIR=""
WORKTREES=()       # worktrees this run created and has not removed
CHILD=""           # the running stage's process group
WATCHDOG=""

say() { printf '%s\n' "$*"; [ -n "$LOG_DIR" ] && [ -d "$LOG_DIR" ] && printf '%s\n' "$*" >>"$LOG_DIR/gate.log"; return 0; }
die_pre() { say "GATE CANNOT START: $*" >&2; STATUS=$EXIT_PRECONDITION; exit "$EXIT_PRECONDITION"; }
now() { date +%s; }
dur() { local s=$1; printf '%dm%02ds' $((s / 60)) $((s % 60)); }

# --- Cleanup and signals -------------------------------------------------------

# remove_worktree removes one worktree this run created; it never touches
# any other worktree and never prunes.
remove_worktree() {
  local wt=$1 out list
  list=$(git -C "$ROOT" worktree list --porcelain 2>&1) || { say "  cleanup: git worktree list failed: $list"; return 1; }
  if ! grep -qxF "worktree $wt" <<<"$list"; then
    # Never registered (adding it failed): only a partial directory of the
    # gate's own may be left.
    case $wt in "$WT_DIR"/wt-*) [ ! -e "$wt" ] || rm -rf -- "$wt" ;; esac
  elif ! out=$(git -C "$ROOT" worktree remove --force "$wt" 2>&1); then
    say "  cleanup: could not remove worktree $wt: $out"
    return 1
  fi
  local keep=() w
  for w in ${WORKTREES[@]+"${WORKTREES[@]}"}; do [ "$w" = "$wt" ] || keep+=("$w"); done
  WORKTREES=(${keep[@]+"${keep[@]}"})
  return 0
}

stop_child() {
  if [ -n "$WATCHDOG" ]; then kill -KILL "$WATCHDOG" 2>/dev/null; wait "$WATCHDOG" 2>/dev/null; WATCHDOG=""; fi
  if [ -n "$CHILD" ]; then
    # The group, and the process itself in case it has not made its group yet.
    kill -TERM -- "-$CHILD" 2>/dev/null
    kill -TERM "$CHILD" 2>/dev/null
    local i=0
    while kill -0 -- "-$CHILD" 2>/dev/null && [ $i -lt 50 ]; do sleep 0.2; i=$((i + 1)); done
    kill -KILL -- "-$CHILD" 2>/dev/null
    kill -KILL "$CHILD" 2>/dev/null
    wait "$CHILD" 2>/dev/null
    CHILD=""
  fi
}

CLEANED=""
cleanup() {
  [ -n "$CLEANED" ] && return 0
  CLEANED=1
  stop_child
  local ok=1 wt
  for wt in ${WORKTREES[@]+"${WORKTREES[@]}"}; do remove_worktree "$wt" || ok=""; done
  if [ -n "$WT_DIR" ] && [ -d "$WT_DIR" ]; then
    rmdir "$WT_DIR" 2>/dev/null || { say "  cleanup: $WT_DIR is not empty"; ok=""; }
  fi
  # The gate's own worktrees must be gone from the repository's list.
  if [ -n "$WT_DIR" ]; then
    local list
    if ! list=$(git -C "$ROOT" worktree list --porcelain 2>&1); then
      say "  cleanup: git worktree list failed: $list"; ok=""
    elif grep -qF "worktree $WT_DIR/" <<<"$list"; then
      say "  cleanup: a worktree under $WT_DIR is still registered"; ok=""
    fi
  fi
  if [ -n "$WT_DIR" ]; then
    if [ -n "$ok" ]; then CLEANUP="ok"; else CLEANUP="FAILED"; fi
  fi
  if [ "$CLEANUP" = "FAILED" ] && [ "$STATUS" -eq 0 ]; then STATUS=$EXIT_CLEANUP; fi
}

summary() {
  [ -n "$LOG_DIR" ] || return 0
  local result="PASS"
  [ "$STATUS" -ne 0 ] && result="FAIL"
  [ -n "$INTERRUPTED" ] && result="INTERRUPTED"
  say ""
  say "== summary"
  say "  base:       ${BASE_SHA:-?}"
  say "  candidate:  ${CAND_SHA:-?}"
  say "  commits:    $COMMITS_TESTED of $COMMITS_TOTAL tested alone"
  say "  tip checks: $TIP_DONE"
  say "  stages run: $STAGES"
  if [ -n "$FAILED_STAGE" ]; then say "  failed:     $FAILED_STAGE ($FAILED_RESULT)"; else say "  failed:     none"; fi
  [ -n "$INTERRUPTED" ] && say "  interrupted by SIG$INTERRUPTED"
  say "  worktrees:  cleanup $CLEANUP"
  say "  logs:       $LOG_DIR"
  say "  exit:       $STATUS"
  say "== RESULT: $result"
}

on_exit() {
  local rc=$?
  trap '' INT TERM HUP # a second Ctrl-C must not cut cleanup short
  [ "$STATUS" -eq 0 ] && STATUS=$rc
  cleanup
  summary
  exit "$STATUS"
}
on_signal() {
  INTERRUPTED=$1
  case $1 in INT) STATUS=130 ;; TERM) STATUS=143 ;; HUP) STATUS=129 ;; esac
  say ""
  say "== interrupted (SIG$1): stopping the running stage and removing worktrees"
  exit "$STATUS" # runs on_exit
}
trap on_exit EXIT
trap 'on_signal INT' INT
trap 'on_signal TERM' TERM
trap 'on_signal HUP' HUP

# --- Stages ---------------------------------------------------------------------

# stage NAME DIR TIMEOUT CMD... runs CMD in DIR as its own process group,
# its output in its own log, and stops it (the group only) after TIMEOUT
# seconds. A stage passes only when CMD exits 0. Go's own -timeout panic
# counts as a timeout too.
stage() {
  local name=$1 dir=$2 limit=$3; shift 3
  STAGES=$((STAGES + 1))
  local n log marker start st result
  n=$(printf '%02d' "$STAGES")
  log="$LOG_DIR/$n-$(printf '%s' "$name" | tr -c 'A-Za-z0-9._-' '-').log"
  marker="$LOG_DIR/.timeout-$n"
  say "== [$n] $name"
  say "   \$ (cd ${dir#"$WT_DIR/"} && $*)"
  start=$(now)
  ( cd "$dir" && exec perl -e 'setpgrp(0, 0) or die "setpgrp: $!\n"; exec { $ARGV[0] } @ARGV or die "cannot run $ARGV[0]: $!\n"' "$@" ) >"$log" 2>&1 &
  CHILD=$!
  perl -e '($t, $g, $m) = @ARGV; sleep $t; open(my $f, ">", $m); close $f; kill "-TERM", $g; sleep 10; kill "-KILL", $g' "$limit" "$CHILD" "$marker" &
  WATCHDOG=$!
  wait "$CHILD"; st=$?
  kill -KILL "$WATCHDOG" 2>/dev/null; wait "$WATCHDOG" 2>/dev/null; WATCHDOG=""
  if [ -e "$marker" ]; then
    kill -KILL -- "-$CHILD" 2>/dev/null # anything left of the stopped group
    rm -f "$marker"
    result=TIMEOUT
  elif [ "$st" -eq 0 ]; then
    result=PASS
  elif grep -q '^panic: test timed out after' "$log"; then
    result=TIMEOUT
  else
    result=FAIL
  fi
  CHILD=""
  local took=$(( $(now) - start ))
  say "   $result in $(dur $took) (exit $st)   log: $log"
  if [ "$result" != PASS ]; then
    say "   --- last lines of the log:"
    tail -n 25 "$log" | sed 's/^/   | /'
    tail -n 25 "$log" >>"$LOG_DIR/gate.log"
    FAILED_STAGE="[$n] $name: $*"
    FAILED_RESULT=$result
    if [ "$result" = TIMEOUT ]; then STATUS=$EXIT_TIMEOUT; else STATUS=$EXIT_FAIL; fi
    exit "$STATUS"
  fi
}

# check NAME CMD... is a stage the gate decides itself (no process of its own).
check() {
  local name=$1; shift
  STAGES=$((STAGES + 1))
  local n; n=$(printf '%02d' "$STAGES")
  say "== [$n] $name"
  if "$@"; then
    say "   PASS"
  else
    say "   FAIL"
    FAILED_STAGE="[$n] $name"; FAILED_RESULT=FAIL; STATUS=$EXIT_FAIL
    [ "$CLEANUP" = FAILED ] && STATUS=$EXIT_CLEANUP
    exit "$STATUS"
  fi
}

# --- Arguments --------------------------------------------------------------------

if [ $# -lt 1 ] || [ $# -gt 2 ] || [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ] && exit 0
  exit "$EXIT_PRECONDITION"
fi
BASE=$1
CANDIDATE=${2:-HEAD}

# --- Platform and dependencies -----------------------------------------------------

# The Wails build is for macOS on Apple silicon (darwin/arm64) and is
# verified with codesign; the gate runs nowhere else.
[ "${BASH_VERSINFO[0]}" -gt 3 ] || { [ "${BASH_VERSINFO[0]}" -eq 3 ] && [ "${BASH_VERSINFO[1]}" -ge 2 ]; } ||
  die_pre "bash 3.2 or later is needed (this is $BASH_VERSION)"
os=$(uname -s 2>/dev/null) arch=$(uname -m 2>/dev/null)
[ "$os" = Darwin ] && [ "$arch" = arm64 ] ||
  die_pre "the gate runs on macOS on Apple silicon (darwin/arm64), for its Wails build; this is ${os:-?}/${arch:-?}. The Go and frontend tests run anywhere, but the full gate does not."

need() { command -v "$1" >/dev/null 2>&1 || die_pre "$1 is not installed or not on PATH ($2)"; }
need git "the gate's worktrees"
need go "the Go tests and builds"
need cc "cgo, for -race and the Wails build"
need node "the frontend tests"
need npm "the frontend dependencies and tests"
need make "make build"
need perl "running each stage as its own process group with a time limit"
need codesign "verifying the Wails build"
need shasum "fixture comparison"
need mktemp "temporary directories"

gitv=$(git --version | sed -n 's/^git version \([0-9]*\)\.\([0-9]*\).*/\1 \2/p')
set -- $gitv
{ [ "${1:-0}" -gt 2 ] || { [ "${1:-0}" -eq 2 ] && [ "${2:-0}" -ge 17 ]; }; } || die_pre "git 2.17 or later is needed for 'git worktree remove' ($(git --version))"
go version >/dev/null 2>&1 || die_pre "go does not run: $(go version 2>&1 | head -1)"
nodev=$(node --version 2>/dev/null | sed -n 's/^v\([0-9]*\).*/\1/p')
[ "${nodev:-0}" -ge 20 ] || die_pre "Node.js 20 or later is needed for the frontend build ($(node --version 2>&1))"
npm --version >/dev/null 2>&1 || die_pre "npm does not run"
if [ -z "${WAILS:-}" ]; then
  WAILS=$(command -v wails 2>/dev/null || true)
  [ -n "$WAILS" ] || WAILS="$(go env GOPATH 2>/dev/null)/bin/wails"
fi
[ -x "$WAILS" ] || die_pre "the Wails CLI was not found (looked for wails on PATH and $WAILS; set WAILS=/path/to/wails)"
"$WAILS" version >/dev/null 2>&1 || die_pre "the Wails CLI at $WAILS does not run"

STAGE_TIMEOUT=${GATE_STAGE_TIMEOUT:-3600}
RACE_TIMEOUT=${GATE_ROOT_RACE_TIMEOUT:-9000}
case "$STAGE_TIMEOUT$RACE_TIMEOUT" in *[!0-9]*) die_pre "GATE_STAGE_TIMEOUT and GATE_ROOT_RACE_TIMEOUT are whole seconds" ;; esac
[ "$STAGE_TIMEOUT" -gt 0 ] && [ "$RACE_TIMEOUT" -gt 0 ] || die_pre "timeouts must be positive"
# Go's -timeout fires first, with its own report; the stage limit is a backstop.
MARGIN=600
[ "$STAGE_TIMEOUT" -lt 1200 ] && MARGIN=$((STAGE_TIMEOUT / 2 + 1))

# --- The repository --------------------------------------------------------------------

ROOT=$(git rev-parse --show-toplevel 2>/dev/null) || die_pre "$(pwd) is not in a git repository"
[ "$(head -1 "$ROOT/go.mod" 2>/dev/null)" = "module github.com/Senior-Geek/istary" ] && [ -f "$ROOT/gui/wails.json" ] ||
  die_pre "$ROOT is not the Istary repository (go.mod module github.com/Senior-Geek/istary with gui/wails.json)"

# The checkout must be clean: the gate tests commits, so uncommitted work
# would not be tested, and the final check proves the checkout unchanged.
dirty=$(git -C "$ROOT" status --porcelain --untracked-files=normal 2>&1) || die_pre "git status failed: $dirty"
if [ -n "$dirty" ]; then
  say "GATE CANNOT START: the working tree at $ROOT is not clean. Commit or set aside these changes first (the gate never stashes, resets or cleans):"
  printf '%s\n' "$dirty" | sed 's/^/  /'
  exit "$EXIT_PRECONDITION"
fi

# resolve REF NAME sets RESOLVED to the commit REF names, or the gate
# stops. A ref must not look like an option and must name exactly one thing.
resolve() {
  local ref=$1 what=$2 err
  case $ref in -*|"") die_pre "the $what '$ref' is not a valid ref" ;; esac
  RESOLVED=$(git -C "$ROOT" rev-parse --verify --quiet "$ref^{commit}" 2>/dev/null) || die_pre "the $what '$ref' does not name a commit"
  err=$(git -C "$ROOT" rev-parse --verify "$ref^{commit}" 2>&1 >/dev/null)
  case $err in *ambiguous*) die_pre "the $what '$ref' is ambiguous (more than one ref has that name); use a full ref such as refs/heads/$ref or refs/tags/$ref" ;; esac
}
resolve "$BASE" base; BASE_SHA=$RESOLVED
resolve "$CANDIDATE" candidate; CAND_SHA=$RESOLVED
[ "$BASE_SHA" != "$CAND_SHA" ] || die_pre "the base and candidate are the same commit ($BASE_SHA): there is nothing to gate"
git -C "$ROOT" merge-base --is-ancestor "$BASE_SHA" "$CAND_SHA" ||
  die_pre "the base $BASE ($BASE_SHA) is not an ancestor of the candidate $CANDIDATE ($CAND_SHA)"
COMMITS=$(git -C "$ROOT" rev-list --reverse "$BASE_SHA..$CAND_SHA") || die_pre "git rev-list failed"
COMMITS_TOTAL=$(printf '%s\n' "$COMMITS" | grep -c .)

START_HEAD=$(git -C "$ROOT" rev-parse HEAD)
START_BRANCH=$(git -C "$ROOT" symbolic-ref -q HEAD || echo "(detached)")

# --- Temporary directories --------------------------------------------------------------

if [ -n "${GATE_LOG_DIR:-}" ]; then
  # Refused inside the repository (it would dirty the checkout), decided
  # before anything is created: the nearest existing directory, resolved.
  want=$GATE_LOG_DIR
  case $want in /*) ;; *) want="$PWD/$want" ;; esac
  probe=$want rest=""
  while [ ! -d "$probe" ]; do rest="/$(basename "$probe")$rest"; probe=$(dirname "$probe"); done
  case "$(cd "$probe" && pwd -P)$rest/" in "$(cd "$ROOT" && pwd -P)/"*) die_pre "GATE_LOG_DIR must be outside the repository ($want)" ;; esac
  mkdir -p "$want" || die_pre "cannot create $want"
  LOG_DIR=$(cd "$want" && pwd -P)
else
  LOG_DIR=$(mktemp -d "${TMPDIR:-/tmp}/istary-gate-log.XXXXXX") || die_pre "cannot create a log directory"
  LOG_DIR=$(cd "$LOG_DIR" && pwd -P)
fi
WT_DIR=$(mktemp -d "${TMPDIR:-/tmp}/istary-gate-wt.XXXXXX") || die_pre "cannot create a worktree directory"
WT_DIR=$(cd "$WT_DIR" && pwd -P)
CLEANUP="pending"

say "== Istary regression gate"
say "  repository: $ROOT ($START_BRANCH at $START_HEAD)"
say "  base:       $BASE = $BASE_SHA"
say "  candidate:  $CANDIDATE = $CAND_SHA ($COMMITS_TOTAL commits)"
say "  platform:   $os/$arch; $(go version | cut -d' ' -f3); node $(node --version); wails $("$WAILS" version 2>/dev/null | head -1)"
say "  timeouts:   ${STAGE_TIMEOUT}s a stage, ${RACE_TIMEOUT}s root race"
say "  logs:       $LOG_DIR"

# add_worktree SHA: a detached worktree of SHA under WT_DIR.
add_worktree() {
  local sha=$1 wt="$WT_DIR/wt-$1" out
  # Recorded first, so cleanup removes it even if adding fails halfway.
  WORKTREES+=("$wt")
  if ! out=$(git -C "$ROOT" worktree add --detach "$wt" "$sha" 2>&1); then
    say "   $out"
    return 1
  fi
}
drop_worktree() {
  if ! remove_worktree "$1"; then
    CLEANUP="FAILED"
    return 1
  fi
}

# --- Each commit alone ----------------------------------------------------------------------

i=0
for c in $COMMITS; do
  i=$((i + 1))
  wt="$WT_DIR/wt-$c"
  say ""
  say "== commit $i/$COMMITS_TOTAL: $(git -C "$ROOT" log --oneline -1 "$c" | cut -c1-72)"
  check "commit $i: worktree" add_worktree "$c"
  stage "commit $i: root vet" "$wt" "$STAGE_TIMEOUT" go vet ./...
  stage "commit $i: root test" "$wt" "$STAGE_TIMEOUT" go test ./engine ./cmd/istary ./app -count=1
  stage "commit $i: gui vet" "$wt/gui" "$STAGE_TIMEOUT" go vet ./...
  stage "commit $i: gui test" "$wt/gui" "$STAGE_TIMEOUT" go test ./... -count=1
  stage "commit $i: frontend test" "$wt/gui/frontend" "$STAGE_TIMEOUT" npm test
  check "commit $i: remove worktree" drop_worktree "$wt"
  COMMITS_TESTED=$i
done

# --- The candidate in full -------------------------------------------------------------------

wt="$WT_DIR/wt-$CAND_SHA"
say ""
say "== candidate $CAND_SHA in full"
check "tip: worktree" add_worktree "$CAND_SHA"
stage "tip: root vet" "$wt" "$STAGE_TIMEOUT" go vet ./...
stage "tip: root test" "$wt" "$STAGE_TIMEOUT" go test ./... -count=1
stage "tip: root shuffle" "$wt" "$STAGE_TIMEOUT" go test ./... -count=1 -shuffle=on
stage "tip: make build" "$wt" "$STAGE_TIMEOUT" make build
# Under the race detector the app package alone takes over an hour (about
# 65 minutes at v0.5.0-automap).
stage "tip: root race" "$wt" $((RACE_TIMEOUT + MARGIN)) go test -race ./... -count=1 -timeout "${RACE_TIMEOUT}s"
stage "tip: gui vet" "$wt/gui" "$STAGE_TIMEOUT" go vet ./...
stage "tip: gui test" "$wt/gui" "$STAGE_TIMEOUT" go test ./... -count=1
stage "tip: gui shuffle" "$wt/gui" "$STAGE_TIMEOUT" go test ./... -count=1 -shuffle=on
stage "tip: gui race" "$wt/gui" $((STAGE_TIMEOUT + MARGIN)) go test -race ./... -count=1 -timeout "${STAGE_TIMEOUT}s"
stage "tip: frontend dependencies" "$wt/gui/frontend" "$STAGE_TIMEOUT" npm ci --prefer-offline --no-audit --no-fund
stage "tip: frontend test" "$wt/gui/frontend" "$STAGE_TIMEOUT" npm test

# Regenerating every fixture and golden must change nothing.
tree_state() { # the worktree's changes, tracked and untracked
  { git -C "$wt" status --porcelain --untracked-files=all && git -C "$wt" diff; } | shasum
}
fixtures_before() { FIX_BEFORE=$(tree_state) || { say "   cannot read the worktree's state"; return 1; }; }
check "tip: fixtures state before" fixtures_before
stage "tip: fixtures regenerate (gui)" "$wt/gui" "$STAGE_TIMEOUT" env ISTARY_UPDATE_FIXTURES=1 go test . -count=1
stage "tip: fixtures regenerate (goldens)" "$wt" "$STAGE_TIMEOUT" go test ./cmd/istary -count=1 -update
fixtures_unchanged() {
  local after
  after=$(tree_state) || { say "   cannot read the worktree's state"; return 1; }
  [ "$after" = "$FIX_BEFORE" ] && return 0
  say "   regenerating changed fixtures:"
  git -C "$wt" status --porcelain --untracked-files=all | sed 's/^/   | /'
  return 1
}
check "tip: fixtures unchanged" fixtures_unchanged

stage "tip: wails build" "$wt/gui" "$STAGE_TIMEOUT" "$WAILS" build -platform darwin/arm64
stage "tip: codesign verify" "$wt/gui" "$STAGE_TIMEOUT" codesign --verify --verbose build/bin/Istary.app
check "tip: remove worktree" drop_worktree "$wt"
TIP_DONE="all"

# --- The repository afterwards ----------------------------------------------------------------

repository_unchanged() {
  local ok=0 now_branch left
  [ "$(git -C "$ROOT" rev-parse HEAD)" = "$START_HEAD" ] || { say "   HEAD moved from $START_HEAD"; ok=1; }
  now_branch=$(git -C "$ROOT" symbolic-ref -q HEAD || echo "(detached)")
  [ "$now_branch" = "$START_BRANCH" ] || { say "   the branch changed from $START_BRANCH to $now_branch"; ok=1; }
  left=$(git -C "$ROOT" status --porcelain --untracked-files=normal)
  [ -z "$left" ] || { say "   the working tree is no longer clean:"; printf '%s\n' "$left" | sed 's/^/   | /'; ok=1; }
  local list
  list=$(git -C "$ROOT" worktree list --porcelain) || { say "   git worktree list failed"; ok=1; }
  if grep -qF "worktree $WT_DIR/" <<<"$list"; then say "   a gate worktree is still registered"; ok=1; fi
  return $ok
}
check "final: repository unchanged, no gate worktrees" repository_unchanged
exit 0 # on_exit removes the worktree directory, reports and sets the status
