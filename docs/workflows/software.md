# Using ax on a new software project

4 October 2026. This is how a first week is supposed to feel. You talk. The agent saves. The next chat already knows.

The product is **spend**. A TypeScript CLI. You drop bank CSVs in `inbox/`. It prints a monthly report: total, and totals by category. One person. Evenings. Sunday night there is no repo. There is an empty folder.

This is a story, not a spec. The bank files are made up.

## 1. Empty folder

Download `ax_*_windows_amd64.exe` from the latest GitHub Release and put it on your PATH as `ax`. Open a new terminal, run `ax version`, then `ax install`. After that, `ax upgrade` replaces the binary. `go build` is only for changing ax itself.

```powershell
mkdir spend
cd spend
```

Open the folder in Cursor. Type `/ax`. The agent asks. You answer. It creates the project. You do not create folders by hand. Values and Taste stay Correctness and Short files unless you change them later.

```
What are you making? A CLI that reads bank CSVs from inbox/ and prints a monthly spend report.
Who is it for? One person who exports a checking account once a month.
Are the notes the work, or is the code the work? The code is the work.
```

The agent may also ask disciplines, constraints, team, and process weight. Then a pack offer. A pack is a reusable how-we-work bundle from other products in the same discipline. Updates later arrive as a merge you can refuse.

If a TypeScript/Node pack exists, the agent asks whether to add it. If not, it asks whether to bootstrap an empty TypeScript starter. Either way, process stays thin until something fails.

`AGENTS.md` and `CLAUDE.md` are already written. The next chat has the vision. `ax check` should pass. You have not written app code.

What is on disk:

| Path | What it is |
| --- | --- |
| `AGENTS.md`, `CLAUDE.md` | Short copies for the next chat. Do not edit them. |
| `.ax/intent.md` | What spend is. |
| `.ax/capabilities/` | Repeatable jobs. Init put four files here. |
| `.ax/scaffolds/` | Temporary model-weakness notes. Empty. |
| `.ax/evals/` | Short scored tasks for later. Empty. |
| `.ax/checks.yaml` | Built-in tests: file shape, links that exist, generated files still match. |
| `.ax/decisions/` | Choices. Init wrote adopt ax, plus toolchain/process if the interview ran. |
| `.ax/observations/` | Dated facts. Empty. |
| `.ax/friction/` | One-line “this hurt.” Empty. |
| `.ax/ax.yaml` | Which editors, which ax version, which packs. |
| `.cursor/`, `.claude/` | Hooks and skills. |

There is no `record/` folder. The code is the work. If this had been a hardware notebook, the same vision and decisions would also show up in `record/`.

You do not type `/ax` again for ordinary work.

## 2. First toolchain choice

You still have no parser. You need a runtime.

In Cursor, say that you want bun and TypeScript. The agent records that choice. You can type `/record-decision` if you want that skill immediately.

The agent writes `.ax/decisions/2026-10-04-use-bun-and-typescript.md` and fills the four sections. Do not leave them blank. That save rewrites `AGENTS.md` with one new line. There is no compile step for you.

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

You talk to Cursor as usual. You do not paste the intent. You do not remind it to log decisions. That is already in `AGENTS.md`. You still say the task for today, in plain words.

## 3. The first real failure

You want Chase checking CSVs to parse. In Cursor:

> Parse Chase checking CSVs in `inbox/`. Print month, total spent, and totals by category. Categories come from `categories.yaml`. Do not add packages.

The agent writes a parser. It adds `lodash` to split a line on commas. You did not ask.

The agent logs the hurt. One line. That is the whole file: `the agent added lodash to split CSV columns`.

Then, in chat:

> Take lodash out. Split the CSV in this repo. Do not add packages.

You do not want this again. You also do not want a merge that was never tested. Say those two rules in chat. The agent records both.

The agent fills Context, Options, Choice, Why on each. For the first: keep the built-in CSV tools, or add a package only after a person says yes. For the second: a report that was never run against a known file is not done.

A rule a machine can test should not stay a paragraph. In the same change, the agent adds checks to `.ax/checks.yaml`. Point `enforces:` at the decision each check is keeping honest. When a check fails, ax prints that decision’s Why. Hooks already run checks after edits. If one fails, the agent says so.

```yaml
- id: unit-tests
  run: bun test
  inputs: [src/**, testdata/**]
  tier: full
  enforces: .ax/decisions/2026-10-07-tests-must-pass-before-merge
- id: allowed-packages
  run: bun run checks/allowed-packages.ts
  inputs: [package.json, bun.lock, checks/allowed-packages.ts]
  tier: fast
  enforces: .ax/decisions/2026-10-07-no-extra-npm-packages-without-asking
```

`run` is a program and its arguments. No pipes or `&&`.

The agent writes a tiny `checks/allowed-packages.ts` that fails if `package.json` lists a name you did not allow, plus one fixture CSV and a test that checks the September totals.

You can also have the agent add a how-we-work note so it sees the rule before it types. One file in `.ax/capabilities/`:

```markdown
---
id: ask-before-deps
kind: capability
when: the work seems to need a new npm package
---

Stop. Name the package and why. Do not edit package.json until a person says yes.
```

No `hints` line. That text is an always-on rule in `AGENTS.md`. The agent is supposed to do it on its own.

If `bun test` fails, ax fails, and it points at “tests must pass before merge.” Fix the test in chat, not the generated files.

Spend has two extra rules. Both came from a failure you watched. Nothing else was added “in case.”

## 4. What Cursor is reading

You still do not edit `AGENTS.md` or `CLAUDE.md`. Those are copies. Saving a decision already rewrites them. If you change `.ax/intent.md` or add a how-we-work note by hand, the agent refreshes. If you edit `AGENTS.md` by hand, `ax check` fails on purpose.

| File | What it is |
| --- | --- |
| `AGENTS.md` | Vision, always-on rules, one line per decision in force. Cursor reads this at the start of a chat. |
| `.cursor/hooks.json` | After a file save, Cursor runs `ax check --tier fast`. |
| `.cursor/skills/{id}/SKILL.md` | Notes marked `hints: [user-invocable]`. |
| `.cursor/agents/` | Only if a note is marked `hints: [isolation]`. |

Claude Code gets the same content in `CLAUDE.md`, plus its own hooks and skills.

## 5. Repeated pain

Later the agent adds `date-fns` to parse a posted date. Same class of failure. The agent logs it: `the agent added date-fns to parse CSV dates`.

In chat, take the package out. The check already fails if it stays. The same pain showed up again, so the agent suggests a short scored task that checks whether it still follows the rule. You say yes or no.

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

Ask in chat when you want scores. It writes a dated fact under `.ax/observations/`. Something like `eval no-unasked-deps 2/3`. Two out of three still sneaks a package in. The check would have caught the merge. The how-we-work note is not wasted yet.

When something is a fact about the product, not a choice and not a score, say it in chat. The agent saves the measurement and how you got it.

## 6. Cut what you no longer need

Anyone can log friction in seconds. When you want a cut, say so in chat. The agent runs retro. That is a proposal, not a merge. Open it.

Every retro tries to delete something. If a reminder looks outdated, the agent asks to drop it. You say yes or no. If the scored task still fails without the note, keep it. The check stays either way. The paragraph does not get to live forever.

That is the week. Start empty. Type `/ax`. Answer a few questions. Say the task. The agent saves. You approve the diff. When it hurts twice, it offers a rule or a scored task. You never learn a Thursday command.

## 7. What the human actually types

**Once, to install ax**

Download `ax_*_windows_amd64.exe` from the latest GitHub Release. Put it on PATH as `ax`. In a new terminal:

```powershell
ax version
ax install
```

After that, `ax upgrade` replaces the binary.

**In an empty folder**

```powershell
mkdir spend
cd spend
```

Open Cursor. Type `/ax`. Answer the questions. Take the pack or bootstrap if offered.

**In chat**

> Use bun and TypeScript. Parse Chase checking CSVs in `inbox/`. Print month, total spent, and totals by category. Categories come from `categories.yaml`. Do not add packages.

The agent records the toolchain. The next chat already has it. You approve the diff.

**When it adds lodash**

> Take lodash out. Split the CSV in this repo. Do not add packages.

The agent logs the friction, records the two decisions, and adds the checks. You approve the diff.

**When it adds a package again**

In chat: take it out. Yes to a scored task if offered.

**When you want a cut**

In chat: run a retro. Read the proposal. Keep or delete. Do not accept a merge you did not look at.
