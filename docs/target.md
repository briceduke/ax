# Where ax is going

4 October 2026. How to use what exists today: [README.md](../README.md).

ax is supposed to stay small. You write a few facts about the product. The files your editor reads are generated. When a model gets stronger, you delete instructions you no longer need.

This file is the destination. Each part says **built**, **started**, or **not built**.

## The idea

People and agents should be able to clone a product repo and work the same way, in Cursor, Claude Code, or the next tool.

There are two repos:

- **This repo (ax)** — the program.
- **Each product repo** — a short “what we’re building,” a log, the actual design files, and generated editor files. It names the ax version it expects.

**Built:** the program can write a log, run checks, and generate Cursor and Claude Code files from a product repo.

**Not built:** a one-command project start, shared bundles you can drop into a new repo, upgrades, or “clone this and you get the same computer.”

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

**Lasts for years.** What you’re building, how-we-work notes, checks, the log, and the real design (code, CAD, and so on). You change these when you decide to.

**Lasts for months.** Temporary “the model still gets this wrong” notes. Each one has a kill condition.

**Thrown away and regenerated.** `AGENTS.md`, `CLAUDE.md`, slash commands, hooks.

**Built:** the lasting files (except tests of the harness itself), the temporary notes, and Cursor / Claude Code generated files.

**Started:** generating those files is a fixed program, not an AI pass. The plan is that an agent writes them and the program only checks the result.

**Not built:** tests that score the harness, and reuse bundles from other projects.

## Starting a project

**Not built.** You create the folders by hand. See the README.

Later: `ax init` asks you what you’re building and writes the minimum. `ax adopt` reads an existing repo and proposes that minimum from history.

## The log

One file per note, named with a date and a short slug, so two people can add files without colliding.

- A **decision** is a choice. You never edit it. A newer decision can say it replaces an older one.
- An **observation** is a measurement or finding, plus how you got it.
- **Friction** is one line about what went wrong or felt heavy.

**Built.**

## Generating editor files

You run `ax compile`. It reads what you wrote and writes the files each editor wants. If nothing changed, it stops. If you edit a generated file, checks fail until you compile again from the source.

Supporting a new editor should mean adding one short description of where that editor keeps instructions, commands, hooks, and tools.

**Built:** Cursor and Claude Code, skip-if-unchanged, “don’t edit this” headers, and the check that catches a stale copy.

**Not built:** an AI writing those files; filling in MCP tool configs from the project; any editor besides those two.

## Checks

`ax check` runs the tests listed in the project. A failure should point at the decision that made the rule, so you can see *why*, not only *what*.

**Built:** fast / full / slow, skip unchanged inputs, your own commands, and the three built-in tests (shape of files, links that exist, generated files match).

**Not built:** checks that need a physical board and get skipped when there isn’t one; an AI grading “is this readable?”; auto-import of a repo’s existing tests.

## Tests of the harness, and deleting workarounds

The plan: a few short tasks that an agent tries, scored by a script or a rubric. When a new model ships, generate the editor files *without* a temporary note and see if the score holds. If it does, delete the note.

The healthy trend is fewer instruction files and the same or better scores.

**Not built.** Compile can already omit one temporary note (`--without`). Nothing scores the result.

## Getting better over time

Anyone can log friction in seconds. On a schedule, an agent reads recent friction, proposes a smaller or clearer set of instructions, runs the harness tests, and opens a pull request. A person merges or rejects. Every retro should try to delete something.

**Started:** you can log friction.

**Not built:** the retro command, the schedule, and using harness tests as the gate.

## Sharing across projects

If several products need the same discipline (say, PCB work), that how-we-work set should become a pack you copy in. Updates arrive as a merge you can refuse. A pack has to pass its tests in an empty dummy project first, so it doesn’t secretly depend on the product it came from.

Fixes land in one product, get cleaned of private detail, and come back to everyone as an upgrade.

**Not built.**

## Same environment everywhere

Clone the repo, open the container, get the same tools. Secrets stay in environment variables. `ax doctor` should prove a fresh container can run the checks.

**Not built.**

## Commands

| Command | What it is for | Status |
| --- | --- | --- |
| `ax log …` | Write a decision, observation, or friction note | Built |
| `ax check` | Run the project’s tests | Built |
| `ax compile` | Write the editor files from what you authored | Built (no AI in the loop) |
| `ax init` | Start a new project by answering a few questions | Not built |
| `ax adopt` | Add ax to a repo that already exists | Not built |
| `ax eval` | Run the harness tests | Not built |
| `ax ablate` | See if a temporary note is still needed | Not built |
| `ax retro` | Propose cuts and fixes from logged friction | Not built |
| `ax pack …` | Copy or extract a shared discipline bundle | Not built |
| `ax propose-upstream` | Send a cleaned-up fix back to ax | Not built |
| `ax upgrade` | Move the project to a new ax version | Not built |
| `ax doctor` | Prove the repo runs in a fresh container | Not built |

## Build order

| Phase | We will call it done when | Status |
| --- | --- | --- |
| 1. Skeleton | `ax check` passes on real product repos | Built |
| 2. Generate editor files | You and a teammate can use Cursor and Claude Code on the same repo | Built |
| 3. Knowledge | An agent in one discipline can answer a question about another from the log and the source files | Not built |
| 4. Harness tests | Scores show up in the log | Not built |
| 5. Retro | One retro lands a change and a deletion | Not built |
| 6. Drop workarounds | At least one temporary note is removed or kept, with numbers | Not built |
| 7. Share upstream | A fix in one product becomes an upgrade in another | Not built |
| 8. Packs and start | A third project starts with `ax init` only | Not built |
