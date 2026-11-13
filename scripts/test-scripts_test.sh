#!/usr/bin/env bash
# Tests of the test levels' own scripts (docs/TESTING.md), in seconds:
# their syntax, the classifier and the parallel Go runner (node --test,
# including a throwaway Go module whose tests fail), and the classifier on
# this repository's real import graph. Run by scripts/test-fast.sh and
# scripts/test-acceptance.sh when the scripts change, and by acceptance always.

set -u -o pipefail
cd "$(dirname "$0")/.." || exit 2
fail=0
for f in scripts/test-fast.sh scripts/test-acceptance.sh scripts/test-scripts_test.sh scripts/lib/common.sh scripts/gate.sh scripts/gate_test.sh; do
  bash -n "$f" || { echo "FAIL syntax: $f"; fail=1; }
done
for f in scripts/lib/*.mjs scripts/acceptance/*.mjs; do
  node --check "$f" || { echo "FAIL syntax: $f"; fail=1; }
done
node --test scripts/lib/*.test.mjs || fail=1

# The real graph: engine reaches the GUI module; a Svelte file reaches no Go.
plan=$(node scripts/lib/classify.mjs "$PWD" engine/item.go) || fail=1
grep -q '"pkg": "github.com/Senior-Geek/istary/gui"' <<<"$plan" || { echo "FAIL: engine/item.go does not reach the GUI module"; fail=1; }
plan=$(node scripts/lib/classify.mjs "$PWD" gui/frontend/src/App.svelte) || fail=1
grep -q '"goPackages": \[\]' <<<"$plan" || { echo "FAIL: a Svelte file reached Go packages"; fail=1; }

[ $fail -eq 0 ] && echo "test scripts: PASS" || echo "test scripts: FAIL"
exit $fail
