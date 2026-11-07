# TODO.md structure

`TODO.md` is the single source of truth for planned work in i3_next. This document defines its structure completely, so that humans and AI agents can edit it without changing its shape. There is no tooling that enforces these rules: **the rules are enforced by review**, so follow them literally.

If you only read one part, read [section 1 (golden rules)](#1-golden-rules) and [section 11 (checklist)](#11-pre-commit-checklist).

## 1. Golden rules

1. **Edit content, never structure.** Do not add, remove, rename or reorder the top-level sections, and do not invent new line formats.
2. **One task = one line.** A task is a single, unwrapped Markdown line. Extra detail goes into indented sub-bullets.
3. **IDs are permanent.** Never renumber, reuse or delete an ID.
4. **Never claim more than is verified.** `[x]` means code, tests and docs are done and `make check` is green. Anything less is `[ ]` or `[~]`.
5. **Priority comes from the rubric**, not from taste or effort (section 5).
6. **TODO changes ship with the code change** that causes them, in the same pull request.
7. **English only**, in the same style as the existing file.

## 2. File anatomy

`TODO.md` has a title, a short introduction, and exactly these top-level sections, in this order:

| # | Heading (exact text) | Purpose |
|---|---|---|
| 0 | `## 0. How this file works` | Short version of these rules |
| 1 | `## 1. Status snapshot` | Honest health table of the project |
| 2 | `## 2. Priority queue` | The ordered "do next" list (max 12) |
| 3 | `## 3. Backlog by area` | All tasks, grouped by area (`### 3.N …`) |
| 4 | `## 4. Release gate` | Final checklist that must be true for a release |
| 5 | `## 5. Milestones and risks` | Milestone table and risk table |
| 6 | `## 6. Progress log` | Dated history of changes to the plan |

Sections are separated by a horizontal rule (`---`). Do not add a seventh or eighth top-level section; put new material into the closest existing one.

## 3. Task line grammar

```
- [<marker>] **<ID>** `<priority>` `<type>` [`<target>`] — <Title>. [✅ <acceptance>.] [⛓ <dependencies>]
```

Concrete example:

```
- [ ] **SWAY-04** `P0` `feat` — Reconnect with backoff and rebuild the snapshot. ✅ e2e: kill and restart headless Sway. ⛓ FIX-21
```

| Field | Rule |
|---|---|
| Marker | `[ ]` open, `[~]` partial or reopened, `[x]` done (section 4) |
| ID | `PREFIX-NN`, upper-case prefix, at least two digits, optional lower-case letter suffix (section 6) |
| Priority | `P0`, `P1` or `P2` (section 5) |
| Type | one of `bug`, `sec`, `feat`, `test`, `infra`, `docs`, `debt` (section 7) |
| Target | optional release tag such as `v1.1`, `v1.2`, `v1.x`; absent means `v1.0` |
| Separator | an em dash `—` surrounded by single spaces (not `-` or `--`) |
| Title | outcome-first sentence: what will be true afterwards, not what will be "looked into" |
| ✅ acceptance | testable, names the test level (unit, e2e, VM, manual QA) or the command that proves it |
| ⛓ dependencies | comma-separated IDs; use only for real blockers |

Formatting details:

- The ID is bold; priority, type and target are in single backticks; commands, paths and identifiers in the title are also in backticks.
- Do not hard-wrap the line. Long lines are fine.
- Sub-bullets are indented by two spaces and start with `✅` or `⛓` only when the inline form would be unreadable.
- A task line never contains a table, a code fence or another task.

Good and bad:

| Bad | Why | Good |
|---|---|---|
| `- [ ] Fix the bus` | no ID, priority or type | `- [ ] **FIX-17** `P0` `bug` — Event bus compiles again. ✅ `go build ./...` passes.` |
| `- [ ] **FIX-17** P0 bug - fix bus` | missing backticks, wrong dash | as above |
| `- [x] **CFG-05** … (mostly works)` | `[x]` with caveats | `- [~] **CFG-05** … nothing calls it yet (reopened 2026-11-07).` |
| `- [ ] **API-09** `P1` `feat` — Investigate snapshots` | not outcome-first, no acceptance | `… — Clients detect gaps and refetch a state snapshot. ✅ test drops an event and observes a refetch.` |

## 4. Markers

| Marker | Meaning | Required companion text |
|---|---|---|
| `[ ]` | Open, not started or in progress | none |
| `[~]` | Partial, or reopened after being marked done | say what is missing; for reopened items add `(reopened YYYY-MM-DD: reason)` |
| `[x]` | Done and verified: code, tests, docs, `make check` green | none |

A task may be `[x]` only if its acceptance criteria are satisfied by tests or recorded QA. If a later review shows it was not, reopen it as `[~]` instead of editing history.

## 5. Priority rubric

Priority decides order. It is **not** a measure of size or of how interesting the task is.

| Level | Meaning |
|---|---|
| `P0` | Do now. At least one is true: **(a)** the build, CI or e2e is broken or reports false results; **(b)** a critical security problem or a fail-open security path; **(c)** it blocks the core user journey: *log in → usable desktop*. |
| `P1` | Needed for the target release and gives something a user or tester can see, run or rely on: a feature, a visible fix, or real hardening. |
| `P2` | Internal quality, polish or nice-to-have. May slip to a later release. |

Tie-breaks, in order:

1. **Tangibility first.** Prefer the task whose result someone can see or run right after merge.
2. **Internal-only work is `P2`**, unless a `P0`/`P1` task is blocked on it. Then it inherits the blocker's priority and gets a `⛓`.

Worked examples:

| Task | Priority | Reason |
|---|---|---|
| Package does not compile | `P0` | rule (a) |
| e2e script exits 0 when tests fail | `P0` | rule (a): false results |
| Installed session entry bounces the user to the login screen | `P0` | rule (c) |
| Lock screen must stay locked if the locker dies | `P0` | rule (b): fail-open |
| Application launcher | `P0` | rule (c): a desktop without a launcher is not usable |
| Base-image checksum check is disabled in dev tooling | `P1` | real security issue, but not on the end-user path |
| Notification center | `P1` | visible feature, needed for the release |
| Remove alias methods from the supervisor | `P2` | internal quality |
| Same alias cleanup, but a `P0` task cannot compile without it | `P0` | inherited from the blocked task |

## 6. IDs

- Format `PREFIX-NN`. Take the next free number for that prefix; gaps are fine.
- Never reuse an ID, even for a dropped task.
- A letter suffix (`FIX-00a`) is allowed only for a follow-up to an ID that already existed before the follow-up was discovered. Prefer a new number.
- The prefix describes the kind of work, not the section where the line lives.

| Prefix | Meaning | Usual area (section 3) |
|---|---|---|
| `FIX` | Defects in already-merged code; may live in any area | any |
| `STRUCT` | Repository structure and conventions | 3.1 |
| `CORE` | Daemon, lifecycle, supervisor, event bus, state | 3.2 |
| `CFG` | Configuration | 3.3 |
| `SWAY` | Sway adapter and window management | 3.4 |
| `API` | Core ⟷ shell API | 3.5 |
| `UI` | Quickshell/QML shell | 3.6 |
| `SVC` | System services | 3.7 |
| `DBUS` | D-Bus integration | 3.8 |
| `PLG` | Plugins | 3.9 |
| `STO` | Storage | 3.10 |
| `SEC` | Security and privacy | 3.11 |
| `OBS` | Logging, diagnostics, CLI | 3.12 |
| `SES` | Session and login | 3.13 |
| `PKG` | Packaging and install | 3.14 |
| `QA` | Tests and quality assurance | 3.15 |
| `PERF` | Performance and reliability | 3.16 |
| `DOC` | Documentation | 3.17 |
| `CI` | CI/CD and release engineering | 3.18 |
| `FEAT` | Features from `docs/new_features.md` | 3.19 |

A new prefix needs a new row here and in `TODO.md` section 0.4, plus either a new area or a clear home in an existing one.

## 7. Types

| Type | Use for |
|---|---|
| `bug` | Behaviour that is wrong |
| `sec` | Security hardening or a security defect |
| `feat` | New user- or system-visible capability |
| `test` | Tests and test infrastructure behaviour |
| `infra` | Build, CI, tooling, repository plumbing |
| `docs` | Documentation and decision records |
| `debt` | Internal cleanup without user-visible change |

If a task fits two types, use the one that explains why it exists (a fail-open lock bug is `sec`, not `bug`).

## 8. Section rules in detail

### 8.1 Section 0 — How this file works

A short, self-contained summary of sections 3–7 of this document (format, markers, rubric, types, prefixes, definition of done, editing rules). If you change a rule here, change it there in the same pull request.

### 8.2 Section 1 — Status snapshot

- Starts with one line: `As of YYYY-MM-DD`, the review method (**static review** or **executed**), and `Current milestone: **Mx — name**`.
- One table with columns `Area | State | Evidence`.
- State is one of `✅`, `⚠️` (with a short word such as `skeleton`, `stub`, `false-green`), or `❌`.
- Evidence names concrete facts and the task IDs that fix them. Never write a state you did not verify; say "static review" when you only read code.
- Update the table whenever a merged change moves an area.

### 8.3 Section 2 — Priority queue

- A numbered list, **at most 12 items**, in the order work should happen.
- Line format: `N. **ID** — outcome in one phrase`. Exactly one bold ID per line; other IDs may appear as plain text.
- Only open (`[ ]`) or partial (`[~]`) tasks. When a task is done, remove it and renumber the list.
- Derived from the rubric (section 5): first things that make the repository honest (P0 a/b), then the shortest path to something a user can run (P0 c), then the most tangible P1 work.
- A closing paragraph named "Next wave" may list candidates that are not queued yet.
- Re-derive the queue after every merged task.

### 8.4 Section 3 — Backlog by area

- Subsections are `### 3.N Area name`, numbered 3.1 … 3.19 in the order of the table in section 6. Do not renumber existing areas; append new areas at the end.
- Each area has two bold labels, in this order: `**Open**` and `**Done**`. An area with no done tasks omits `**Done**`; an area with no open tasks omits `**Open**`.
- **Open** lists `[ ]` and `[~]` tasks sorted by priority (`P0` → `P2`); inside one priority keep related tasks together and put the most tangible first.
- **Done** lists `[x]` tasks, sorted by ID prefix then number. Titles are short; no acceptance text is needed.
- A task lives in exactly one area. If it touches several, put it where the user-visible result lands and use `⛓` for the rest.
- The `FEAT` area (3.19) may group tasks under italic epic labels (`*Epic A — name*`). Epic grouping is the only extra structure allowed inside an area, and an epic label is never a task.

### 8.5 Section 4 — Release gate

- A checklist of conditions that must all hold at the same time for a release.
- These lines use `- [ ]` / `- [x]` **without** an ID or bold; related task IDs appear in parentheses at the end.
- Tick a line only when every task it references is `[x]` and the condition is proven by a test or recorded QA.
- Add a line only for a condition that blocks a release; do not mirror ordinary tasks here.

### 8.6 Section 5 — Milestones and risks

- Milestone table columns: `# | Milestone | Scope | Exit criteria (measurable)`. Milestones `M0…Mn` lead to the release; `F1…Fn` are post-release feature trains.
- Exit criteria are measurable (a command passes, a test exists), not intentions.
- Risk table columns: `Risk | Impact | Mitigation`. Add a risk when a plan depends on something uncertain; remove it only when the uncertainty is resolved, and say so in the progress log.

### 8.7 Section 6 — Progress log

- A two-column table `Date | Change`, newest first.
- One row per merged change that moves a task, reopens one, or alters the status table. Mention IDs.
- Dates use `YYYY-MM-DD`. Never edit or delete old rows.

## 9. Workflows

### 9.1 Add a task

1. Pick the area and the prefix; take the next free number.
2. Write one line in the grammar of section 3, with outcome-first title and testable acceptance.
3. Choose priority with the rubric; insert the line in **Open**, sorted.
4. If it executes commands, reads user files, or changes outputs, put the security or safety constraint into the acceptance.
5. Re-derive the queue if the task can enter the top 12.

### 9.2 Finish a task

1. Verify the acceptance criteria (tests, `make check`, `make vm-test` where relevant).
2. Change the marker to `[x]`, move the line to **Done** of its area, shorten the title.
3. Remove it from the queue and renumber.
4. Tick release-gate lines whose conditions are now fully met.
5. Update the status table and add a progress-log row.

### 9.3 Partially finish or reopen a task

- Partial: set `[~]` and state in the title what is still missing.
- Reopened after review: set `[~]`, keep it in **Open**, append `(reopened YYYY-MM-DD: reason)`, and add a progress-log row.
- If the missing part is large, create a new task (new ID) and reference it from the `[~]` line.

### 9.4 Report a defect found in merged code

Create a `FIX-NN` task (type `bug`, `sec` or `test`). If it makes an existing `[x]` task false, also reopen that task and reference the `FIX` ID in the line.

### 9.5 Split a task

Keep the original ID for the part that stays; create new IDs for the new parts. Reference the new IDs from the original line. Do not reuse the original ID for different work.

### 9.6 Drop a task

Keep the line, flip it to `[x]`, start the title with `DROPPED —` and the reason, move it to **Done**, add a progress-log row.

### 9.7 Re-prioritize

Change the priority tag, re-sort within **Open**, re-derive the queue, and add a progress-log row that names the IDs and the reason.

### 9.8 Add a feature idea

Features are tracked as `FEAT-NN` tasks in area 3.19. Every feature needs: a type and priority, a target release tag, testable acceptance criteria, dependencies on existing tasks, and a note on limits and security. Update `docs/new_features.md` with the new `FEAT` IDs so the two files stay traceable.

### 9.9 Start a new milestone or release train

Add a row to the milestone table with measurable exit criteria and list the scope as IDs. If the milestone changes what blocks a release, update section 4.

## 10. Guidance for AI agents

When you are asked to update `TODO.md`:

1. Read the whole file first, including section 0 and the progress log. Do not infer structure from a single section.
2. Decide changes from evidence in the repository (code, tests, docs). If you only read code, say "static review" in the status snapshot and do not tick items that need a test result.
3. Preserve all existing IDs and all existing lines unless a rule above tells you to move or reword them.
4. Keep tasks on one line; keep the heading text exact; keep tables as tables.
5. Do not mark anything `[x]` because code "looks done". Without a passing test or recorded QA, use `[~]` and say what is missing.
6. Do not invent IDs in the queue or in `⛓`; every referenced ID must exist as a task line.
7. When you change priority or status, add the matching progress-log row in the same edit.
8. Prefer a new, precise task over making an existing task vaguer.
9. If the request conflicts with these rules, follow the request only where it does not break the structure, and say what you left out and why.
10. After editing, run through the checklist below and report any item you could not satisfy.

## 11. Pre-commit checklist

Structure
- [ ] The seven top-level sections exist with the exact headings and in the order of section 2; no new top-level section.
- [ ] Area subsections are `### 3.N …`, in order, each with `**Open**` and/or `**Done**` labels in that order.

Task lines
- [ ] Every task line follows the grammar in section 3 (marker, bold ID, priority, type, optional target, em dash).
- [ ] No task line is wrapped; extra detail is in indented sub-bullets.
- [ ] No ID is duplicated, reused or renumbered; new IDs are the next free numbers.
- [ ] Every `⛓` and every ID in the queue refers to an existing task.
- [ ] Acceptance criteria are testable; titles are outcome-first.

Order and priority
- [ ] **Open** lists are sorted `P0 → P2`; **Done** lists are sorted by ID.
- [ ] Priorities follow the rubric; any change is explained in the progress log.
- [ ] The queue has at most 12 items, only open or partial tasks, and was re-derived.

Truthfulness
- [ ] Every `[x]` is backed by passing tests or recorded QA.
- [ ] Reopened items carry `(reopened YYYY-MM-DD: reason)`.
- [ ] The status snapshot has today's date, the review method, and only verified states.
- [ ] Release-gate lines are ticked only when all referenced tasks are `[x]`.

History
- [ ] A progress-log row (newest first, `YYYY-MM-DD`) describes this change.
- [ ] If rules changed, section 0 of `TODO.md` and this document were updated together.