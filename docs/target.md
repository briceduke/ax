# Where ax is going

4 October 2026. How to use what exists today: [README.md](../README.md).

ax is supposed to stay small. You write a few facts about the product. The files your editor reads are generated. When a model gets stronger, you delete instructions you no longer need.

This file is the destination. Each part says **built**, **started**, or **not built**.

## The idea

People and agents should be able to clone a product repo and work the same way, in Cursor, Claude Code, or the next tool.

There are two repos:

- **This repo (ax)** — the program.
- **Each product repo** — a short “what we’re building,” a log, the actual design files, and generated editor files. It names the ax version it expects.

**Built:** the program can start a project, write a log, run checks and evals, generate Cursor and Claude Code files, propose retros and pack updates, and write local upgrade notes.

**Not built:** fetching a new ax version from the network, submitting GitHub issues for you, or “clone this and you get the same computer.”

## Rules we are aiming at

1. Write down decisions and facts once. Generate summaries. Don’t keep two copies of the same story.
2. If a rule matters and a machine can test it, make it a check, not a paragraph.
3. You edit the source. Editor files are throwaways.
4. Every workaround says why it exists and when to delete it.
5. Don’t change how the team works because it “would be nice.” Change it because something broke or a test said so.
6. Add process when you fail. Try to remove it when models improve.
7. Anything we keep on disk should be short and readable by a person.
8. Same repo, same environment, any machine.

## What lives in a product repo

Three kinds of files:

**Lasts for years.** What you’re building, how-we-work notes, checks, evals, the log, packs you copied in, and the real design (code, CAD, and so on). You change these when you decide to.

**Lasts for months.** Temporary “the model still gets this wrong” notes. Each one has a kill condition.

**Thrown away and regenerated.** `AGENTS.md`, `CLAUDE.md`, slash commands, hooks.

**Built:** the lasting files, the temporary notes, scored evals, Cursor / Claude Code generated files, and local pack/upstream/upgrade drafts.

**Started:** generating editor files is a fixed program, not an AI pass. The plan is that an agent writes them and the program only checks the result. Rubric evals use a substring stand-in for the same reason.

## Starting a project

**Built.** `ax init --name --intent` writes the minimum tree, the built-in record-decision note, the first decision, and compiles. `ax adopt` reads an existing repo, writes an intent stub, and can register `go test ./...` if it finds Go.

Interactive questions run only if stdin is a terminal.

## The log

One file per note, named with a date and a short slug, so two people can add files without colliding.

- A **decision** is a choice. You never edit it. A newer decision can say it replaces an older one. Adopted history can be marked `reconstructed: true`.
- An **observation** is a measurement or finding, plus how you got it. Eval scores use method `eval`.
- **Friction** is one line about what went wrong or felt heavy.

**Built.**

## Generating editor files

You run `ax compile`. It reads what you wrote and writes the files each editor wants. If nothing changed, it stops. If you edit a generated file, checks fail until you compile again from the source.

Supporting a new editor should mean adding one short description of where that editor keeps instructions, commands, hooks, and tools.

**Built:** Cursor and Claude Code, skip-if-unchanged, “don’t edit this” headers, the check that catches a stale copy, and a built-in `/record-decision` command.

**Not built:** an AI writing those files; filling in MCP tool configs from the project; any editor besides those two.

## Checks

`ax check` runs the tests listed in the project. A failure should point at the decision that made the rule, so you can see *why*, not only *what*.

**Built:** fast / full / slow, skip unchanged inputs, your own commands, and the built-in tests (shape of files including evals, links that exist, generated files match).

**Not built:** checks that need a physical board and get skipped when there isn’t one; an AI grading “is this readable?” (the rubric stand-in is substring match); a full auto-import of every existing test in a large repo.

## Tests of the harness, and deleting workarounds

A few short tasks under `core/evals/`. A script or a rubric scores them. `ax eval` writes the scores into the log. `ax ablate` generates the editor files without a temporary note and compares scores. If the score holds, it proposes deleting the note. A person still decides.

**Built:** eval format, script graders, rubric stand-in, ablation, and three hardware-fixture evals.

**Not built:** calling a model to attempt the task, or a scheduled ablation bot.

## Getting better over time

Anyone can log friction in seconds. `ax retro` reads recent friction, proposes a smaller or clearer set of instructions, and always tries to delete something. The proposal is local files. A person merges or rejects.

**Built:** the retro command and proposal shape check.

**Not built:** a calendar schedule that runs retro for you.

## Sharing across projects

If several products need the same discipline (say, PCB work), that how-we-work set should become a pack you copy in. Updates arrive as a merge you can refuse.

Fixes land in one product, get cleaned of private detail, and come back as files under `.ax/upstream/`. CI in this repo runs tests and compiles both fixtures.

**Built:** pack add/extract/bootstrap, local sanitized upstream drafts, `ax upgrade --from` a local directory, GitHub Actions for this repo.

**Not built:** publishing a pack to a dummy project gate automatically; fetching ax from the network; opening a real GitHub issue without an injected client.

## Same environment everywhere

`ax doctor` checks `ax.yaml`, `ax check`, and that `ax` is on PATH. If Docker is installed it says so. Missing Docker does not fail the command.

**Started:** doctor without requiring Docker.

**Not built:** a committed container that gives every clone the same computer.

## Commands

| Command | What it is for | Status |
| --- | --- | --- |
| `ax log …` | Write a decision, observation, or friction note | Built |
| `ax check` | Run the project’s tests | Built |
| `ax compile` | Write the editor files from what you authored | Built (no AI in the loop) |
| `ax init` | Start a new project | Built |
| `ax adopt` | Add ax to a repo that already exists | Built |
| `ax eval` | Run the harness tests and log scores | Built (no model calls) |
| `ax ablate` | See if a temporary note is still needed | Built |
| `ax retro` | Propose cuts and fixes from logged friction | Built (no auto-merge) |
| `ax pack …` | Copy or extract a shared discipline bundle | Built |
| `ax propose-upstream` | Write a cleaned-up fix locally | Built (GitHub submit is a seam) |
| `ax upgrade` | Move the project to a new ax version from a local dir | Built (no network fetch) |
| `ax doctor` | Prove the repo’s files and PATH are sane | Built (Docker optional) |

## Build order

| Phase | We will call it done when | Status |
| --- | --- | --- |
| 1. Skeleton | `ax check` passes on real product repos | Built |
| 2. Generate editor files | You and a teammate can use Cursor and Claude Code on the same repo | Built |
| 3. Knowledge | `/record-decision` exists after compile; a choice is written into the log | Built |
| 4. Harness tests | Scores show up in the log | Built |
| 5. Retro | One retro writes a proposal that includes a deletion | Built (human still merges) |
| 6. Drop workarounds | Ablation writes numbers and a keep-or-drop proposal | Built (human still deletes) |
| 7. Share upstream | A fix becomes local upstream files and CI runs on ax itself | Built (no auto GitHub) |
| 8. Packs and start | A third project starts with `ax init` only | Built |
