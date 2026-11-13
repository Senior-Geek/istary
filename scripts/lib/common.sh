# Shared by scripts/test-fast.sh and scripts/test-acceptance.sh (docs/TESTING.md).
# Sourced, not run. It gives each level:
#   - stages: a command run in a directory as its own process group, with a
#     time limit, its output in its own log, timed; the level stops at the
#     first stage that does not pass;
#   - isolation: every stage runs with HOME, TMPDIR and ISTARY_SAVE_DIR in the
#     level's own temporary directory (Go's and npm's caches are kept), so no
#     test can reach a real save folder;
#   - the real saves: a checksum of every file in the GUI's and the terminal
#     game's save folders before and after, which must agree;
#   - a timing summary and one result line.
# It only ever stops processes it started (their process groups).

set -u -o pipefail

TIERS_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)
LEVEL=""            # the level's name, for the report
STAGES=0
FAILED_STAGE=""
STATUS=0
INTERRUPTED=""
CHILD=""
WATCHDOG=""
WORK=""             # the level's temporary directory
LOG_DIR=""
SUMMARY=()          # "result|seconds|name"
NOTES=()            # lines for the report
CLEANUPS=()         # commands run at exit, last first
T_START=$(date +%s)
REAL_HOME=$HOME

# Measured test times, to balance the parallel Go tests: kept outside the
# repository, starting from the committed baseline.
if [ -n "${XDG_CACHE_HOME:-}" ]; then CACHE_BASE=$XDG_CACHE_HOME; elif [ -d "$REAL_HOME/Library/Caches" ]; then CACHE_BASE=$REAL_HOME/Library/Caches; else CACHE_BASE=$REAL_HOME/.cache; fi
TIMINGS_CACHE="$CACHE_BASE/istary-tests/timings.json"
TIMINGS_BASELINE="$TIERS_ROOT/scripts/lib/test-timings.json"

# The real save folders: the GUI's (games and the earlier single save) and the
# terminal game's. Read only; never written by these scripts.
REAL_SAVES=("$REAL_HOME/Library/Application Support/Istary" "$REAL_HOME/.config/wizardry")

say() { printf '%s\n' "$*"; [ -n "$LOG_DIR" ] && [ -d "$LOG_DIR" ] && printf '%s\n' "$*" >>"$LOG_DIR/$LEVEL.log"; return 0; }
dur() { local s=$1; printf '%dm%02ds' $((s / 60)) $((s % 60)); }
note() { NOTES+=("$*"); }
die_pre() { say "$LEVEL CANNOT START: $*" >&2; STATUS=2; exit 2; }
need() { command -v "$1" >/dev/null 2>&1 || die_pre "$1 is not installed or not on PATH ($2)"; }

tiers_init() {
  LEVEL=$1
  need git "the changed files"
  need go "the Go tests and builds"
  need node "the frontend tests and the test runner"
  need npm "the frontend"
  need perl "running each stage as its own process group with a time limit"
  need shasum "checking the real saves"
  WORK=$(mktemp -d "${TMPDIR:-/tmp}/istary-$LEVEL.XXXXXX") || die_pre "cannot create a temporary directory"
  WORK=$(cd "$WORK" && pwd -P)
  LOG_DIR="$WORK/logs"
  mkdir -p "$LOG_DIR" "$WORK/home" "$WORK/tmp" "$WORK/no-saves" || die_pre "cannot create $WORK"
  trap on_exit EXIT
  trap 'on_signal INT' INT
  trap 'on_signal TERM' TERM
  trap 'on_signal HUP' HUP
  TEST_TMP="$WORK/tmp"
  ramdisk_up
  # The caches the tools would find through HOME, named so a stage with its
  # own HOME still finds them.
  ISO_ENV=(
    "HOME=$WORK/home" "TMPDIR=$TEST_TMP" "ISTARY_SAVE_DIR=$WORK/no-saves"
    "GOCACHE=$(go env GOCACHE)" "GOMODCACHE=$(go env GOMODCACHE)" "GOPATH=$(go env GOPATH)" "GOENV=$(go env GOENV)"
    "npm_config_cache=$(npm config get cache 2>/dev/null || echo "$REAL_HOME/.npm")"
  )
}

# ramdisk_up puts the tests' temporary files on a RAM disk of the run's own
# (macOS), mounted inside WORK, not under /Volumes. Every save the tests make
# is an atomic write with F_FULLFSYNC, which on the real disk is a drive-wide
# flush that parallel tests queue behind: a save-heavy test takes 58 s on the
# disk and 23 s here. Nothing about the saves changes but where they land;
# the full gate still runs on the real disk. TIERS_RAMDISK=0 turns it off; if
# it cannot be made, the run uses the disk and says so.
RAM_DEV=""
ramdisk_up() {
  [ "${TIERS_RAMDISK:-1}" = 0 ] && { note "temporary files on the disk (TIERS_RAMDISK=0)"; return 0; }
  [ "$(uname -s)" = Darwin ] && command -v hdiutil >/dev/null && command -v newfs_hfs >/dev/null || return 0
  local mp="$WORK/ram" mb=${TIERS_RAMDISK_MB:-1024}
  RAM_DEV=$(hdiutil attach -nomount "ram://$((mb * 2048))" 2>/dev/null | awk 'NR == 1 {print $1}')
  if [ -z "$RAM_DEV" ]; then note "no RAM disk could be made: temporary files on the disk (slower)"; return 0; fi
  CLEANUPS+=('ramdisk_down')
  mkdir -p "$mp" &&
    newfs_hfs -v "IstaryTests" "$RAM_DEV" >/dev/null 2>&1 &&
    mount -t hfs -o nobrowse "$RAM_DEV" "$mp" 2>/dev/null &&
    mkdir -p "$mp/tmp" || { note "the RAM disk could not be mounted: temporary files on the disk (slower)"; return 0; }
  TEST_TMP="$mp/tmp"
  say "  temporary files: a ${mb} MB RAM disk ($RAM_DEV) at $mp"
}
ramdisk_down() {
  [ -n "$RAM_DEV" ] || return 0
  if mount | grep -qF " on $WORK/ram "; then umount "$WORK/ram" 2>/dev/null || umount -f "$WORK/ram" || return 1; fi
  hdiutil detach "$RAM_DEV" >/dev/null 2>&1 || hdiutil detach -force "$RAM_DEV" >/dev/null 2>&1 || { say "  could not detach the RAM disk $RAM_DEV: hdiutil detach $RAM_DEV"; return 1; }
  RAM_DEV=""
}

stop_child() {
  if [ -n "$WATCHDOG" ]; then kill -KILL "$WATCHDOG" 2>/dev/null; wait "$WATCHDOG" 2>/dev/null; WATCHDOG=""; fi
  if [ -n "$CHILD" ]; then
    kill -TERM -- "-$CHILD" 2>/dev/null
    kill -TERM "$CHILD" 2>/dev/null
    local i=0
    while kill -0 -- "-$CHILD" 2>/dev/null && [ $i -lt 50 ]; do sleep 0.2; i=$((i + 1)); done
    kill -KILL -- "-$CHILD" 2>/dev/null
    wait "$CHILD" 2>/dev/null
    CHILD=""
  fi
}

on_exit() {
  local rc=$?
  trap '' INT TERM HUP
  [ "$STATUS" -eq 0 ] && STATUS=$rc
  stop_child
  local i
  for ((i = ${#CLEANUPS[@]} - 1; i >= 0; i--)); do eval "${CLEANUPS[$i]}" || { say "  cleanup failed: ${CLEANUPS[$i]}"; [ "$STATUS" -eq 0 ] && STATUS=4; }; done
  report
  # The temporary directory goes, except the logs when something failed.
  if [ -n "$WORK" ] && [ -d "$WORK" ]; then
    if [ "$STATUS" -eq 0 ]; then rm -rf "$WORK"; else rm -rf "$WORK/home" "$WORK/tmp" "$WORK/no-saves" 2>/dev/null; fi
  fi
  exit "$STATUS"
}
on_signal() {
  INTERRUPTED=$1
  case $1 in INT) STATUS=130 ;; TERM) STATUS=143 ;; HUP) STATUS=129 ;; esac
  say ""
  say "== interrupted (SIG$1): stopping its own running stage"
  exit "$STATUS"
}

report() {
  [ -n "$LEVEL" ] || return 0
  local total=$(( $(date +%s) - T_START )) line r s n
  say ""
  say "== $LEVEL: timing"
  for line in ${SUMMARY[@]+"${SUMMARY[@]}"}; do
    IFS='|' read -r r s n <<<"$line"
    say "$(printf '  %-8s %8s  %s' "$r" "$(dur "$s")" "$n")"
  done
  say "$(printf '  %-8s %8s  %s' total "$(dur "$total")" "")"
  for line in ${NOTES[@]+"${NOTES[@]}"}; do say "  note: $line"; done
  [ -n "$FAILED_STAGE" ] && say "  failed: $FAILED_STAGE"
  [ -n "$INTERRUPTED" ] && say "  interrupted by SIG$INTERRUPTED"
  if [ "$STATUS" -eq 0 ]; then say "  logs: (removed: everything passed)"; else say "  logs: $LOG_DIR"; fi
  local result=PASS
  [ "$STATUS" -ne 0 ] && result=FAIL
  [ -n "$INTERRUPTED" ] && result=INTERRUPTED
  say "== $LEVEL RESULT: $result in $(dur "$total") (exit $STATUS)"
}

# stage NAME DIR TIMEOUT CMD...: CMD in DIR, isolated, as its own process
# group; stopped (that group only) after TIMEOUT seconds.
stage() {
  local name=$1 dir=$2 limit=$3; shift 3
  STAGES=$((STAGES + 1))
  local n log marker start st result
  n=$(printf '%02d' "$STAGES")
  log="$LOG_DIR/$n-$(printf '%s' "$name" | tr -c 'A-Za-z0-9._-' '-').log"
  marker="$LOG_DIR/.timeout-$n"
  say "== [$n] $name"
  start=$(date +%s)
  ( cd "$dir" && exec env "${ISO_ENV[@]}" perl -e 'setpgrp(0, 0) or die "setpgrp: $!\n"; exec { $ARGV[0] } @ARGV or die "cannot run $ARGV[0]: $!\n"' "$@" ) >"$log" 2>&1 &
  CHILD=$!
  perl -e '($t, $g, $m) = @ARGV; sleep $t; open(my $f, ">", $m); close $f; kill "-TERM", $g; sleep 10; kill "-KILL", $g' "$limit" "$CHILD" "$marker" &
  WATCHDOG=$!
  wait "$CHILD"; st=$?
  kill -KILL "$WATCHDOG" 2>/dev/null; wait "$WATCHDOG" 2>/dev/null; WATCHDOG=""
  if [ -e "$marker" ]; then
    kill -KILL -- "-$CHILD" 2>/dev/null
    rm -f "$marker"; result=TIMEOUT
  elif [ "$st" -eq 0 ]; then result=PASS
  else result=FAIL
  fi
  CHILD=""
  local took=$(( $(date +%s) - start ))
  SUMMARY+=("$result|$took|$name")
  say "   $result in $(dur $took) (exit $st)"
  # The runners' own short reports are worth seeing even when they pass.
  grep -E '^(gotest|live): |^  (slowest|FAIL)|^# (pass|fail) ' "$log" | sed 's/^/   | /' | while IFS= read -r l; do say "$l"; done
  if [ "$result" != PASS ]; then
    say "   --- last lines of $log:"
    tail -n 30 "$log" | sed 's/^/   | /' | while IFS= read -r l; do say "$l"; done
    FAILED_STAGE="[$n] $name"
    if [ "$result" = TIMEOUT ]; then STATUS=124; else STATUS=1; fi
    exit "$STATUS"
  fi
}

# check NAME CMD...: a stage the script decides itself.
check() {
  local name=$1; shift
  STAGES=$((STAGES + 1))
  local n start; n=$(printf '%02d' "$STAGES"); start=$(date +%s)
  say "== [$n] $name"
  if "$@"; then
    SUMMARY+=("PASS|$(( $(date +%s) - start ))|$name"); say "   PASS"
  else
    SUMMARY+=("FAIL|$(( $(date +%s) - start ))|$name"); say "   FAIL"
    FAILED_STAGE="[$n] $name"; STATUS=1; exit 1
  fi
}

# saves_fingerprint: every file in the real save folders and its checksum;
# the files are only read.
saves_fingerprint() {
  local d
  for d in "${REAL_SAVES[@]}"; do
    [ -d "$d" ] || { echo "absent $d"; continue; }
    (cd "$d" && find . -type f -print0 | sort -z | xargs -0 shasum -a 256) | sed "s#^#$d #"
  done
}
saves_before() { SAVES_BEFORE=$(saves_fingerprint) || return 1; ISTARY_BEFORE=$(user_istary); return 0; }
saves_unchanged() {
  local after
  after=$(saves_fingerprint) || return 1
  [ "$after" = "$SAVES_BEFORE" ] && return 0
  say "   the real save folders changed during the run:"
  diff <(printf '%s\n' "$SAVES_BEFORE") <(printf '%s\n' "$after") | sed 's/^/   | /' | head -20 | while IFS= read -r l; do say "$l"; done
  [ -n "$ISTARY_BEFORE$(user_istary)" ] && say "   (an Istary you started was running: $ISTARY_BEFORE $(user_istary); it may have saved)"
  return 1
}
# user_istary: Istary applications running that these scripts did not start
# (they are only reported, never stopped).
user_istary() { pgrep -f '/Istary\.app/Contents/MacOS/Istary' 2>/dev/null | tr '\n' ' ' | sed 's/ $//'; }

# changed_files BASE [CANDIDATE]: the files changed since BASE's merge base,
# renames as their old and new paths; without CANDIDATE, the working tree
# (staged, unstaged and untracked) as well.
changed_files() {
  local base=$1 cand=${2:-} mb
  if [ -n "$cand" ]; then
    mb=$(git -C "$TIERS_ROOT" merge-base "$base" "$cand") || return 1
    git -C "$TIERS_ROOT" diff --name-only --no-renames "$mb" "$cand" || return 1
  else
    mb=$(git -C "$TIERS_ROOT" merge-base "$base" HEAD) || return 1
    git -C "$TIERS_ROOT" diff --name-only --no-renames "$mb" || return 1
    git -C "$TIERS_ROOT" ls-files --others --exclude-standard || return 1
  fi
}

# plan_get EXPR: a value from the plan (classify.mjs's JSON in $PLAN_FILE).
plan_get() { node -e 'const p = JSON.parse(require("fs").readFileSync(process.argv[1], "utf8")); const v = (new Function("p", "return " + process.argv[2]))(p); console.log(Array.isArray(v) ? v.join("\n") : v)' "$PLAN_FILE" "$1"; }

print_plan() {
  say "== plan ($(plan_get 'p.files.length') changed files)"
  node -e '
    const p = JSON.parse(require("fs").readFileSync(process.argv[1], "utf8"))
    const kinds = {}
    for (const f of p.files) (kinds[f.kind] ??= []).push(f.path)
    for (const [k, fs] of Object.entries(kinds)) console.log(`  ${k}: ${fs.length > 4 ? fs.slice(0, 4).join(", ") + ` and ${fs.length - 4} more` : fs.join(", ")}`)
    if (p.areas.length) console.log(`  areas: ${p.areas.join(", ")}`)
    if (p.broad) console.log(`  BROADENED to everything: ${p.broadReasons.join("; ")}`)
    console.log(`  Go packages: ${p.goPackages.map((x) => x.pkg.replace("github.com/Senior-Geek/istary/", "") || ".").join(", ") || "none"}`)
    console.log(`  frontend: ${p.frontend ? "tests" : "no"}${p.frontendBuild ? " and build" : ""}`)
    if (p.race.length) console.log(`  race: ${p.race.map((x) => x.pkg.split("/").pop()).join(", ")}`)
    if (p.live.length) console.log(`  live checks: ${p.live.join(", ")}`)
    if (p.gateRequired) console.log(`  FULL GATE DUE before release: ${p.gateReasons.join("; ")}`)
  ' "$PLAN_FILE" | while IFS= read -r l; do say "$l"; done
}
