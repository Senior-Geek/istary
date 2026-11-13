# The regression gate

`scripts/gate.sh` is the check a branch passes before it is merged or
released. It tests the branch's tip in full, and with `--per-commit` every
commit on the branch alone first, and reports one result. It is the third and
most thorough of Istary's testing levels; see [TESTING.md](TESTING.md) for
the faster two and for when the gate is mandatory.

## Running it

```sh
scripts/gate.sh [--per-commit] BASE [CANDIDATE]
scripts/gate.sh main g1-regression-gate
scripts/gate.sh --per-commit main g1-regression-gate
```

- **BASE:** the commit the work starts from, usually `main` or the last
  release tag.
- **CANDIDATE:** the commit to gate. It defaults to `HEAD`.
- **`--per-commit`:** also test every commit after BASE alone, before the
  candidate. Without it, only the candidate is tested.
- BASE must be a proper ancestor of CANDIDATE. A ref that names nothing,
  looks like an option or is ambiguous (a branch and a tag with one name)
  is refused; use `refs/heads/NAME` or `refs/tags/NAME` instead.

Run it from a checkout of Istary with a **clean working tree**: no changes
and no untracked files. The gate tests commits, not your files, so it
refuses to start rather than let uncommitted work look tested. It never
stashes, resets, cleans or checks anything out in your checkout.

## What it runs

Each commit is checked out in its own temporary worktree, outside the
repository. Your checkout is never built in or changed.

**Each commit after BASE, alone (with `--per-commit` only):**

- root `go vet ./...` and `go test ./engine ./cmd/istary ./app`;
- GUI `go vet ./...` and `go test ./...`;
- frontend `npm test`.

**The candidate in full:**

- root vet, tests, tests in shuffled order, `make build`, and tests under
  the race detector;
- GUI vet, tests, shuffled tests and race tests;
- frontend `npm ci` (from the npm cache when it can) and `npm test`;
- fixture regeneration (`ISTARY_UPDATE_FIXTURES=1` in `gui/`, `-update`
  for the terminal goldens), which must change nothing;
- the Wails build for `darwin/arm64`, verified with `codesign`.

Finally it checks that your checkout is exactly as it was (same branch and
commit, still clean) and that no worktree it made is still registered.

The gate stops at the first stage that does not pass.

## Platform and dependencies

The full gate runs on **macOS on Apple silicon** only, because of the
Wails build and its `codesign` check. Elsewhere it refuses to start. The Go
and frontend tests themselves run on any platform, by hand.

It checks for these before starting and names any that is missing:

- bash 3.2 or later (the macOS `/bin/bash` will do);
- git 2.17 or later;
- Go, and a C compiler (`cc`) for `-race` and cgo;
- Node.js 20 or later, and npm;
- make, perl (for running stages with a time limit), codesign, shasum;
- the Wails CLI: `wails` on `PATH`, else `$(go env GOPATH)/bin/wails`, or
  set `WAILS=/path/to/wails`.

## Time and timeouts

A full run takes about two hours: the candidate's checks alone. The root
race tests take about 65 minutes of that (the `app` package). With
`--per-commit`, each commit tested alone adds about 8 to 9 minutes.

- **Every stage** has a limit, `GATE_STAGE_TIMEOUT` seconds (default 3600).
- **The root race tests** have `GATE_ROOT_RACE_TIMEOUT` seconds (default
  9000, which is 150 minutes). It is passed to Go's own `-timeout`.
- **Race stages** get 10 more minutes beyond Go's limit, so Go's timeout
  fires first with its own report.

Each stage runs as its own process group. At its limit the gate stops that
group and nothing else (TERM, then KILL 10 seconds later) and reports
`TIMEOUT`. A Go `panic: test timed out` is reported as `TIMEOUT` too.

## Logs

Each stage's full output goes to its own file, for example
`20-tip--root-race.log`. The gate's own report is copied to `gate.log`. Both
are in a new temporary directory (`$TMPDIR/istary-gate-log.*`), or in
`GATE_LOG_DIR`, which must be outside the repository. The logs are kept
after the run.

For each stage the gate prints:

- its name and the command;
- the result (`PASS`, `FAIL` or `TIMEOUT`) and the time it took;
- the path of its log.

For a stage that did not pass, it also prints the last lines of the log.

## The result

The summary at the end gives:

- the base and candidate commits;
- how many commits were tested alone;
- whether the tip checks all ran;
- the number of stages run and the first that failed;
- whether the worktrees were removed;
- where the logs are;
- the exit status.

The final line is `== RESULT: PASS`, `FAIL` or `INTERRUPTED`. It is `PASS`
only when every stage passed and every worktree was removed.

| Exit | Meaning |
|---|---|
| 0 | every stage passed; cleanup succeeded |
| 1 | a stage failed |
| 2 | the gate did not start: usage, platform, a missing dependency, not Istary, a dirty tree, a bad ref or ancestry |
| 4 | a worktree could not be removed (reported, with its path) |
| 124 | a stage timed out |
| 129, 130, 143 | interrupted by SIGHUP, SIGINT (Ctrl-C) or SIGTERM |

## Interruption and cleanup

On Ctrl-C, SIGTERM or SIGHUP, the gate:

1. stops the running stage's process group;
2. removes its worktrees;
3. prints the summary;
4. exits with 130, 143 or 129.

It removes only the worktrees it created. It never runs `git worktree
prune` or touches any other worktree.

If a removal fails, the gate says which worktree and exits nonzero. Remove
it yourself with `git worktree remove --force PATH`.

## Testing the gate

`scripts/gate_test.sh` tests the gate's own control flow in about 25
seconds, without the real test suite. Each test runs the real gate in a
throwaway repository, with stub tools and a git wrapper that injects
failures. It covers:

- failures a pipeline would hide;
- a failing build;
- missing dependencies and the wrong platform;
- a dirty tree;
- bad, option-like and ambiguous refs;
- wrong ancestry;
- a failing worktree;
- a failing stage;
- timeouts;
- SIGINT and SIGTERM;
- a cleanup failure;
- changed fixtures.

For each test it checks:

- the exit status and what the gate reported;
- that no worktree, temporary directory or stage process is left;
- that the repository is unchanged.

Run `scripts/gate_test.sh` after any change to the gate. Run
`scripts/gate_test.sh timeout interrupt` to run only some groups of tests.
