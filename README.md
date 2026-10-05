# ax

4 October 2026. People talk. The agent saves. The next chat already knows.

ax keeps one copy of how you work on a product and writes the short files Cursor and Claude Code paste into every chat. You do not learn a command schedule.

The longer plan is in [docs/target.md](docs/target.md).

## What it feels like

You open a folder and talk about what you are making. The agent asks until that is concrete, then saves it. Architecture, toolchain, and rules work the same way: you say the choice, it is saved as a decision, and later chats already have it. If something hurts, you say so. If you measure something, you say the number and how you got it. If the same pain keeps showing up, the agent suggests a rule and you say yes or no.

Saving the vision, a decision, a measurement, or a pain note rewrites `AGENTS.md` and `CLAUDE.md` before the command returns. There is no second compile step for you.

Those two files stay short. They hold the vision, the few rules that must always be on, and one line per decision still in force. Full writeups stay on disk under `.ax/`. The agent opens a decision file when the work depends on it.

## Install

On a normal Windows PC, open [the latest Release](https://github.com/briceduke/ax/releases/latest), download `ax_*_windows_amd64.exe` (or the `.zip` and unzip it), and put it on your PATH so the command is `ax`. Open a new terminal and run `ax version`. Then:

```powershell
ax install
```

That writes one skill you invoke yourself:

- Cursor: `~/.cursor/skills/ax/SKILL.md`
- Claude Code: `~/.claude/skills/ax/SKILL.md`

Both set `disable-model-invocation: true`. Nothing suggests ax until you type `/ax`. A folder you never typed `/ax` in stays alone.

ARM laptop: `windows_arm64`. Mac: `darwin_arm64` or `darwin_amd64`. Linux: `linux_amd64` or `linux_arm64`.

After that, `ax upgrade` replaces the binary. Outside a product repo it only replaces the binary. Inside one, it also updates the project pin.

`go build -o ax.exe ./cmd/ax` is only for changing ax itself, in this repo.

## Start a project

Type `/ax` in an empty folder. The agent asks what you are making, who it is for, and whether the notes are the work or the code is the work, then creates the project.

If the folder already has code, `/ax` starts from what is there (`ax adopt`).

After `.ax` exists, ordinary chat uses `AGENTS.md` or `CLAUDE.md`. You do not type `/ax` again for ordinary work.

`ax init --name` and `--intent` stay for tests and scripts. `--notes record` is for when the notes themselves are the work.

A software repo gets two ax files at the root: `AGENTS.md` and `CLAUDE.md`. Everything else ax stores is under `.ax/`. Editor hooks stay in `.cursor/` and `.claude/`.

When the notes are the work (hardware, research), the same vision, decisions, measurements, and pain notes also appear in `record/`. Reminders, checks, and packs stay in `.ax/`. Ask later to hide or show the notes; that moves the same files.

The tool finds a project by walking up until it sees `.ax/ax.yaml`.

## Day to day

Describe the task in chat. The agent reads the root files and saves when you make a choice, take a measurement, or hit friction. You approve or reject the diff.

You do not edit `AGENTS.md` or `CLAUDE.md`. Those are copies. If you edit them by hand, `ax check` fails on purpose.

The rest of the harness is offered in chat. You say yes or no. The commands remain so the agent has something to call:

- A check failed: the agent says so. Hooks already run checks after edits.
- You want a rule to stick: the agent writes the decision and a check that points at it.
- The same pain shows up again: the agent suggests a rule, or a short scored task. You say yes or no.
- A reminder is no longer needed: the agent asks to drop it.
- The next project should work the same way: the agent copies the useful part.
- A new editor: a new target file is the shell. `.ax` does not move. The root files are rewritten.
- Another machine: the environment is checked without you asking. The agent says what is wrong.
- Newer ax: the agent says what would change. You say yes or no.

Do not edit an old decision. Write a new one. If it replaces an older choice, the agent passes `--supersedes`.

## Releases

A conventional `feat` or `fix` merged to `main` releases itself. No click for the normal path. Commits must start with `feat`, `fix`, or `docs` (`feat!:` or a `BREAKING CHANGE:` footer for breaking). `docs` does not bump. `feat` is minor. `fix` is patch. On 0.x, breaking is minor; 1.0 is a later, conscious choice. Agents must use those prefixes — do not use `chore` for user-facing work.

release-please opens a version PR that bumps `VERSION` and `internal/version/version.go` together. After CI is green, a workflow squash-merges only that PR, tags `v*`, and GoReleaser attaches zip/tar files plus a raw `ax` / `ax.exe` for each OS. `ax upgrade` clones git for the project pin and downloads the matching Release to replace the running `ax`. `--skip-binary` leaves the executable alone.

## What lives where

| Path | Meaning |
| --- | --- |
| `AGENTS.md`, `CLAUDE.md` | Short copies for the next chat. Vision, always-on rules, one line per decision in force. |
| `.ax/ax.yaml` | Which editors, which ax version, which packs, and whether notes are also in `record/`. |
| `.ax/intent.md` | What this product is. |
| `.ax/decisions/` | Choices. One file each. Never edited. |
| `.ax/observations/` | Dated facts and how you got them. Eval scores land here too. |
| `.ax/friction/` | One-line “this hurt.” |
| `.ax/capabilities/` | Repeatable jobs you want the agent to keep doing. |
| `.ax/scaffolds/` | Temporary “don’t forget this” notes. Each one says when to delete it. |
| `.ax/evals/` | Short scored tasks. |
| `.ax/checks.yaml` | Commands or built-in tests that must stay true. |
| `.ax/tools.yaml` | Optional local MCP servers. Compile copies them into editor `mcp.json` files. |
| `.ax/packs/` | Shared bundles copied in from another project. |
| `.ax/manifest.yaml` | Whether generated editor files still match. |
| `record/` | Visible copies of vision, decisions, measurements, and pain notes when the notes are the work. |

## Command reference

For people working on ax itself. Ordinary product work happens in chat.

```
ax log friction "one line"
ax log observation "what you found" --method "how you found it"
ax log decision "short title" [--supersedes older-id]
ax check [--tier fast|full|slow] [--all]
ax compile [--target cursor] [--without scaffold-id]
ax eval [--without scaffold-id] [--runs n]
ax retro
ax ablate
ax propose-upstream [--friction id] [--layer machinery|schema|target|pack] [--change text] [--helps-others] [--submit]
ax upgrade [--from dir] [--submit] [--skip-binary]
ax pack add <dir>
ax pack extract <dir> --id <id> [--file path] [--replace old=PARAM]
ax pack bootstrap <discipline>
ax init [--name <name>] [--intent <text>] [--targets cursor,claude-code] [--notes record] [--interview]
ax adopt
ax notes show|hide
ax install [--home <dir>]
ax doctor [--require-container]
ax version
```

`ax log decision` reads markdown on stdin: `## Context`, `## Options`, `## Choice`, `## Why`. No pipe writes those headings empty.

`ax init --name` and `--intent` are for tests and scripts. `/ax` is how a person starts. `ax install --home` is how tests write skills without touching a real user home directory.

A save already rewrites the root files. `ax compile` is for hand-edits under `.ax/` and for adding an editor target.

`ax doctor` checks `.ax/ax.yaml`, runs `ax check`, and confirms `ax` is on PATH. If Docker is installed and the project has a `Dockerfile`, doctor builds the image and runs `ax check` inside it. Missing Docker prints `container not verified` and still passes. Pass `--require-container` to fail instead.

You still approve every change. ax does not merge GitHub PRs for you. It only talks to a model or to GitHub when a CLI is already on your PATH and you asked (`ax eval` with `agent`/`cursor`/`claude`, `AX_COMPILE_AGENT=1`, or `--submit` with `gh`).
