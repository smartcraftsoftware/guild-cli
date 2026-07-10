# OTel PR Cost Tracking — guild-cli Removal Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the hook-based Claude Code cost-tracking machinery from guild-cli, now that guild-4 ingests cost/token data directly from Claude Code's OpenTelemetry export (see the companion guild-4 plan). Everything else in guild-cli (`auth`, `config`, `issues`, `time`, `context`, `version`, `completion`) is untouched.

**Architecture:** Pure deletion — no new code. Three commands (`guild commit cost`, `guild hook *`, `guild setup claude`) and their supporting internals are removed, along with their `cmd/root.go` registration and their README documentation.

**Tech Stack:** Go, Cobra, the standard library `testing` package (no testify in this repo).

**⚠️ Do not start this plan until the companion guild-4 plan has shipped and the OTel ingestion pipeline is confirmed working in production.** Removing `guild commit cost` / the hooks before that point would leave a gap with no cost tracking at all. This is called out in both plans' Rollout sections.

**Companion plan:** `docs/superpowers/plans/2026-07-10-otel-pr-cost-tracking-guild4.md` (in the `guild-4` repo).

**Spec:** `docs/superpowers/specs/2026-07-10-otel-pr-cost-tracking-design.md` (in the `guild-4` repo — this repo has no local copy; read it there before starting).

---

## Task 1: Confirm nothing else depends on the code being removed

**Files:** none modified — this is a verification-only task.

- [ ] **Step 1: Confirm the removal scope is fully self-contained**

Run, from the repo root:
```bash
grep -rln "gitRevParseHead\|gitDetectRepo\|parseGitRemote\|isGitCommitCommand\|reportCommitCost\|claudeSettingsPath\|installClaudeHooks\|mergeHookEntry" cmd/ internal/ main.go
```
Expected output: exactly `cmd/hook.go`, `cmd/setup.go`, `cmd/commit.go` — nothing else. If any other file appears, stop and investigate before proceeding; it means something outside the three files being deleted depends on this code.

- [ ] **Step 2: Confirm no test file references the commands being removed**

Run:
```bash
grep -rln "newCommitCmd\|newHookCmd\|newSetupCmd\|commit cost\|hook session-start\|hook post-tool-use\|hook stop\|setup claude" cmd/*_test.go
```
Expected output: empty (no existing test file references these — confirmed during design research; there is no `hook_test.go`, `commit_test.go`, or `setup_test.go` in this repo today).

---

## Task 2: Delete the three command files

**Files:**
- Delete: `cmd/hook.go`
- Delete: `cmd/commit.go`
- Delete: `cmd/setup.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Delete the files**

```bash
git rm cmd/hook.go cmd/commit.go cmd/setup.go
```

- [ ] **Step 2: Remove their registration in `cmd/root.go`**

Two identical blocks need the same three lines removed — one in `init()`, one in `NewRootCmd()` (this repo keeps a duplicate wiring function for testing; see the comment above `NewRootCmd()`: "Subcommands are added here to keep wiring in one place" — despite the comment, there are genuinely two copies, both need editing).

In `init()`, remove:
```go
	rootCmd.AddCommand(newCommitCmd())
	rootCmd.AddCommand(newHookCmd())
	rootCmd.AddCommand(newSetupCmd())
```
(keep `rootCmd.AddCommand(newContextCmd())` and everything else)

In `NewRootCmd()`, remove the matching:
```go
	root.AddCommand(newCommitCmd())
	root.AddCommand(newHookCmd())
	root.AddCommand(newSetupCmd())
```

- [ ] **Step 3: Update the stale "report token costs" description**

Both `rootCmd` and the copy inside `NewRootCmd()` have identical `Short`/`Long` text mentioning cost reporting, which is no longer accurate once this command is gone — update both copies from:
```go
	Short: "Guild CLI — terminal interface to SmartCraft's Guild platform",
	Long: `Guild CLI provides a focused developer interface to SmartCraft's Guild
project tracking platform. Manage issues, log time, report token costs,
and pull context — all from the terminal.`,
```
to:
```go
	Short: "Guild CLI — terminal interface to SmartCraft's Guild platform",
	Long: `Guild CLI provides a focused developer interface to SmartCraft's Guild
project tracking platform. Manage issues, log time, and pull context —
all from the terminal.`,
```

- [ ] **Step 4: Verify it builds**

Run: `go build ./...`
Expected: no errors. (This is the real test here — Go's compiler will immediately catch any remaining reference to the deleted functions/types, which Task 1 already confirmed don't exist elsewhere, but this is the authoritative check.)

- [ ] **Step 5: Run the existing test suite**

Run: `go test ./...`
Expected: all existing tests pass unchanged (none of them touch the removed commands, per Task 1 Step 2).

- [ ] **Step 6: Commit**

```bash
git add cmd/root.go
git commit -m "Remove hook-based Claude Code cost tracking (superseded by OTel ingestion in guild-4)"
```

---

## Task 3: Update README.md

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Remove the "Claude Code Integration" section**

Delete the entire section from `## Claude Code Integration` (currently line 48) through the blank line immediately before `## Commands` (currently line 124) — this covers "How it works", both "Setup" subsections (individual developer and Claude Enterprise managed settings), and "Manual commit cost reporting".

Replace it with a short pointer instead of deleting silently, so anyone who remembers this section doesn't think the feature vanished without explanation:

```markdown
## Claude Code AI Cost Tracking

AI token cost tracking now happens entirely on the Guild server side, via Claude Code's native OpenTelemetry export — there is nothing to install or configure in this CLI. See your Guild instance's admin documentation for the one-time, org-wide Managed Settings configuration.
```

- [ ] **Step 2: Remove the now-dead rows from the Commands table**

Delete these four rows from the `## Commands` table:
```
| `guild commit cost` | Report token cost for a commit |
| `guild setup claude` | Install Claude Code hooks into settings.json |
| `guild hook session-start` | Hook: prompt login if not authenticated |
| `guild hook post-tool-use` | Hook: capture git commits during a session |
| `guild hook stop` | Hook: report session cost to Guild on exit |
```
(Yes, that's 5 rows, not 4 — count them directly in the file rather than trusting this plan's arithmetic; grep for `commit cost`, `setup claude`, and `hook ` in the table to make sure all matching rows are gone.)

- [ ] **Step 3: Remove the stale session-state note in the Configuration section**

Delete this line (currently under `## Configuration`):
```
Session data captured by hooks is stored temporarily in `~/.guild/sessions/` and cleaned up after each session ends.
```

- [ ] **Step 4: Update the README's opening description to match**

`README.md`'s top-level description (near line 3) says roughly "Manage issues, log time, and report token costs — all from the terminal", mirroring the CLI's own `Long` description fixed in the companion guild-4-side `cmd/root.go` edit. Update it to drop "report token costs" the same way.

- [ ] **Step 5: Verify no dangling references remain**

Run:
```bash
grep -n "guild commit cost\|guild setup claude\|guild hook\|~/.guild/sessions" README.md
```
Expected: no output.

Note: this same grep will *not* catch `guild pr cost`, which the README (and `cmd/root.go`'s help text) documents but which doesn't exist anywhere in this codebase — a pre-existing, unrelated stale reference (the `pr` command was removed in an earlier PR per git history, but its README mention wasn't). Out of scope for this plan; flag it separately if picked up.

- [ ] **Step 6: Commit**

```bash
git add README.md
git commit -m "Remove Claude Code hook documentation from README"
```

---

## Task 4: Final check and PR

- [ ] **Step 1: Run the full build and test suite one more time**

```bash
go build ./... && go test ./...
```

- [ ] **Step 2: Push and open PR**

```bash
git push -u origin <branch-name>
gh pr create --fill
```

Note in the PR description that this depends on the guild-4 OTel ingestion pipeline already being live — link the companion PR.
