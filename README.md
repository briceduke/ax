# ax

4 October 2026. A small command-line tool that keeps one copy of “how we work on this product” and writes the files Cursor and Claude Code read.

The longer plan, and what is not built yet, is in [docs/target.md](docs/target.md).

## What you can do today

- Write down what you are building.
- Write down decisions, measurements, and things that went wrong.
- Turn that into `AGENTS.md` (Cursor) and `CLAUDE.md` (Claude Code).
- Run a few checks that fail if those files are out of date or the log is malformed.

You cannot yet start a project with one command, reuse a pack from another project, or have ax improve itself on a schedule.

## Install

You need Go. From this repo:

```powershell
go build -o ax.exe ./cmd/ax
```

Put `ax.exe` on your PATH and name it so the command is `ax`. After you set up a project, Cursor and Claude will run `ax check --tier fast` after edits.

## Add ax to a product repo

There is no `ax init` yet. Do this once.

**1. Tell ax which tools you use.** Create `ax.yaml` at the repo root:

```yaml
ax: 0.1.0
targets: [cursor, claude-code]
packs: []
```

**2. Ignore ax’s cache.** Add `.ax/` to `.gitignore`. Optional but useful: in `.gitattributes`, put `* text=auto eol=lf` so Windows and Linux hash files the same way.

**3. Say what you are building.** Create `core/intent.md`. Keep it under a page. Cover:

- what it is
- who it is for
- what you value
- what “good” looks like
- how much process you will tolerate

**4. Make empty folders**

```
core/capabilities
core/scaffolds
log/decisions
log/observations
log/friction
```

**5. Turn on the built-in checks.** Copy [testdata/fixtures/software/core/checks.yaml](testdata/fixtures/software/core/checks.yaml) to `core/checks.yaml`. After step 6, change every `enforces:` path to the decision file you just wrote.

**6. Log that you started using ax**

```powershell
ax log decision "adopt ax for harness management"
```

That prints a file path. Open it and fill in Context, Options, Choice, and Why.

**7. Generate the editor files and see that checks pass**

```powershell
ax compile
ax check
```

Run these from anywhere inside the repo. ax walks up until it finds `ax.yaml`.

## Day to day

You edit the files under `core/` and `log/`. You do not edit `AGENTS.md` or `CLAUDE.md`. Those are copies. If you change `core/intent.md` or add a how-we-work note, run `ax compile` again. If you edit `AGENTS.md` by hand, `ax check` fails on purpose.

**Something went wrong or felt heavy**

```powershell
ax log friction "the agent guessed pin names instead of reading the datasheet"
```

One line. That is the whole file.

**You measured something**

```powershell
ax log observation "4.1 mA while recording" --method "ammeter on the dev board"
```

**You made a real choice**

```powershell
ax log decision "detent twist is the power switch"
```

Fill in the four sections. Do not go back and edit an old decision. Write a new one. If it replaces an old one:

```powershell
ax log decision "use a slide switch" --supersedes 2026-10-04-detent-twist-is-the-power-switch
```

**Then**

```powershell
ax compile
ax check
```

## The folders, in plain language

| You write | Meaning |
| --- | --- |
| `core/intent.md` | What this product is. The only long note that lasts. |
| `core/capabilities/` | Repeatable jobs you want the agent to do, even when models get smarter. Example: “when we make a choice, write it down.” |
| `core/scaffolds/` | Temporary “don’t forget this” notes for a weakness the model has today. Each one must say when you will delete it. |
| `core/checks.yaml` | Commands or built-in tests that must stay true. |
| `log/decisions/` | Choices. One file each. Never edited. |
| `log/observations/` | Dated facts and how you got them. |
| `log/friction/` | One-line “this hurt.” |

`ax compile` writes `AGENTS.md`, `CLAUDE.md`, and a few hook files so the editor runs `ax check` after you save. It also writes `.generated/manifest.yaml` so it can tell if those files still match what you wrote.

A second `ax compile` with no changes does nothing.

## How-we-work notes (capabilities)

One markdown file per job. The top matter has to look like this:

```markdown
---
id: record-decision
kind: capability
when: a choice between real alternatives gets made
hints: [user-invocable]
---

Write a dated entry: context, options considered, choice, why.
```

- No `hints` — the text is copied into `AGENTS.md` / `CLAUDE.md` and the agent is supposed to do it on its own.
- `hints: [user-invocable]` — you get a Cursor command and a Claude skill you can type, like `/record-decision`.
- `hints: [isolation]` — a separate agent file, for work you want in its own session.

Extra fields in that top matter are errors. ax will not guess.

## Temporary notes (scaffolds)

```markdown
---
id: verify-pinouts
kind: scaffold
compensates: the model invents pin names that are not in the datasheet
added: 2026-10-02 (friction/2026-10-02-wrong-pin)
retire_when: we can drop this after the model quotes the datasheet on its own
---

Before you assign a pin, quote the datasheet row it comes from.
```

`added` must point at a friction file that exists.

## Your own checks

In `core/checks.yaml` you can add a command:

```yaml
- id: board-fits-case
  run: python checks/fit.py
  inputs: [interfaces/envelope.yaml, hardware/case/**, pcb/**]
  tier: full
  enforces: log/decisions/2026-10-04-envelope-v1
```

`run` is a program and its arguments. No pipes or `&&`. If you need a pipeline, put it in a script.

`enforces` is the decision this check is keeping honest. When the check fails, ax prints that decision’s Why.

`tier` is how often it runs:

- `fast` — seconds. Runs after every edit, and also when you run `ax check`.
- `full` — minutes. Default for `ax check`. Includes fast.
- `slow` — hours. Only if you pass `--tier slow`.

ax remembers the last pass. If the check and its input files did not change, it skips. `ax check --all` runs everything anyway.

Built in:

- files in `core/` and `log/` are shaped right
- links between files actually exist
- generated editor files still match what you last compiled

## Log size

Decisions: at most 60 lines, and they must have `## Context`, `## Options`, `## Choice`, `## Why`. Observations: at most 30 lines. Friction: one line.

## Commands that exist

```
ax log friction "one line"
ax log observation "what you found" --method "how you found it"
ax log decision "short title" [--supersedes older-id]
ax check [--tier fast|full|slow] [--all]
ax compile [--target cursor] [--target claude-code]
```

`--without` on compile is for later experiments. You can ignore it.
