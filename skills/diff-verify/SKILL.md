---
name: diff-verify
description: >
  History-aware change review. Trace every deleted or rewritten line back to
  the commit that introduced it, read that commit's message, and decide
  whether the pending change silently undoes an earlier fix or re-adds code
  that was removed on purpose. Read-only: it never edits files, runs tests,
  or writes git state. Spawns 3-4 parallel sub-agents by default; the count
  is user-overridable. Use when the user runs /diff-verify, asks "did this
  change undo an earlier fix", "verify the diff against commit history", or
  "why was this line here". Not for style review (critical-go-review),
  architecture (improve-codebase), or a failing golden fixture
  (diagnose-golden-fixture).
---

# Diff verify

A diff shows what changed. It does not show what broke. Every removed line
was added by someone for a reason, and the commit message that added it is
often the only record of that reason. This skill reads that record before
the change is accepted.

The rule: no removed or rewritten line is accepted until you have named the
commit that introduced it and what that commit was trying to do.

## Invocation

- `/diff-verify` - verify the current working tree against `HEAD`
- `/diff-verify <base-ref>` - verify the branch against a base (for example `master`)
- `/diff-verify <sha>` - verify one commit
- `/diff-verify <A..B>` or `<A...B>` - verify an explicit range
- `/diff-verify <N>` - any scope above, but run `N` sub-agents
- `/diff-verify --write` - also save the report under `plans/reviews/diff-verify/`

Count and range can be combined: `/diff-verify 5 master`.

This skill reports. It does not fix. Hand findings back to the user, then
let them invoke their normal flow.

## Phase 1 - Freeze the scope

Pick the range from the invocation, then record it. Never start archaeology
without the exact diff command written down.

| Input | Range | Command |
|---|---|---|
| nothing, dirty tree | staged + unstaged vs HEAD | `git diff HEAD`, plus untracked files from `git status --porcelain` |
| nothing, clean tree | branch vs default branch | `git diff master...HEAD`, say this assumption out loud |
| base ref `X` | merge-base to HEAD | `B=$(git merge-base X HEAD)`, then diff `$B..HEAD` |
| one sha `S` | that commit | `git diff S^..S` |
| explicit `A..B` or `A...B` | as given | as given |
| nothing at all (clean tree on master) | unknown | stop and ask for a range |

Untracked files never appear in `git diff`. List them with
`git status --porcelain` and mark them as pure additions for Phase 4.

Facts to record: branch, HEAD, base, merge-base, date, the exact diff
command, and the size of the change.

```bash
BASE=$(git merge-base master HEAD)
git status --porcelain
git diff --stat "$BASE"..HEAD
git diff --numstat "$BASE"..HEAD   # "-" means binary
```

Then read the change's own commit messages. A commit whose subject says
`refactor` or `docs` but whose diff changes behavior is a finding of its
own: the message is the only record of intent, and it is wrong.

```bash
git log --date=short --format='%h %ad %s%n%b%n---' "$BASE"..HEAD
git show --stat <sha>          # per commit, compare message against diff
```

## Phase 2 - Build the reversal work list

### 2.1 Map hunks to old line ranges

`-U0` removes context lines, so every hunk states its exact range.

```bash
git diff -U0 --no-color "$BASE"..HEAD
```

A hunk header `@@ -l,c +n,d @@` means old lines `l..l+c-1` were removed or
rewritten when `c > 0`. When `c == 0` the hunk is a pure insertion. A line
that was rewritten also counts as a reversal: the old text goes away, even
if the new text differs by one character.

### 2.2 Blame the old lines

Blame the base revision, not HEAD, because at HEAD the lines are already
gone. This is the mistake the skill exists to catch.

```bash
git blame -w -M -C "$BASE" -- <file>              # whole file
git blame -w -M -C -L <l>,+<c> "$BASE" -- <file>  # one hunk
```

Flags matter: `-w` ignores whitespace-only edits so a formatter does not
steal the blame; `-M` follows moves inside the file; `-C` follows copies,
and a second `-C` looks across files.

Before calling a deletion a loss, check whether the text reappears
elsewhere in the diff. A line that moved to another file was not removed,
it was relocated. Find the new home, cite it, and classify the change as
supersession. Binary files have no line history; give them a file-level
finding instead: who touched the file last, and what the blamed commit said.

Read every blamed commit:

```bash
git log -1 --date=short --format='%h %ad %an%n%s%n%n%b' <sha>
git show --stat <sha>
```

Known limit: blame points at the last commit that touched the line, not
the first commit that introduced it. If the blamed commit is a rename, a
format sweep, or a mechanical `refactor` with a huge diff and no stated
behavior change, climb back to the real origin:

```bash
git log -L <l>,+<c>:<file> --oneline
git log -S'<distinctive substring of the old line>' --oneline -- <file>
```

### 2.3 Lines born inside the range

A line that does not exist at `$BASE` cannot be blamed there. If one branch
commit added it and a later branch commit removes it, the reason lives in
the first commit. Find it inside the range:

```bash
git log -S'<line text>' --oneline "$BASE"..HEAD -- <file>
```

This is the intra-branch reversal case. It hides well because the final
diff against master looks clean.

### 2.4 Scan additions in reverse

For every notable addition, ask whether history already removed it once.

```bash
git log -S'<line text>' --oneline --all -- <file>
git log -G'<regex>' --oneline -- <file>       # -G for a regex, -S for a string
```

If the log shows add, then delete, then the pending add, read the delete
commit. The re-add is safe only when the delete's reason no longer holds,
and that has to be argued from evidence, not assumed.

Notable additions to scan: guards and validations, constants and
thresholds, test assertions, golden page bounds and needles, dependency
entries, exported API, `VERSION`, `//nolint` reason comments, and comments
that state a constraint.

### 2.5 Emit the work list

One row per removed, rewritten, or suspicious added line:

```
File | Hunk | Kind | Old lines | Blamed sha | Status
```

Count the removed lines in this table and compare with `--numstat`. If the
numbers disagree, a file was skipped. Fix that before classifying anything.

## Phase 3 - Classify

| Class | Meaning | Default severity | Evidence required |
|---|---|---|---|
| Accidental reversal | The blamed commit fixed a bug or enforced a contract; the removal brings it back | BLOCKER | quoted commit body, issue, or ledger, plus a current test that still expects the old behavior |
| Policy violation | The change breaks a standing rule | BLOCKER | the rule's source file, quoted |
| Re-add of deliberately removed code | History removed this once for a reason that still applies | BLOCKER or REVIEW | the removing commit's message and why it still holds |
| Intentional supersession | A newer mechanism replaces the old line | NOTE | the replacement, cited at HEAD as `file:line` |
| Obsolete cleanup | The line pointed at symbols or designs that are gone | NOTE | the commit that superseded them |
| Unproven | No rationale found | REVIEW | the exact searches run, and the words "no rationale found" |

Confidence:

- **high** - a quoted commit body, issue, or ledger proves the intent
- **medium** - a test or fixture at HEAD still expects the old behavior
- **low** - only the shape of the diff suggests intent

A low-confidence BLOCKER is not a BLOCKER. Downgrade it to REVIEW and put
the open question in the report.

## Phase 4 - Sub-agent wave

Use sub-agents to explore the diff and its history. The default is 3 or 4
sub-agents. If the user names a count, use that count. If the user does not
name one, use the default 3 or 4; that is the count we run with. If the
harness caps concurrency below the count, run the groups in waves.

Spawn each agent with the harness's read-only explore agent type. If the
harness has no explore type, use the general agent and put the hard rules
below at the top of its prompt.

Scale the count to the diff:

| Diff size | Agents |
|---|---|
| under 2 files or under 30 changed lines | 1, or the orchestrator solo |
| 2 to 5 files | 2 |
| 6 to 30 files | 3 |
| over 30 files or over about 1500 changed lines | 4 |

Split by non-overlapping file groups. Two agents never own the same file.
A split that fits this repo:

- **A - rendering core:** `internal/layout/`, `internal/css/`, `internal/html/`
- **B - output pipeline:** `internal/pdf/`, `internal/convert/`, `internal/load/`, `internal/imageout/`
- **C - tests, fixtures, binaries:** `internal/**/*_test.go`, `testdata/`, `cmd/`
- **D - contracts and rationale:** `go.mod`, `go.sum`, `VERSION`, `CHANGELOG.md`, `Makefile`, `.github/`, `documentation/`, `frontend/`, plus the cross-reference scout below

Always keep one agent on the rationale scout: cross-reference `plans/`,
`plans/PR/`, `knowledge-base/`, and GitHub PR numbers found in commit
subjects. A subject like `fix(layout): ... (#61)` means issue or PR 61 may
hold the full story.

### Sub-agent rules (hard)

- Read-only git only: `log`, `show`, `diff`, `blame`, `merge-base`,
  `rev-parse`, `status`. Banned: `add`, `commit`, `push`, `checkout`,
  `restore`, `clean`, `reset`, `stash`, `bisect`, `worktree`.
- No file edits. No `make`. No `go test`, `go build`, or any command that
  compiles. This review is about history, not current behavior.
- Quote raw evidence: exact sha, exact subject, exact body text, exact
  command. A claim without a sha is rejected.
- Include negative results. "No blamed commit found for these lines" is a
  valid finding.
- Return finding blocks only. No prose summaries.

### Finding schema (one block per line or tight line group)

```
FILE:
LINES:          old line range at base, or "born in range"
KIND:           deleted | rewritten | added
OLD_TEXT:       exact removed text, trimmed to a distinctive substring
BLAMED:         <short sha> <subject>, or none
BLAMED_BODY:    quoted paragraph from the commit message, or none
CHECKS:         exact commands run
INTENT:         one sentence: what the old line was for
CLASS:          accidental-reversal | policy-violation | re-add | supersession | obsolete | unproven
SEVERITY:       BLOCKER | REVIEW | NOTE
CONFIDENCE:     high | medium | low
ACTION:         keep | revert hunk | replaced by <file:line> | ask user
```

### Prompt skeleton for each agent

```text
You are agent <X> in a read-only diff-verify wave for the repository at
<path>. Do not edit files. Do not run make, go test, or go build. Do not run
any git command that writes state.

Scope: diff <BASE>..HEAD (<exact command>).
Your file group: <paths>.
Agent <Y> owns <other paths>; do not touch those files.

For every removed or rewritten line in your group:
1. Map the hunk to its old line range from `git diff -U0`.
2. Run `git blame -w -M -C -L <l>,+<c> <BASE> -- <file>`.
3. Read each blamed commit with `git log -1 --format='%h %ad %s%n%b'`.
4. If blame landed on a mechanical commit, climb to the origin with
   `git log -L` or `git log -S`.
5. For lines born inside the range, use `git log -S ... <BASE>..HEAD`.
6. For notable additions, use `git log -S ... --all` to check for a
   previous deliberate removal.
7. Return one finding block per schema from the skill.

Evidence rules: quote the commit body. If no rationale exists, say
"no rationale found" and list the searches. Never guess intent.
```

## Phase 5 - Verify and synthesize

1. Re-run the blame and `git show` for every BLOCKER yourself. Never cite
   a sha you have not opened. A sub-agent summary is a claim, not proof.
2. Dedupe by blamed sha. One commit can explain many lines; keep the
   strongest representative and list the rest under it.
3. Rank BLOCKER, then REVIEW, then NOTE.
4. Reconcile the work list with `--numstat`. If lines are missing, run
   that group yourself or re-spawn it once with the gap named.
5. Do not fix anything. Do not even sketch the fix unless the user asks
   after reading the report.

## Phase 6 - Report

Default: print the report in chat. Write a file only when the user asks or
passes `--write`; the path is
`plans/reviews/diff-verify/<branch-slug>-<YYYY-MM-DD>.md` (create the
directory only when writing).

```text
Diff verify: <branch> vs <base>
Scope: merge-base <sha>, <N> files, +<A>/-<D>, <date>
Verdict: CLEAR | <N> findings need a decision (<B> blockers)

Findings
DV-01 [BLOCKER] <file:lines> removed "<old text>"
  Blamed: <sha> <subject> (<date>)
  Commit said: "<quoted body>"
  Why this matters: <the fix or rule the removal undoes>
  Action: revert hunk | keep with new proof | ask user

Cleared
  <n> removed lines checked and cleared:
  <sha> <subject> - superseded by <file:line>
  <sha> <subject> - refactor only, behavior preserved by <file:line>

Unproven
  <file:lines> "<old text>"
  No rationale found. Searched: <commands>

Re-add scan
  <n> notable additions checked against history; <m> had a prior removal,
  listed above.

Commands run
  <exact commands, in order>
```

Report rules:

- Quote the commit body. Do not paraphrase it.
- "No rationale found" is a complete answer. Never invent intent.
- One line of evidence per severity claim.
- End with the decision list: what the human must choose, and what happens
  if they do nothing.
- No em dashes. Apply the unslop pass before sending.

## Guardrails

- Read-only git. Never write the index, refs, or working tree.
- To inspect an old revision on disk, use `git show <sha>:<path>` for one
  file, or a detached worktree under `/tmp/opencode/` created and removed
  by the orchestrator. Sub-agents do not create worktrees.
- Never edit code, tests, fixtures, or plans as part of this skill.
- Do not stash or clean to get a tidy view. Preserve unrelated dirt.
- This skill does not replace `make test`. It answers the question tests
  cannot answer: why the old line existed.
- If an agent returns nothing or dies, diagnose first; do not re-spawn the
  same prompt blindly. Run that group solo if needed.
- If `git` is missing or the folder is not a repository, stop and say so.

## Repo notes (gowkhtmltopdf)

Where intent is recorded: read the commit bodies first. They are short in
this repo, so also check:

- `plans/<version>/NN-canonical-*.md` ledgers and `plans/<version>/phases/`
- PR bodies under `plans/PR/` and `plans/<version>/PR/`
- performance ledgers `plans/0.2.6/perf-improve/` and `plans/0.2.6/perf-time/`
- `knowledge-base/wiki/log.md` and concept pages (local only, not in git)
- `documentation/`, `CHANGELOG.md`, `RELEASE.md`

Keyword search for rationale:

```bash
git log --all --grep='<keyword>' --oneline
rg -i '<keyword>' plans/ knowledge-base/ documentation/
gh pr view <number> --json title,body   # only if gh is installed and authed
```

Lines that are never ordinary, check them with extra care:

- dependency allowlist in `TestDirectModuleAllowlist`; only
  `go-text/typesetting` and `tdewolff/canvas` may be direct
- golden contract: `fixturePageBounds`, needles, fixture headers
- claim wording policed by `make claim-scan`
- `VERSION` and `CHANGELOG.md` discipline, `make check-versions`
- performance fast paths, because their reason is a measurement
- `//nolint` reason comments and `[DEBUG-...]` cleanup

## Worked example (illustrative)

Situation: the pending diff deletes one line from
`internal/convert/convert.go`:

```go
if req.Timeout <= 0 { req.Timeout = DefaultTimeout }
```

**Step 1, find the hunk.** `git diff -U0` reports `@@ -412,1 +412,0 @@`,
so old line 412 was removed.

**Step 2, blame the old line at the base revision.**

```bash
git blame -w -M -C -L 412,+1 "$BASE" -- internal/convert/convert.go
# a1b2c3d4 fix(convert): default the request timeout before fetch
```

**Step 3, read the blamed commit.**

```bash
git log -1 --format='%h %ad %s%n%b' a1b2c3d4
# a1b2c3d4 2026-08-11 fix(convert): default the request timeout before fetch (#57)
#
# A zero timeout made the loader block forever on slow hosts.
# Guard at the boundary so every caller gets the default.
```

**Step 4, check whether current code still expects the old behavior.**

```bash
rg -n 'DefaultTimeout' internal/convert
# internal/convert/convert_test.go still asserts the default is applied
```

**Finding:**

```text
DV-01 [BLOCKER] internal/convert/convert.go:412 removed the timeout default
  Blamed: a1b2c3d4 fix(convert): default the request timeout before fetch (#57)
  Commit said: "A zero timeout made the loader block forever on slow hosts."
  Why this matters: deleting the guard brings back the unbounded wait, and
  the test that expects the guard still exists.
  Action: keep the guard, or delete the guard and the test together and say
  what replaces the timeout default.
```

The shas and issue numbers above are illustrative.

## Definition of done

- Scope frozen with the exact range, base sha, and diffstat.
- Every removed or rewritten line has a blamed sha, or a pickaxe result
  proving it was born inside the range.
- Every BLOCKER re-verified by the orchestrator, not accepted from a
  sub-agent summary.
- Additions scanned for re-adds.
- Report contains the exact commands and the negative results.
- No files edited. No git write commands run.
- Verdict and decision list delivered to the user.

## Related skills

- `skills/diagnose-golden-fixture/SKILL.md` - when the checked change
  already broke a fixture
- `skills/critical-go-review/SKILLS.md` - quality review of the code
  itself, not its history
- `skills/improve-codebase/SKILL.md` - architecture review
- `skills/PR/PR_TEMPLATE.md` - when the verified change becomes a PR
