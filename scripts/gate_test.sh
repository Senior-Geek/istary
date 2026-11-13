#!/usr/bin/env bash
# Tests of scripts/gate.sh's own control flow (G-1), in seconds and without
# touching the real repository: each test makes a small throwaway repository
# that looks like Istary (go.mod, gui/wails.json), puts stub go, npm, node,
# make, wails, codesign, cc and uname first on PATH, wraps git to inject
# failures, runs the real gate against it, and checks the gate's exit
# status, what it reported, that its worktrees and temporary directories
# are gone, that no process it started is left, and that the repository is
# as it was.
#
#   scripts/gate_test.sh            all tests
#   scripts/gate_test.sh NAME...    only these (e.g. timeout interrupt-int)

set -u -o pipefail

HERE=$(cd "$(dirname "$0")" && pwd -P)
GATE="$HERE/gate.sh"
REAL_GIT=$(command -v git) || { echo "git is needed"; exit 2; }
SCRATCH=$(mktemp -d "${TMPDIR:-/tmp}/istary-gate-test.XXXXXX") || exit 2
SCRATCH=$(cd "$SCRATCH" && pwd -P)
trap 'rm -rf "$SCRATCH"' EXIT
export REAL_GIT

# --- Stubs --------------------------------------------------------------------------

STUBS="$SCRATCH/stubs"
mkdir -p "$STUBS"
cat >"$STUBS/stub" <<'EOF'
#!/bin/bash
# A stand-in for a tool the gate runs. It records each call (directory and
# command) and does what the test asks through the environment:
#   STUB_FAIL     a pattern: a matching call prints STUB_FAIL_OUTPUT and exits STUB_FAIL_CODE
#   STUB_SLEEP    a pattern: a matching call starts a child and waits (for timeouts and interrupts)
#   STUB_FIXTURE_CHANGE=1  regenerating fixtures changes a tracked file
name=$(basename "$0")
call="$name $*"
echo "$PWD|$call" >>"$STUB_LOG"
case $name in
  uname) case ${1:-} in -s) echo "${STUB_OS:-Darwin}" ;; -m) echo "${STUB_ARCH:-arm64}" ;; *) echo "${STUB_OS:-Darwin}" ;; esac; exit 0 ;;
  node) [ "${1:-}" = --version ] && { echo "${STUB_NODE:-v22.0.0}"; exit 0; } ;;
  npm) [ "${1:-}" = --version ] && { echo 10.0.0; exit 0; } ;;
  cc) exit 0 ;;
  go) case ${1:-} in version) echo "go version go1.99 darwin/arm64"; exit 0 ;; env) echo /nonexistent; exit 0 ;; esac ;;
  wails) [ "${1:-}" = version ] && { echo v2.0.0-stub; exit 0; } ;;
esac
if [ -n "${STUB_FAIL:-}" ] && [[ "$call" == *"$STUB_FAIL"* ]]; then
  printf '%b\n' "${STUB_FAIL_OUTPUT:-stub: injected failure}"
  exit "${STUB_FAIL_CODE:-1}"
fi
if [ -n "${STUB_SLEEP:-}" ] && [[ "$call" == *"$STUB_SLEEP"* ]]; then
  sleep 300 &
  echo "$$ $!" >"$STUB_PIDS"
  wait
  exit 0
fi
case $call in
  "npm test") printf '# pass 3\n# fail 0\n' ;;
  "wails build"*) mkdir -p build/bin/Istary.app ;;
  "go test . -count=1"|"go test ./cmd/istary -count=1 -update")
    if [ -n "${STUB_FIXTURE_CHANGE:-}" ] && [ -n "${ISTARY_UPDATE_FIXTURES:-}${call##*-count=1}" ]; then
      echo changed >>"$(git rev-parse --show-toplevel)/fixture.json"
    fi ;;
esac
echo "stub: $call ok"
exit 0
EOF
chmod +x "$STUBS/stub"
for t in go npm node make wails codesign cc uname; do ln -s stub "$STUBS/$t"; done
cat >"$STUBS/git" <<'EOF'
#!/bin/bash
# git, except that a call matching GIT_FAIL fails.
if [ -n "${GIT_FAIL:-}" ] && [[ "$*" == *"$GIT_FAIL"* ]]; then
  echo "git: injected failure: $*" >&2
  exit 1
fi
exec "$REAL_GIT" "$@"
EOF
chmod +x "$STUBS/git"
# The system tools the gate and the stubs use, and nothing else: so a stub
# removed from PATH is really missing.
SYS="$SCRATCH/sys"
mkdir -p "$SYS"
for t in bash env sed grep tr tail head cat date mktemp rm rmdir mkdir ls basename dirname sleep perl shasum cut sort wc touch cp; do
  p=$(PATH=/usr/bin:/bin command -v "$t") || { echo "$t is needed"; exit 2; }
  ln -s "$p" "$SYS/$t"
done

# --- A repository like Istary ------------------------------------------------------------

# make_repo DIR: base (tagged base), then two commits on branch feature,
# checked out; also a commit "other" off base that feature does not contain.
make_repo() {
  local r=$1
  mkdir -p "$r/gui/frontend"
  (
    cd "$r" || exit 1
    g() { "$REAL_GIT" "$@" >/dev/null 2>&1 || { echo "setup: git $* failed"; exit 1; }; }
    g init -q -b main
    g config user.email gate@test
    g config user.name gate-test
    printf 'module github.com/Senior-Geek/istary\n\ngo 1.24.0\n' >go.mod
    echo '{}' >gui/wails.json
    echo '{}' >gui/frontend/package.json
    echo '{}' >fixture.json
    printf 'build/\nnode_modules/\n' >.gitignore
    g add -A; g commit -q -m base; g tag base
    g checkout -q -b other; echo other >other.txt; g add -A; g commit -q -m other
    g checkout -q main; g checkout -q -b feature
    echo one >one.txt; g add -A; g commit -q -m "feature one"
    echo two >two.txt; g add -A; g commit -q -m "feature two"
  ) || return 1
}

# --- Running the gate --------------------------------------------------------------------------

FAILS=0 PASSES=0
CURRENT=""
fail() { echo "  FAIL: $*"; CASE_OK=""; }
pass_case() { if [ -n "$CASE_OK" ]; then echo "ok   $CURRENT"; PASSES=$((PASSES + 1)); else echo "FAIL $CURRENT"; FAILS=$((FAILS + 1)); echo "  --- gate output:"; sed 's/^/  | /' "$OUT" | tail -40; fi; }

# setup NAME: a fresh repository and environment for one test.
setup() {
  CURRENT=$1 CASE_OK=1
  T="$SCRATCH/$1"
  REPO="$T/repo" OUT="$T/out" LOGS="$T/logs" TMP="$T/tmp"
  mkdir -p "$T" "$TMP"
  make_repo "$REPO" || { fail "could not make the repository"; return 1; }
  START_HEAD=$("$REAL_GIT" -C "$REPO" rev-parse HEAD)
  : >"$T/calls"
  STUB_PATH="$STUBS"
}

# gate ARGS...: run the gate in REPO with the stubs; sets RC. Extra
# environment comes from ENV (an array of NAME=VALUE). The tests run it with
# --per-commit, its exhaustive mode, unless MODE is "tip" (the default mode).
ENV=()
MODE=per-commit
gate() {
  local mode=()
  [ "$MODE" = tip ] || mode=(--per-commit)
  (cd "$REPO" && env -i HOME="$HOME" PATH="$STUB_PATH:$SYS" TMPDIR="$TMP" \
    REAL_GIT="$REAL_GIT" STUB_LOG="$T/calls" STUB_PIDS="$T/pids" WAILS="$STUB_PATH/wails" \
    GATE_LOG_DIR="$LOGS" ${ENV[@]+"${ENV[@]}"} bash "$GATE" ${mode[@]+"${mode[@]}"} "$@") >"$OUT" 2>&1
  RC=$?
}

expect_rc() { [ "$RC" -eq "$1" ] || fail "exit status $RC, want $1"; }
expect_out() { grep -qF -- "$1" "$OUT" || fail "output lacks: $1"; }
expect_no_out() { ! grep -qF -- "$1" "$OUT" || fail "output has: $1"; }
called() { grep -qF -- "$1" "$T/calls" || fail "not run: $1"; }
not_called() { ! grep -qF -- "$1" "$T/calls" || fail "run but should not be: $1"; }

# expect_clean: only the repository's own worktree is registered, the
# gate's temporary worktree directory is gone, the checkout is unchanged.
expect_clean() {
  local n
  n=$("$REAL_GIT" -C "$REPO" worktree list --porcelain | grep -c '^worktree ')
  [ "$n" -eq 1 ] || fail "$n worktrees registered: $("$REAL_GIT" -C "$REPO" worktree list)"
  ! ls -d "$TMP"/istary-gate-wt.* >/dev/null 2>&1 || fail "a gate worktree directory is left in $TMP"
  [ "$("$REAL_GIT" -C "$REPO" rev-parse HEAD)" = "$START_HEAD" ] || fail "HEAD moved"
  [ -z "$("$REAL_GIT" -C "$REPO" status --porcelain)" ] || fail "the checkout changed: $("$REAL_GIT" -C "$REPO" status --porcelain)"
}
# expect_not_started: the gate stopped before making anything.
expect_not_started() {
  [ ! -s "$T/calls" ] || ! grep -qE '\|(go (vet|test)|npm (test|ci)|make|wails build)' "$T/calls" || fail "a stage ran: $(head -3 "$T/calls")"
  expect_no_out "== RESULT"
  expect_clean
}
# expect_dead: no process the sleeping stub started is still running.
expect_dead() {
  [ -s "$T/pids" ] || { fail "the stub never started"; return; }
  local p
  for p in $(cat "$T/pids"); do
    if kill -0 "$p" 2>/dev/null; then fail "process $p of the stopped stage is still running"; kill -9 "$p" 2>/dev/null; fi
  done
}
# wait_started: until the sleeping stub is running (at most 30 s).
wait_started() {
  local i=0
  while [ ! -s "$T/pids" ] && [ $i -lt 150 ]; do sleep 0.2; i=$((i + 1)); done
  [ -s "$T/pids" ]
}

wanted() { [ ${#ONLY[@]} -eq 0 ] && return 0; local o; for o in "${ONLY[@]}"; do [ "$o" = "$1" ] && return 0; done; return 1; }
ONLY=("$@")

# --- The tests ------------------------------------------------------------------------------------

t_pass() {
  setup pass || return
  ENV=(); gate base feature
  expect_rc 0
  expect_out "== RESULT: PASS"
  expect_out "commits:    2 of 2 tested alone"
  expect_out "tip checks: all"
  expect_out "worktrees:  cleanup ok"
  expect_out "failed:     none"
  # Every stage ran, in the worktree of its own commit.
  local c tip
  tip=$("$REAL_GIT" -C "$REPO" rev-parse feature)
  for c in $("$REAL_GIT" -C "$REPO" rev-list base..feature); do
    called "wt-$c|go vet ./..."
    called "wt-$c|go test ./engine ./cmd/istary ./app -count=1"
    called "wt-$c/gui|go vet ./..."
    called "wt-$c/gui|go test ./... -count=1"
    called "wt-$c/gui/frontend|npm test"
  done
  for s in "|go test ./... -count=1" "|go test ./... -count=1 -shuffle=on" "|make build" "|go test -race ./... -count=1 -timeout 9000s" \
    "/gui|go test -race ./... -count=1 -timeout 3600s" "/gui/frontend|npm ci --prefer-offline" "/gui|go test . -count=1" \
    "|go test ./cmd/istary -count=1 -update" "/gui|wails build -platform darwin/arm64" "/gui|codesign --verify --verbose build/bin/Istary.app"; do
    called "wt-$tip$s"
  done
  # Nothing ran in the checkout itself.
  local here
  here=$(grep "^$REPO|" "$T/calls")
  ! grep -qE '\|(go (vet|test)|npm (test|ci)|make |wails build|codesign)' <<<"$here" ||
    fail "a stage ran in the checkout: $(head -2 <<<"$here")"
  [ -s "$LOGS/gate.log" ] || fail "no gate.log"
  grep -qF "== RESULT: PASS" "$LOGS/gate.log" || fail "gate.log lacks the result"
  expect_clean
  pass_case
}

# 1. A failing command whose output a later command in a pipeline would
#    have accepted: the old gate's `npm test | grep "# pass|fail"`.
t_pipeline() {
  setup pipeline || return
  ENV=(STUB_FAIL="npm test" STUB_FAIL_OUTPUT='# pass 3\n# fail 0')
  gate base feature
  expect_rc 1
  expect_out "== RESULT: FAIL"
  expect_out "failed:     [06] commit 1: frontend test: npm test (FAIL)"
  expect_clean
  pass_case
  # And a failing command inside the gate's own pipeline (git diff | shasum).
  setup pipeline-internal || return
  ENV=(GIT_FAIL=" diff")
  gate base feature
  expect_rc 1
  expect_out "tip: fixtures state before (FAIL)"
  not_called "-update"
  expect_clean
  pass_case
}

# 2. A failed build followed by commands that succeed.
t_build() {
  setup build-make || return
  ENV=(STUB_FAIL="make build" STUB_FAIL_CODE=2)
  gate base feature
  expect_rc 1
  expect_out "tip: make build: make build (FAIL)"
  not_called "go test -race"
  expect_clean
  pass_case
  setup build-wails || return
  ENV=(STUB_FAIL="wails build")
  gate base feature
  expect_rc 1
  expect_out "tip: wails build"
  not_called "codesign --verify"
  expect_out "== RESULT: FAIL"
  expect_clean
  pass_case
}

# 3. A required dependency is missing or unusable; the wrong platform.
t_dependency() {
  local tool
  for tool in npm go make codesign; do
    setup "missing-$tool" || return
    STUB_PATH="$T/stubs"; mkdir -p "$STUB_PATH"; cp -P "$STUBS"/* "$STUB_PATH"/; rm "$STUB_PATH/$tool"
    ENV=(); gate base feature
    expect_rc 2
    expect_out "GATE CANNOT START: $tool"
    expect_not_started
    pass_case
  done
  setup missing-wails || return
  ENV=(); STUB_PATH="$STUBS"
  (cd "$REPO" && env -i HOME="$HOME" PATH="$STUBS:$SYS" TMPDIR="$TMP" REAL_GIT="$REAL_GIT" STUB_LOG="$T/calls" \
    WAILS=/nonexistent/wails GATE_LOG_DIR="$LOGS" bash "$GATE" base feature) >"$OUT" 2>&1; RC=$?
  expect_rc 2
  expect_out "the Wails CLI was not found"
  expect_not_started
  pass_case
  setup old-node || return
  ENV=(STUB_NODE=v18.0.0); gate base feature
  expect_rc 2
  expect_out "Node.js 20 or later"
  expect_not_started
  pass_case
  setup platform || return
  ENV=(STUB_OS=Linux STUB_ARCH=x86_64); gate base feature
  expect_rc 2
  expect_out "runs on macOS on Apple silicon"
  expect_out "this is Linux/x86_64"
  expect_not_started
  pass_case
}

# 4. A dirty working tree: nothing is stashed, reset or cleaned.
t_dirty() {
  setup dirty-tracked || return
  echo changed >"$REPO/one.txt"
  ENV=(); gate base feature
  expect_rc 2
  expect_out "is not clean"
  expect_out " M one.txt"
  [ "$(cat "$REPO/one.txt")" = changed ] || fail "the change was touched"
  "$REAL_GIT" -C "$REPO" checkout -q -- one.txt
  expect_not_started
  pass_case
  setup dirty-untracked || return
  echo new >"$REPO/new.txt"
  ENV=(); gate base feature
  expect_rc 2
  expect_out "?? new.txt"
  [ -f "$REPO/new.txt" ] || fail "the untracked file was removed"
  rm "$REPO/new.txt"
  expect_not_started
  pass_case
}

# 5, 6. Invalid, option-like and ambiguous refs; not Istary.
t_refs() {
  setup bad-base || return
  ENV=(); gate no-such-ref feature
  expect_rc 2; expect_out "the base 'no-such-ref' does not name a commit"; expect_not_started; pass_case
  setup bad-candidate || return
  gate base no-such-ref
  expect_rc 2; expect_out "the candidate 'no-such-ref' does not name a commit"; expect_not_started; pass_case
  setup option-ref || return
  gate base --output=/tmp/x
  expect_rc 2; expect_out "is not a valid ref"; expect_not_started; pass_case
  setup ambiguous-ref || return
  "$REAL_GIT" -C "$REPO" tag feature other
  gate base feature
  expect_rc 2; expect_out "is ambiguous"; expect_not_started; pass_case
  setup usage || return
  gate
  expect_rc 2; expect_out "usage:"; expect_not_started; pass_case
  setup not-istary || return
  printf 'module example.com/other\n' >"$REPO/go.mod"; "$REAL_GIT" -C "$REPO" commit -qam other-module
  START_HEAD=$("$REAL_GIT" -C "$REPO" rev-parse HEAD)
  gate base feature
  expect_rc 2; expect_out "is not the Istary repository"; expect_not_started; pass_case
}

# 7. The candidate does not descend from the base, or is the base.
t_ancestry() {
  setup not-ancestor || return
  ENV=(); gate other feature
  expect_rc 2; expect_out "is not an ancestor of the candidate"; expect_not_started; pass_case
  setup reversed || return
  gate feature base
  expect_rc 2; expect_out "is not an ancestor"; expect_not_started; pass_case
  setup same-commit || return
  gate feature feature
  expect_rc 2; expect_out "nothing to gate"; expect_not_started; pass_case
}

# 8. Adding a worktree fails.
t_worktree() {
  setup worktree-add || return
  ENV=(GIT_FAIL="worktree add"); gate base feature
  expect_rc 1
  expect_out "commit 1: worktree (FAIL)"
  expect_out "worktrees:  cleanup ok"
  not_called "go vet"
  expect_clean
  pass_case
}

# 9. A test stage fails: later stages do not run.
t_stage() {
  setup stage-fail || return
  ENV=(STUB_FAIL="go test ./... -count=1 -shuffle=on" STUB_FAIL_OUTPUT="--- FAIL: TestSomething")
  gate base feature
  expect_rc 1
  expect_out "tip: root shuffle: go test ./... -count=1 -shuffle=on (FAIL)"
  expect_out "--- FAIL: TestSomething"
  not_called "make build"
  not_called "go test -race"
  expect_clean
  pass_case
}

# 10. A stage runs past its limit; Go's own -timeout is a timeout too.
t_timeout() {
  setup timeout || return
  ENV=(STUB_SLEEP="go test -race ./... -count=1 -timeout" GATE_STAGE_TIMEOUT=2 GATE_ROOT_RACE_TIMEOUT=2)
  local t0=$SECONDS
  gate base feature
  expect_rc 124
  expect_out "TIMEOUT in"
  expect_out "tip: root race"
  expect_out "(TIMEOUT)"
  [ $((SECONDS - t0)) -lt 60 ] || fail "took $((SECONDS - t0)) s"
  # The GUI vet ran for the two commits alone, not for the tip.
  [ "$(grep -c '/gui|go vet' "$T/calls")" -eq 2 ] || fail "a later stage ran: $(grep -c '/gui|go vet' "$T/calls") GUI vets"
  expect_dead
  expect_clean
  pass_case
  setup timeout-go || return
  ENV=(STUB_FAIL="go test -race ./... -count=1" STUB_FAIL_OUTPUT="panic: test timed out after 2h30m0s")
  gate base feature
  expect_rc 124
  expect_out "tip: root race: go test -race ./... -count=1 -timeout 9000s (TIMEOUT)"
  expect_clean
  pass_case
}

# 11. An interrupted run stops its stage and removes its worktrees.
interrupt() {
  local sig=$1 want=$2
  setup "interrupt-$(echo "$sig" | tr 'A-Z' 'a-z')" || return
  ENV=(STUB_SLEEP="go test ./engine")
  # A background job ignores SIGINT; perl restores the default before the gate starts.
  (cd "$REPO" && exec perl -e '$SIG{INT} = "DEFAULT"; exec @ARGV' env -i HOME="$HOME" PATH="$STUB_PATH:$SYS" \
    TMPDIR="$TMP" REAL_GIT="$REAL_GIT" STUB_LOG="$T/calls" STUB_PIDS="$T/pids" WAILS="$STUB_PATH/wails" GATE_LOG_DIR="$LOGS" \
    "${ENV[@]}" bash "$GATE" --per-commit base feature) >"$OUT" 2>&1 &
  local pid=$!
  if ! wait_started; then fail "the stage never started"; kill -9 "$pid"; wait "$pid"; pass_case; return; fi
  kill -"$sig" "$pid"
  wait "$pid"; RC=$?
  expect_rc "$want"
  expect_out "interrupted (SIG$sig)"
  expect_out "== RESULT: INTERRUPTED"
  expect_out "worktrees:  cleanup ok"
  expect_dead
  expect_clean
  pass_case
}
t_interrupt() { interrupt INT 130; interrupt TERM 143; }

# 12. Removing a worktree fails: reported, and the gate does not pass.
t_cleanup() {
  setup cleanup-fail || return
  ENV=(GIT_FAIL="worktree remove"); gate base feature
  expect_rc 4
  expect_out "could not remove worktree"
  expect_out "worktrees:  cleanup FAILED"
  expect_out "== RESULT: FAIL"
  expect_no_out "RESULT: PASS"
  not_called "go test -race" # it stops at the first commit's removal
  # Tidy the test repository by hand.
  local wt
  for wt in $("$REAL_GIT" -C "$REPO" worktree list --porcelain | sed -n 's/^worktree //p' | grep -v "^$REPO\$"); do
    "$REAL_GIT" -C "$REPO" worktree remove --force "$wt"
  done
  rm -rf "$TMP"/istary-gate-wt.*
  pass_case
}

# Regenerating fixtures changes a file.
t_fixtures() {
  setup fixtures-changed || return
  ENV=(STUB_FIXTURE_CHANGE=1); gate base feature
  expect_rc 1
  expect_out "regenerating changed fixtures"
  expect_out "fixture.json"
  expect_out "tip: fixtures unchanged (FAIL)"
  not_called "wails build"
  expect_clean
  pass_case
}

# The log directory must not be inside the repository.
t_logdir() {
  setup logdir-inside || return
  LOGS="$REPO/logs"
  ENV=(); gate base feature
  expect_rc 2
  expect_out "GATE_LOG_DIR must be outside the repository"
  [ ! -e "$REPO/logs" ] || fail "the refused log directory was created"
  expect_not_started
  pass_case
}

# The default mode: the candidate in full, no commit alone; the same
# result, cleanup and checks of the checkout.
t_tip() {
  setup tip || return
  MODE=tip; ENV=(); gate base feature; MODE=per-commit
  expect_rc 0
  expect_out "== RESULT: PASS"
  expect_out "mode:       the candidate in full"
  expect_out "commits:    the candidate only, 2 commits after the base"
  expect_out "tip checks: all"
  expect_no_out "== commit 1/"
  local c tip
  tip=$("$REAL_GIT" -C "$REPO" rev-parse feature)
  for c in $("$REAL_GIT" -C "$REPO" rev-list base..feature^); do not_called "wt-$c|"; done
  for s in "|go test ./... -count=1" "|go test ./... -count=1 -shuffle=on" "|make build" "|go test -race ./... -count=1 -timeout 9000s" \
    "/gui|go test -race ./... -count=1 -timeout 3600s" "/gui|go test . -count=1" "|go test ./cmd/istary -count=1 -update" \
    "/gui|wails build -platform darwin/arm64" "/gui|codesign --verify --verbose build/bin/Istary.app"; do
    called "wt-$tip$s"
  done
  expect_clean
  pass_case
  # A failing stage fails it the same way.
  setup tip-fail || return
  MODE=tip; ENV=(STUB_FAIL="go test -race ./... -count=1 -timeout 9000s"); gate base feature; MODE=per-commit
  expect_rc 1
  expect_out "tip: root race"
  expect_out "== RESULT: FAIL"
  not_called "wails build"
  expect_clean
  pass_case
  # And the usage names the option.
  setup tip-usage || return
  MODE=tip; ENV=(); gate; MODE=per-commit
  expect_rc 2
  expect_out "usage: scripts/gate.sh [--per-commit] BASE [CANDIDATE]"
  pass_case
}

for t in pass pipeline build dependency dirty refs ancestry worktree stage timeout interrupt cleanup fixtures logdir tip; do
  wanted "$t" && "t_$t"
done
echo
echo "$PASSES passed, $FAILS failed"
[ "$FAILS" -eq 0 ]
