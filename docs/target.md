# Target state

4 October 2026. This is what ax is meant to become. Each section says what is implemented in 0.1.0 and what is not.

Status: **done** ships in this repo. **partial** exists but is thinner than the target. **not yet** is specified only.

The how-to for what works today is in [README.md](../README.md).

## What ax is

A portable harness for people and agents building a product. The harness itself should get smaller and better over time. It runs from a clone of the product repo, in Cursor, Claude Code, or whatever comes next.

Two repos: this repo is the machinery (schema, compiler, check runner, later evals and packs). Each product repo holds its own core, log, design sources, generated adapters, and vendored packs, and pins an ax version.

**done** — ax repo exists; a product repo can pin `0.1.0`, hold a core and log, and compile Cursor plus Claude Code adapters.

**not yet** — packs, versioned releases, upgrade PRs, `ax doctor`, identical container from clone.

## Principles

These are the bar. The skeleton follows them; they are not themselves executable.

1. Store what cannot be derived; generate the rest.
2. Rules become checks.
3. Durable core, disposable adapters.
4. Every crutch has an expiry.
5. Change needs evidence.
6. Add on failure, remove on schedule.
7. Humans read it: short, plain, dated.
8. Runs anywhere: clone plus container.

## Layers in a product repo

| Layer | Contents | Lifespan | Status |
| --- | --- | --- | --- |
| Durable core | intent, capabilities, checks, evals, log, design sources | Years. Changes through decisions. | **partial** — intent, capabilities, checks, log. No evals. |
| Scaffolds | Workarounds for current model weaknesses | Months. Each has a retire condition. | **done** — schema and compile include them. No ablation that retires them. |
| Adapters | `AGENTS.md`, `CLAUDE.md`, skills, commands, hooks, MCP | Disposable. Regenerated. | **done** — Cursor and Claude Code. |

Deterministic work (validation, check running, hashing) is code. Judgment work (compile to a target, retros, teaching, extracting packs) is agents, with code validating the result.

**partial** — compile is a deterministic assembler with a `Writer` seam. The agent compiler is not wired.

## Project core

```
project/
  ax.yaml              # pin, targets, packs
  core/
    intent.md
    capabilities/
    scaffolds/
    checks.yaml
    evals/
  log/
    decisions/
    observations/
    friction/
  packs/
  .generated/
```

**done** — `ax.yaml` (`ax`, `targets`, `packs: []`), `intent.md`, capabilities, scaffolds, `checks.yaml`, the three log kinds, `.generated/manifest.yaml`.

**not yet** — `core/evals/`, `packs/`, tools in the core, git-based immutability of decisions.

Log rules that already hold: one file per entry; ids are `date-slug`, not numbers; a decision is never edited; a later decision may `supersedes` it.

## Compile

`ax compile` turns the core into each tool's native files. Target specs in this repo say where things land. A new tool is one new spec.

**done** — `targets/cursor.md`, `targets/claude-code.md`; hash of intent, capabilities, scaffolds, checks, and target specs; skip when unchanged; validator (required files, frontmatter, every capability lands); markdown `do not edit` headers; JSON hashes in the manifest; `--target`; `--without <scaffold-id>` into a throwaway worktree; staleness check.

**not yet** — agent writer; a tools schema that fills MCP; more targets.

## Checks

`ax check` runs the registry. Failures name the decision they enforce.

**done** — tiers `fast` / `full` / `slow` (cumulative; default `full`); hash cache in `.ax/`; `--all`; `run` as argv, no shell; builtins `ax-format`, `ax-consistency`, `ax-staleness`; failure output includes the enforced Why.

**not yet** — after-edit hooks that work if `ax` is missing; judgment (LLM) checks; registering an existing project's tests automatically; hardware-in-the-loop skip tags.

## Evals and ablation

Evals are regression tests for the harness. Ablation compiles without a scaffold and compares scores. Health is harness size against mean eval score: size should fall while scores hold.

**not yet** — eval format, `ax eval`, headless agent runs, `ax ablate`. `--without` is only the compile half.

## Self-improvement

Friction is cheap to log. A retro clusters it, proposes core changes (including at least one deletion), runs evals, and opens a PR. A human merges.

**partial** — `ax log friction` works.

**not yet** — `ax retro`, scheduled retros, agents logging friction unprompted, evals as a gate on harness PRs.

## Packs, upstream, onboarding

Packs are vendored discipline bundles. Improvements flow up as sanitized issues or PRs and back down as versioned upgrades. A project starts thin via `ax init` or `ax adopt`.

**not yet** — all of it: `ax pack`, `ax propose-upstream`, releases, `ax upgrade`, `ax init`, `ax adopt`, `ax doctor`, fixtures CI that compiles every target.

Until `ax init` exists, hand-init as in the README.

## CLI

| Command | Target | Status |
| --- | --- | --- |
| `ax log decision\|observation\|friction` | Write a log entry | **done** |
| `ax check [--tier] [--all]` | Run checks whose inputs changed | **done** |
| `ax compile [--target] [--without]` | Core to adapters | **partial** — assembler, not an agent |
| `ax init` | Interview, write a thin core, offer packs | **not yet** |
| `ax adopt` | Onboard an existing repo from history | **not yet** |
| `ax eval` | Headless evals in the container | **not yet** |
| `ax ablate` | Score scaffolds with and without | **not yet** |
| `ax retro` | Cluster friction, propose core changes, open PR | **not yet** |
| `ax pack add\|extract\|bootstrap` | Install, extract, or bootstrap a discipline | **not yet** |
| `ax propose-upstream` | File sanitized issues or PRs to ax | **not yet** |
| `ax upgrade` | Move to a new ax release via PR | **not yet** |
| `ax doctor` | Fresh-container portability check | **not yet** |

## Build order

| Phase | Exit | Status |
| --- | --- | --- |
| 1 Skeleton | `ax check` passes on real projects | **done** |
| 2 Compile | One core yields working Cursor and Claude Code adapters | **done** |
| 3 Knowledge | Record-decision capability; an agent answers across disciplines from the log | **not yet** |
| 4 Evals | Scores in the log | **not yet** |
| 5 Retro | One accepted change and one accepted deletion | **not yet** |
| 6 Ablation | A scaffold retired or confirmed with numbers | **not yet** |
| 7 Upstream | Friction in one project becomes an upgrade PR in another | **not yet** |
| 8 Packs and onboarding | A third project starts from `ax init` | **not yet** |
