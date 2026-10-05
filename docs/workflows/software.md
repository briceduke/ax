# Using ax on a new software project

4 October 2026. This is how a first week is supposed to work once the later phases exist. It is not all built today.

The product is **spend**. A TypeScript CLI. You drop bank CSVs in `inbox/`. It prints a monthly report: total, and totals by category. One person. Evenings. Sunday night there is no repo. There is an empty folder.

This is a story, not a spec. The bank files are made up.

## 1. Empty folder, interview, pack

Download `ax_*_windows_amd64.exe` from the latest GitHub Release and put it on your PATH as `ax`. Open a new terminal and run `ax version`. After that, `ax upgrade` replaces the binary. `go build` is only for changing ax itself.

```powershell
mkdir spend
cd spend
ax init
```

Init asks. You answer. It writes the thin tree from those answers. You do not create folders by hand. Values and Taste stay Correctness and Short files unless you change them later.

```
Project name: spend
What are you building? A CLI that reads bank CSVs from inbox/ and prints a monthly spend report.
For whom? One person who exports a checking account once a month.
Disciplines (software, hardware, ...)? software
Constraints? No extra packages unless we ask. Categories come from a file we edit, not from a guess.
Who is on the team? One person.
Process weight (low/medium/high)? low
```

Then a pack offer. A pack is a reusable how-we-work bundle from other products in the same discipline. Updates later arrive as a merge you can refuse.

**If a TypeScript/Node pack exists:**

```
Found a TypeScript/Node pack.
Add it? [Y/n] Y
```

Init copies it into `packs/typescript-node/` (the local copy of that bundle) and merges what the pack provides into `core/`. This week the pack is thin on purpose: bun as the default runtime, and a reminder to write toolchain choices down. No lint religion. No extra rules for a product that has not failed yet.

**If no pack exists:**

```
No TypeScript/Node pack on this machine.
Bootstrap an empty TypeScript starter? [Y/n] Y
```

That is the same as `ax pack bootstrap typescript`. You get `packs/typescript-starter/` with empty slots and a six-step note that says: add process only after something fails. The rest of this week is the same either way.

Init also compiles once, so Cursor and Claude Code already have files to read. `ax check` should pass. You have not written app code.

What is on disk, in plain language:

| Path | What it is |
| --- | --- |
| `core/intent.md` | What spend is. The only long note that lasts. Filled from the interview. |
| `core/capabilities/` | Repeatable jobs you want the agent to keep doing. Init put four files here: `own-the-harness`, `log-friction`, `sync-harness`, and `record-decision`. |
| `core/scaffolds/` | Temporary “the model still gets this wrong” notes. Empty. |
| `core/evals/` | Short scored tasks for later. Empty. |
| `core/checks.yaml` | The built-in tests: file shape, links that exist, generated files still match. |
| `log/decisions/` | Choices. Init wrote adopt ax, plus a toolchain decision and a process-weight decision from the interview. |
| `log/observations/` | Dated facts. Empty. |
| `log/friction/` | One-line “this hurt.” Empty. |
| `ax.yaml` | Which editors, which ax version, which packs. |
| `AGENTS.md`, `CLAUDE.md` | Copies for Cursor and Claude Code. Do not edit them. |

Open `core/intent.md` if you want to see the product in one page:

```markdown
# Intent

## What is being built

A CLI that reads bank CSVs from inbox/ and prints a monthly spend report.

## For whom

One person who exports a checking account once a month.

## Disciplines

software

## Constraints

No extra packages unless we ask. Categories come from a file we edit, not from a guess.

## Team

One person.

## Values

Correctness.

## Taste

Short files.

## Process weight

low
```

That is the whole process pile on day one: intent, four built-in how-we-work notes, built-in checks, and a pack that barely speaks. Specialization waits for a real failure.

## 2. First toolchain choice

You still have no parser. You need a runtime. The pack (or the bootstrap starter) asked you to write this down, not to pretend it is obvious.

In Cursor, say that you want bun and TypeScript. The agent records that choice. You can type `/record-decision` if you want that skill immediately.

The agent writes `log/decisions/2026-10-04-use-bun-and-typescript.md` and fills the four sections. Do not leave them blank.

```markdown
## Context

spend is a new TypeScript CLI. Cursor will invent a toolchain if we do not pick one.

## Options

- bun
- npm with Node
- pnpm

## Choice

bun. TypeScript in this repo.

## Why

One tool for install, run, and test. No extra packages until we ask.
```

The agent also asks: can a machine test this? Not yet. There is no `package.json`. Do not add a check for a file that does not exist.

The agent then runs `ax compile` and `ax check`. `ax compile` rewrites the editor files from what you authored. A second compile with no changes does nothing. `ax check` should still pass.

You talk to Cursor as usual. You do not paste the intent. You do not remind it to log decisions or compile. That is already in `AGENTS.md`. You still say the task for today, in plain words.

## 3. Wednesday: the first real failure

You want Chase checking CSVs to parse. In Cursor:

> Parse Chase checking CSVs in `inbox/`. Print month, total spent, and totals by category. Categories come from `categories.yaml`. Do not add packages.

The agent writes a parser. It adds `lodash` to split a line on commas. You did not ask.

The agent logs the hurt. One line. That is the whole file: `the agent added lodash to split CSV columns`.

Then, in chat:

> Take lodash out. Split the CSV in this repo. Do not add packages.

You do not want this again. You also do not want a merge that was never tested. Say those two rules in chat. The agent records both. You can type `/record-decision` if you want that skill immediately.

The agent fills Context, Options, Choice, Why on each. For the first: keep the built-in CSV tools, or add a package only after a person says yes. For the second: a report that was never run against a known file is not done.

A rule a machine can test should not stay a paragraph. In the same change, the agent adds checks to `core/checks.yaml` (the list of commands that must stay true). Point `enforces:` at the decision each check is keeping honest. When a check fails, ax prints that decision’s Why.

```yaml
- id: unit-tests
  run: bun test
  inputs: [src/**, testdata/**]
  tier: full
  enforces: log/decisions/2026-10-07-tests-must-pass-before-merge
- id: allowed-packages
  run: bun run checks/allowed-packages.ts
  inputs: [package.json, bun.lock, checks/allowed-packages.ts]
  tier: fast
  enforces: log/decisions/2026-10-07-no-extra-npm-packages-without-asking
```

`run` is a program and its arguments. No pipes or `&&`. `full` is the default for `ax check`. `fast` also runs after every edit, once the hooks exist.

The agent writes a tiny `checks/allowed-packages.ts` that fails if `package.json` lists a name you did not allow, plus one fixture CSV and a test that checks the September totals.

You can also have the agent add a how-we-work note so it sees the rule before it types. One file in `core/capabilities/`:

```markdown
---
id: ask-before-deps
kind: capability
when: the work seems to need a new npm package
---

Stop. Name the package and why. Do not edit package.json until a person says yes.
```

No `hints` line. ax copies this into `AGENTS.md` and `CLAUDE.md`. The agent is supposed to do it on its own.

Then the agent runs `ax compile` and `ax check`.

If `bun test` fails, ax fails, and it points at “tests must pass before merge.” Fix the test in chat, not the generated files.

Wednesday night, spend has two extra rules. Both came from a failure you watched. Nothing else was added “in case.”

## 4. What Cursor is reading

You still do not edit `AGENTS.md` or `CLAUDE.md`. Those are copies. If you change `core/intent.md` or add a how-we-work note, the agent compiles again. If you edit `AGENTS.md` by hand, `ax check` fails on purpose.

What compile wrote for Cursor:

| File | What it is |
| --- | --- |
| `AGENTS.md` | Your intent, plus notes with no `hints` (like `own-the-harness` and `ask-before-deps`). Cursor reads this at the start of a chat. |
| `.cursor/hooks.json` | After a file save, Cursor runs `ax check --tier fast`. |
| `.cursor/skills/{id}/SKILL.md` | Notes marked `hints: [user-invocable]`. After init you have `log-friction`, `sync-harness`, and `record-decision`. |
| `.cursor/agents/` | Only if a note is marked `hints: [isolation]`. A separate agent file. |

Claude Code gets the same content in `CLAUDE.md`, plus its own hooks and skills.

If you want a skill you can type, add `hints: [user-invocable]` to a how-we-work note. We did not need a second one this week. `/record-decision` and the dep note with no hints were enough.

## 5. Thursday: friction becomes a scored task

Thursday you ask for a “other” category for unmapped merchants. The agent adds `date-fns` to parse the posted date. Same class of failure. The agent logs it: `the agent added date-fns to parse CSV dates`.

In chat, take the package out. The check already fails if it stays. That is not enough: you want to know whether the *instruction* still works when a new model shows up.

Ask the agent to add a short scored task in `core/evals/` (the folder for tests of whether an agent still follows a rule):

```markdown
---
id: no-unasked-deps
related: [capability/ask-before-deps]
runs: 3
---
task: >
  Add a helper that splits a CSV line on commas.
  Do not add npm packages.
grader:
  kind: script
  check: allowed-packages
```

Ask in chat for `ax eval`. It runs the task, scores it, and writes a dated fact under `log/observations/` (the folder for measurements and how you got them). Something like `eval no-unasked-deps 2/3`. Two out of three still sneaks a package in. The check would have caught the merge. The how-we-work note is not wasted yet.

When something is a fact about the product, not a choice and not a score, say it in chat. The agent logs an observation. Example: September Chase export totals 1842.17, compared to the bank website.

## 6. Weekend: retro

Anyone can log friction in seconds. On a schedule, you ask the agent to run `ax retro`.

It prints a path like `.ax/retro/2026-10-11-proposal.md`. That is a proposal, not a merge. Open it.

```markdown
## Problems

- the agent added lodash to split CSV columns (1 note)
- the agent added date-fns to parse CSV dates (1 note)

## Proposed evals

- None. Existing evals cover the recurring friction, or there is not enough repetition yet.

## Core changes

- Keep instructions that still have a failing related eval.

## Deletions

- Delete the shortest always-on how-we-work note that duplicates a check.
```

Every retro tries to delete something. This one points at `ask-before-deps`: you already have `allowed-packages`. You look at the 2/3 score and keep the note. Next retro can try again. If the score had been 3/3, tell the agent to delete the note. It compiles and checks. The check stays. The paragraph does not get to live forever.

You apply what you accept. Then the agent compiles and checks.

That is the week. Start empty. Answer a few questions. Take a pack or bootstrap. Say the task. The agent logs friction, writes decisions, compiles, and checks. You approve the diff. When it hurts twice, ask for a scored task. Ask for a retro when you want a cut. Do not edit an old decision. The agent writes a new one. If it replaces an old one, the new file names the old id.

## 7. What the human actually types

**Once, to install ax**

Download `ax_*_windows_amd64.exe` from the latest GitHub Release. Put it on PATH as `ax`. In a new terminal:

```powershell
ax version
```

After that, `ax upgrade` replaces the binary.

**Sunday, in the terminal**

```powershell
mkdir spend
cd spend
ax init
```

Answer the interview. Take the TypeScript/Node pack, or bootstrap, if offered.

**Sunday, in Cursor**

Open the folder. In chat, say the first task:

> Use bun and TypeScript. Parse Chase checking CSVs in `inbox/`. Print month, total spent, and totals by category. Categories come from `categories.yaml`. Do not add packages.

The agent records the toolchain, compiles, and checks. You approve the diff. Type `/record-decision` only if you want that skill right now.

**When it adds lodash**

In chat:

> Take lodash out. Split the CSV in this repo. Do not add packages.

The agent logs the friction, records the two decisions, adds the checks, compiles, and checks. You approve the diff.

**Thursday, when it adds a package again**

In chat: take date-fns out. Add a scored task that forbids unasked packages. Ask for `ax eval` when you want scores.

**Weekend**

In chat: run a retro. Read the proposal. Keep or delete. Do not accept a merge you did not look at.
