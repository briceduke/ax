# ax

4 October 2026. A small command-line tool that keeps one copy of “how we work on this product” and writes the files Cursor and Claude Code read.

The longer plan is in [docs/target.md](docs/target.md).

## What you can do today

- Start a project with `ax init`, or add ax to an existing repo with `ax adopt`.
- Write down what you are building.
- Write down decisions, measurements, and things that went wrong.
- Turn that into `AGENTS.md` (Cursor) and `CLAUDE.md` (Claude Code). The agent runs `ax` from those files. Slash commands such as `/record-decision` still exist after compile if you want a skill immediately.
- Run checks that fail if those files are out of date or the log is malformed.
- Run short scored tasks (`ax eval`), propose cuts from friction (`ax retro`), and see whether a temporary note is still needed (`ax ablate`).
- Copy a shared bundle in (`ax pack add`) or write a cleaned-up fix locally (`ax propose-upstream`). Nothing is merged for you.

You still approve every change. ax does not merge GitHub PRs for you. It only talks to a model or to GitHub when a CLI is already on your PATH and you asked (`ax eval` with `agent`/`cursor`/`claude`, `AX_COMPILE_AGENT=1`, or `--submit` with `gh`).

## Install

Download the binary for your OS from the [latest GitHub Release](https://github.com/briceduke/ax/releases/latest). On Windows that is `ax_*_windows_amd64.exe`. Put it on your PATH as `ax`. Open a new terminal and run `ax version`. You should see the same version string as that release.

After that, `ax upgrade` replaces the binary. Outside a product repo it only replaces the binary. Inside one, it also updates the project pin.

`go build -o ax.exe ./cmd/ax` is only for changing ax itself, in this repo.

After you set up a project, Cursor and Claude will run `ax check --tier fast` after edits. Releases also ship Linux and macOS binaries.

## Releases

A conventional `feat` or `fix` merged to `main` releases itself. No click for the normal path. Commits must start with `feat`, `fix`, or `docs` (`feat!:` or a `BREAKING CHANGE:` footer for breaking). `docs` does not bump. `feat` is minor. `fix` is patch. On 0.x, breaking is minor; 1.0 is a later, conscious choice. Agents must use those prefixes — do not use `chore` for user-facing work.

release-please opens a version PR that bumps `VERSION` and `internal/version/version.go` together. After CI is green, a workflow squash-merges only that PR, tags `v*`, and GoReleaser attaches the binaries. `ax upgrade` still clones this git repo and reads `VERSION`; it does not read Release zips.

## Start a new project

After Install, `ax version` should print. Then:

```powershell
ax init --name widget --intent "A small recorder for field notes."
```

That writes the folders below, logs that you adopted ax, generates the editor files, and leaves you in a state where `ax check` passes.

If stdin is a terminal, or you pass `--interview`, ax asks the project name, what you are building, for whom, disciplines, constraints, who is on the team, and process weight. Answers go into `core/intent.md` and the first toolchain/process decisions. Values and Taste stay Correctness and Short files. Tests and scripts should pass `--name` and `--intent` and omit `--interview`.

To add ax to a repo that already has code:

```powershell
ax adopt
```

That writes a short intent stub, copies the built-in harness notes (`own-the-harness`, `log-friction`, `sync-harness`, `record-decision`), and if it finds `go.mod`, registers `go test ./...` as a check. Inferred decisions are marked `reconstructed: true`.

`ax doctor` checks `ax.yaml`, runs `ax check`, and confirms `ax` is on your PATH. If Docker is installed and the project has a `Dockerfile`, doctor builds the image and runs `ax check` inside it. Missing Docker prints `container not verified` and still passes. Pass `--require-container` to fail instead. This repo ships a `Dockerfile` and a thin `.devcontainer` so a clone can use the same Go image.

## Add ax by hand

You can still do this without `ax init`. Create `ax.yaml` at the repo root:

```yaml
ax: 0.1.0
targets: [cursor, claude-code]
packs: []
```

Add `.ax/` to `.gitignore`. Optional but useful: in `.gitattributes`, put `* text=auto eol=lf` so Windows and Linux hash files the same way.

Create `core/intent.md` (under a page). Cover what is being built, for whom, values, taste, and process weight.

Make empty folders:

```
core/capabilities
core/scaffolds
core/evals
log/decisions
log/observations
log/friction
```

Copy [testdata/fixtures/software/core/checks.yaml](testdata/fixtures/software/core/checks.yaml) to `core/checks.yaml`. After you log the first decision, point every `enforces:` path at that file.

```powershell
ax log decision "adopt ax for harness management"
ax compile
ax check
```

Run these from anywhere inside the repo. ax walks up until it finds `ax.yaml`.

## Day to day

Describe the task in chat. The agent reads `AGENTS.md`, then runs `ax log`, `ax compile`, and `ax check` when those apply. You approve or reject the diff.

Slash commands such as `/record-decision` still exist after compile. Type one if you want that skill run immediately. You do not have to.

You do not edit `AGENTS.md` or `CLAUDE.md`. Those are copies. If you edit `AGENTS.md` by hand, `ax check` fails on purpose.

**Something went wrong or felt heavy**

The agent runs `ax log friction` with one line. That is the whole file.

```powershell
ax log friction "the agent guessed pin names instead of reading the datasheet"
```

**You measured something**

```powershell
ax log observation "4.1 mA while recording" --method "ammeter on the dev board"
```

**You made a real choice**

The agent runs `ax log decision` and fills the four sections. Do not go back and edit an old decision. Write a new one. If it replaces an old one:

```powershell
ax log decision "use a slide switch" --supersedes 2026-10-04-detent-twist-is-the-power-switch
```

Ask whether a machine can test the choice. If yes, add a check in the same change.

**Then**

The agent runs:

```powershell
ax compile
ax check
```

Ask in chat when you want `ax eval` or `ax retro`. Leave a retro proposal unmerged until you accept it.

## The folders, in plain language

| You write | Meaning |
| --- | --- |
| `core/intent.md` | What this product is. The only long note that lasts. |
| `core/capabilities/` | Repeatable jobs you want the agent to do, even when models get smarter. Example: “when we make a choice, write it down.” |
| `core/scaffolds/` | Temporary “don’t forget this” notes for a weakness the model has today. Each one must say when you will delete it. |
| `core/evals/` | Short scored tasks. A script or a rubric says pass or fail. |
| `core/tools.yaml` | Optional local MCP servers (`name`, `command`, `args`). Compile copies them into editor `mcp.json` files. |
| `core/checks.yaml` | Commands or built-in tests that must stay true. |
| `log/decisions/` | Choices. One file each. Never edited. |
| `log/observations/` | Dated facts and how you got them. Eval scores land here too. |
| `log/friction/` | One-line “this hurt.” |
| `packs/` | Shared bundles copied in from another project. |

`ax compile` writes `AGENTS.md`, `CLAUDE.md`, skill files under `.cursor/skills/` and `.claude/skills/`, and a few hook files so the editor runs `ax check` after you save. It also writes `.generated/manifest.yaml` so it can tell if those files still match what you wrote. Optional `core/tools.yaml` (name, command, args) is copied into `.cursor/mcp.json` and `.mcp.json`. No product MCP servers are invented.

Default compile is a fixed assembler. Set `AX_COMPILE_AGENT=1` and have an agent CLI on PATH if you want an agent to write the adapters; ax still validates the result. CI and the default path stay assembler-only.

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
- `hints: [user-invocable]` — compile writes a Cursor skill at `.cursor/skills/{id}/SKILL.md` and a Claude skill at `.claude/skills/{id}/SKILL.md`. You can type `/{id}` if you want that skill now.
- `hints: [isolation]` — a separate agent file, for work you want in its own session.

Extra fields in that top matter are errors. ax will not guess.

Init writes four built-ins. `own-the-harness` has no hints, so it lands in `AGENTS.md`. `log-friction`, `sync-harness`, and `record-decision` are user-invocable skills. Compile includes them even if the project file is missing.

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

## Harness tests (evals)

Short markdown files under `core/evals/`:

```markdown
---
id: cross-discipline-impact
related: [capability/record-decision]
runs: 3
---
task: >
  Find the record-decision note from the product files.
grader:
  kind: script
  file: core/capabilities/record-decision.md
```

`ax eval` runs each grader and writes a dated observation (`method: eval`) with the score.

- If `agent`, `cursor`, or `claude` is on PATH, ax runs the eval task in the project directory first, then the grader scores what is on disk. Tests inject a fake runner and never call a paid API.
- Script graders: a named check passed, a file exists, or the first number in a file is in range. These still run when no agent CLI is present.
- Rubric graders: required substrings in `core/` and `log/` after an agent run. If no agent CLI is present, the eval is logged as skipped/unavailable. It does not pretend a model ran.

`ax eval --without verify-pinouts` compiles without that temporary note first, then scores.

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

- files in `core/` and `log/` are shaped right (including evals)
- links between files actually exist
- generated editor files still match what you last compiled

## Getting better over time

`ax retro` reads friction since the last retro, recent log notes, and the latest eval scores. It clusters problems, suggests an eval when the same pain repeats with no test, and **always proposes at least one deletion**. The proposal is markdown under `.ax/retro/`. You merge or reject it.

`ax ablate` compiles without each temporary note, runs the related evals with and without it, writes the scores, and proposes retirement when the score without the note is at least as good.

## Sharing

`ax pack add <dir>` copies a pack into `packs/` and merges the files it provides. `ax pack extract` copies selected local files into a pack directory and can replace product words with `{{PARAM}}`. `ax pack bootstrap pcb` writes a six-step empty starter: decide the toolchain, then add checks, evals, capabilities, and scaffolds only after something fails.

`ax propose-upstream` writes a cleaned issue and pull-request draft under `.ax/upstream/`. Absolute paths and anything listed in `.ax/private.txt` are stripped. It does not open GitHub unless you pass `--submit` and `gh` is on PATH. Even then it only files an issue. You still approve the change.

`ax upgrade` without `--from` clones the ax GitHub repo (injectable in tests) and reads `VERSION`. It does not use GitHub Release zips. `--from <dir>` still uses a local tree. It copies any new how-we-work notes, bumps the pin in `ax.yaml`, recompiles, runs checks, and writes `.ax/upgrade/report.md`. `--submit` files a GitHub issue if `gh` exists. There is no auto-merge.

## Log size

Decisions: at most 60 lines, and they must have `## Context`, `## Options`, `## Choice`, `## Why`. Observations: at most 30 lines. Friction: one line.

## Commands that exist

```
ax log friction "one line"
ax log observation "what you found" --method "how you found it"
ax log decision "short title" [--supersedes older-id]
ax check [--tier fast|full|slow] [--all]
ax compile [--target cursor] [--target claude-code] [--without scaffold-id]
ax eval [--without scaffold-id] [--runs n]
ax retro
ax ablate
ax propose-upstream [--friction id] [--layer machinery|schema|target|pack] [--change text] [--helps-others] [--submit]
ax upgrade [--from dir] [--submit]
ax pack add <dir>
ax pack extract <dir> --id <id> [--file path] [--replace old=PARAM]
ax pack bootstrap <discipline>
ax init [--name <name>] [--intent <text>] [--targets cursor,claude-code] [--interview]
ax adopt
ax doctor [--require-container]
ax version
```
