# Testing Istary: three levels

Istary has three levels of testing. Each answers a different question, and
each costs a different amount of time.

| Level | Command | Question | Time (M3 Pro) |
|---|---|---|---|
| 1, fast | `scripts/test-fast.sh [BASE]` | Is what I changed still right? | seconds to ~3 min |
| 2, acceptance | `scripts/test-acceptance.sh [BASE [CANDIDATE]]` | Is this milestone ready to accept? | ~10 to 15 min |
| 3, full gate | `scripts/gate.sh BASE [CANDIDATE]` | May this commit be released? | ~2 h |

BASE defaults to `main` for levels 1 and 2. All three print a timing for
every stage and a total, and end in one result line: `RESULT: PASS`, `FAIL`
or `INTERRUPTED`. The exit status is 0 only for `PASS`.

## Choosing what to run

Levels 1 and 2 start from the files changed since BASE and follow them
through the code (`scripts/lib/classify.mjs`). They never pick tests by file
name. For each changed file:

- **A Go file** belongs to a package. The tests run are that package's and
  those of every package that depends on it, in either module, test imports
  included (`go list -test`). A change to `engine` therefore runs `app`,
  `cmd/istary` and `gui` as well; a change to `app` does not run `engine`.
- **A `_test.go` file or test data** runs its own package's tests.
- **Other files in a package directory** (embedded scenario data) are
  treated as the package's code.
- **Frontend source** runs the frontend tests and a production frontend
  build, and no Go tests.
- **Recorded GUI fixtures** (`gui/frontend/src/lib/testdata`) run the GUI's
  Go tests, which record them, and the frontend tests, which read them.
- **Documentation** (`*.md`, `docs/`, `images/`) runs nothing.
- **The gate or the test scripts** run their own tests.
- **Anything else** broadens the plan to everything, and says why:
  - the Go modules (`go.mod`, `go.sum`);
  - the frontend's dependencies or build configuration;
  - a file in no package that the classifier does not know.

File names only label a change's areas. The labels choose the live checks
and say when race tests and the full gate are due:

| Area | For example | Adds |
|---|---|---|
| UI | Svelte components, CSS | live layout, sheet and scenes checks |
| Creation | `TrainingScreen`, `app/creation*`, `classguide` | live creation checks |
| Equipment and trading | `equip`, `shop`, `CharacterSheet` | live sheet and equipment checks |
| Combat and maze | `combat`, `dungeon`, `camp`, `expedition` | their packages' tests (the GUI walkthroughs are integration tests) |
| Persistence | `save`, `games`, `backups`, `settings`, `lock` | race tests; full gate due |
| Concurrency | `gui/session.go`, the locks, `cmd/istary/main.go`, `engine/game.go` | race tests; full gate due |

## Level 1: fast checks

```sh
scripts/test-fast.sh              # the working tree against main
scripts/test-fast.sh v0.11.0-castle-interface
scripts/test-fast.sh --all        # as if everything had changed
```

It tests your working tree as it is, staged, unstaged and untracked, in your
checkout. It runs:

- a compile of both Go modules, and `go vet` on the packages reached;
- the tests of every package reached, in parallel processes (below);
- the frontend tests, and a production frontend build to a temporary
  directory;
- the gate's own tests and the test scripts' tests, when those changed;
- a build of the terminal game, to a temporary file, when it is reached.

It does not run:

- the race detector;
- fixture or golden regeneration (the ordinary tests still compare against
  them);
- the Wails build;
- the live window checks.

It prints which of those are due. It fails if your checkout changed during
the run, or if the real save folders did.

## Level 2: milestone acceptance

```sh
scripts/test-acceptance.sh                          # HEAD against main
scripts/test-acceptance.sh main ux-3c-boltac
scripts/test-acceptance.sh --race --sizes 980x692   # race on every concurrent package, one size
scripts/test-acceptance.sh --no-live                # without driving the window
```

It tests a commit in a temporary worktree outside your checkout, so it
needs a clean checkout, as the gate does. It runs:

- `npm ci`, and `go vet` on both modules;
- **every** Go test of both modules, in parallel processes, and the
  frontend tests;
- the test scripts' own tests, and the gate's when it changed;
- `make build`, and the Wails app (the production frontend with it),
  verified with `codesign`;
- race tests on the packages with concurrent code (`engine`, `cmd/istary`,
  `gui`), when the change reaches concurrency or saving, or with `--race`;
- the live checks (`scripts/acceptance/live.mjs`), at 980×692 and 1440×900
  by default.

The live checks run `wails dev` in the worktree, on a free port, on a scratch
library written by `TestWriteAcceptanceLibrary`
(`gui/acceptance_seed_test.go`), and drive it through headless Chrome. The
suites are:

- **layout:** the four areas at all eight places, the page not scrolling,
  the information scrolling on its own, every action reachable, hiding the
  party;
- **sheet:** the embedded sheet and its toggle, switching members, Escape,
  Worn & Wielded and the Pack on every member, a Remove / Use round trip
  saved exactly as before, the Inn's and Temple's sheets;
- **scenes:** the illustrated places (ART-2A), the Castle and the Edge of
  Town:
  - the picture loaded, filling the window, cropped and not stretched, and
    seen (nothing painted over it) through the display;
  - the panels near-opaque, and every control in view and uncovered;
  - Hide panels, Show panels and Escape, with the focus;
  - Hide party giving its room to the panels;
  - the same at 1512×982 and 1728×1117 (the 14- and 16-inch MacBook Pro
    full screens), 2560×1440 and the smallest window, 760×560;
  - the ways between the two places, a place without artwork, the sheet
    over the picture, and the game saved exactly as before;
  - the story box (ART-2B): where it opens, dragged and resized with the
    mouse and the keys, kept inside the window, closed and not reopened by
    going elsewhere, reopened where it was left, the narrative in it and
    the saving and practice notes still below;
  - the appearances (ART-2B, ART-2C): the defaults on a library that has
    never chosen (Slate panels, a Parchment story box, the whole picture);
    Settings' three choices, clicked, taking effect at once and kept in
    `gui-settings.json`; the same appearance in another game and back, the
    game unchanged; the story box open and where it was left through
    Settings and between the places;
  - every combination, 27 at each place: shown as chosen, nothing moved
    (Hide panels gone with the picture hidden), the panels and the story
    box 90–95% opaque, all text meeting WCAG AA contrast with the picture
    behind taken at its darkest and at its lightest, and every control in
    view, uncovered and enabled;
- **creation:** decisions in the Actions, the class table at zero points,
  taking a point back, Discard (nothing saved), Keep and save.

Two more suites are run by hand, not by level 2:

- **looks** takes screenshots only: the Castle and the Edge of Town with
  the default appearance and a contrasting one (Light panels, a Dark story
  box, the picture subdued) at 980×692, 1440×900, 1512×982 and 1728×1117,
  the story box open and, at the first two, closed; and the picture hidden
  at 1440×900 (`--suites looks`).
- **restarted** checks that Istary opens with the appearance kept in
  `gui-settings.json` (the defaults, if it keeps none), then chooses
  another. Run it twice on one library, without seeding it again between:
  the second run checks what the first chose, across a real restart.

When nothing changed between BASE and CANDIDATE (a release check on
`main`), it checks as if everything had changed.

## Level 3: the full regression gate

```sh
scripts/gate.sh main ux-3c-boltac                 # the candidate in full
scripts/gate.sh --per-commit main ux-3c-boltac    # and every commit alone first
```

See [REGRESSION-GATE.md](REGRESSION-GATE.md). It is unchanged in what it
runs on the candidate:

- vet;
- tests, and tests in shuffled order;
- race on everything, the `app` package's 65 minutes included;
- the frontend;
- fixture and golden regeneration, which must change nothing;
- the Wails build and `codesign`.

By default it now tests the candidate only. `--per-commit` restores the
earlier mode, in which every commit after BASE is also tested alone.

### When the full gate is mandatory

- Before every release tag, on the merged `main` that will be tagged.
  Nothing else replaces it.
- Before merging a branch that touches persistence (saving, loading, save
  folders, migration) or concurrency (the session or save-folder locks,
  goroutines). Levels 1 and 2 report this as "FULL GATE DUE".
- Before merging a change to the Go modules or the toolchain.
- Before merging a change to the gate itself.
- Use `--per-commit` when every commit on the branch must be known good on
  its own, for example before a bisect-sensitive history is merged.

## Parallel Go tests

`scripts/lib/gotest.mjs` is how levels 1 and 2 make the Go suites fast. For
each package it:

1. compiles the test binary once;
2. lists its top-level tests;
3. deals them into shards, longest first, using measured times;
4. runs each shard as its own process: `-test.run '^(A|B|…)$'`.

A shard passes only if its process exits 0 **and** every test dealt to it
reported `--- PASS` or `--- SKIP`. A test that never ran, or a process that
ended early, fails the run, and the missing tests are named.

- **Measured times** are kept in
  `~/Library/Caches/istary-tests/timings.json` and updated after each run.
  The first run uses the committed baseline,
  `scripts/lib/test-timings.json`.
- **Each shard is a whole process.** Tests that share package state in one
  process (such as the global `math/rand`) see a different history than in
  `go test`. A test that depends on that history fails here. That is a
  latent flaw in the test, and the shuffled run in the gate can meet it too.
  One such test was found and fixed: `TestTheRescuedComeHome`, whose
  `stairs` case met a wandering party under about 1.5% of random states.
- **The floor is the slowest single test.** The slowest loop over seeds
  within one test, so they cannot be split without changing them:
  - `gui.TestRealExpeditionsThroughTheBridge`;
  - `app.TestResumeFollowsUninterruptedPlay`;
  - `app.TestCampResumeFollowsUninterruptedPlay`.

  On the real disk they spend half their time waiting for saves to be
  flushed (below).

## The RAM disk

Every save the game makes is an atomic write that calls `F_FULLFSYNC`. That
is a flush of the whole drive, and the flushes of tests running side by side
queue behind one another:

- one save-heavy test takes 58 s on the disk, but only 23 s of CPU;
- on the disk, all the Go tests in parallel take about 3 minutes, and the
  slowest test about twice as long as alone.

On macOS, levels 1 and 2 therefore put the tests' temporary files on a RAM
disk of their own (`hdiutil attach -nomount ram://…`, HFS+, about 1 GB):

- It is mounted inside the run's temporary directory, not under `/Volumes`,
  and detached when the run ends.
- The saves are written exactly as before, only to memory. The same
  save-heavy test takes 23 s.
- Durability on a real disk is still the full gate's to check: it runs on
  the real disk.

Set `TIERS_RAMDISK=0` to run on the disk, or `TIERS_RAMDISK_MB` to change
its size. If the RAM disk cannot be made, the run uses the disk and says so.

If a run is killed outright (SIGKILL) it cannot detach its RAM disk. Find it
with `hdiutil info` (a `ram://` image) and detach it with
`hdiutil detach /dev/diskN`.

## Isolation and safety

- **Isolated environment.** Every stage runs with `HOME`, `TMPDIR` and
  `ISTARY_SAVE_DIR` in the run's own temporary directory (`TMPDIR` on its
  RAM disk). Go's and npm's caches are kept. The live app's library is a
  scratch directory.
- **Real saves checked.** Both real save folders are checksummed before and
  after, and must agree:
  - `~/Library/Application Support/Istary`, the app's;
  - `~/.config/wizardry`, the terminal game's.

  If an Istary you started is running, it is named in the report; it may
  save during the run, and the check would then fail.
- **Only its own processes.** Each stage runs as its own process group, with
  a time limit, and only that group is ever stopped. The live checks start
  `wails dev`, the dev app, Vite and Chrome in groups of their own and stop
  only those.
  - At the end, level 2 checks that no process naming its temporary
    directory is left.
  - Nothing is ever stopped by application name. An Istary you are running
    is never touched.
- **Logs.** A passing run removes its temporary directory. A failing run
  keeps its logs and prints where they are.

## Measured times

Apple M3 Pro (5 performance and 6 efficiency cores), 18 GB, at
`v0.11.0-castle-interface` (`8f8114f`), with the RAM disk:

| Run | Time |
|---|---|
| Level 1, a Svelte file changed | 6 s (frontend tests and build; no Go) |
| Level 1, `app/shop.go` changed | 54 s (`app`, `cmd/istary`, `gui`) |
| Level 1, `engine/character.go` changed | 55 s (every package) |
| Level 1, `--all` | 1 min 23 s (of which the gate's own tests 25 s) |
| Level 2, `main` against itself (everything, race, live at both sizes) | 5 min 14 s |
| Level 2, a branch reaching no concurrent code | about 2 min |
| Level 3, the full gate | about 2 h (`--per-commit` adds about 8.5 min a commit) |

Within level 2, on everything:

| Stage | Time |
|---|---|
| Every Go test, both modules, parallel | 49 s (`go test` one package at a time: about 8.5 min) |
| Race tests on `engine`, `cmd/istary`, `gui`, parallel | 2 min 9 s (the gate's GUI race alone: 11 min) |
| Live checks, three suites, each size | 50 s (123 checks; the scenes suite, added since, about 2 min 45 s more with ART-2C's 54 combinations) |
| The gate's own tests | 23 s |
| Wails build and codesign | 4 s |

Without the RAM disk (`TIERS_RAMDISK=0`), the parallel Go tests take about
3 minutes.

## What the faster levels do not cover

- **Race on the `app` package** (65 minutes): only the full gate runs it.
  `app` has no goroutines of its own; its race run guards against ones
  added later.
- **Shuffled order** and **fixture and golden regeneration**: only the full
  gate. The ordinary tests still compare every fixture and golden.
- **Real-disk durability** of saves (`F_FULLFSYNC`): levels 1 and 2 save to
  a RAM disk; the gate saves to the real disk.
- **Each commit alone:** only `scripts/gate.sh --per-commit`.
- **The live checks** cover the town interface only:
  - the layout at every place;
  - the character sheet and equipment;
  - the illustrated places' pictures and panels;
  - character creation.

  Combat, the maze, camp, Boltac's purchases and the Temple's services are
  not driven live. They are covered by their Go tests, the GUI walkthrough
  tests among them.
- **The classifier's graph** is read from the checkout running it. For a
  candidate whose imports differ greatly from the checkout, use `--all` or
  the gate.
- **The terminal game's screen** is covered by its golden tests only; no
  level drives a terminal.
