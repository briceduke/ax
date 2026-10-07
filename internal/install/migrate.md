---
name: ax-migrate
description: Update this repo's .ax files to the installed ax. Invoke with /ax-migrate. Not suggested automatically.
disable-model-invocation: true
---

You update this product repo so its files match the ax that is installed. The person typed /ax-migrate.

`ax upgrade` bumps the pin and recompiles. It does not rewrite decision files, and it does not replace a capability file that is already there. That is this skill.

1. If there is no `.ax/ax.yaml`, stop. They need `/ax` first.
2. Read the `ax:` pin in `.ax/ax.yaml`.
3. Run `ax version`. That is the installed version.
4. If the installed version is older than the pin, stop and say so. Do not change files.
5. Apply each section below whose version is newer than the pin and not newer than the installed version. Do them in order. "Nothing to do" means skip.
6. Do not edit `AGENTS.md` or `CLAUDE.md` by hand. Do not commit unless they ask. Do not log a decision for this migration.
7. Edit files under `.ax/` only. If `record/decisions` exists, make the same Choice edits there. Compile does not copy them.
8. Then run `ax compile` and `ax check --all`.
9. If the installed version is newer than the last section below and check still fails on file shape, stop. Say this skill does not cover that version yet.

## 0.1.0 through 0.5.0

Nothing to do.

## 0.6.0

Also do this when `ax check` says a choice is empty, is more than one sentence, or is over 200 characters, even if the pin and `ax version` both still say 0.5.0. That failure means this rule is already in the binary.

For every file in `.ax/decisions/`, including ones a later decision superseded:

- Read `## Choice`.
- If it is already one sentence and at most 200 characters, leave it.
- Otherwise replace `## Choice` with one sentence that states the decision. Move the extra sentences into `## Why`. Do not drop facts. Do not write a new decision. Do not pass `--supersedes`. The choice is the same. Only the line that lands in `AGENTS.md` has to be one sentence.

Then fix the two capability files ax originally wrote, and only those sentences.

In `.ax/capabilities/own-the-harness.md`, if you still see "Open a decision file when the work depends on it", replace that paragraph with:

```
AGENTS.md and CLAUDE.md stay short: the vision, the few rules that must always be on, and one sentence per decision still in force. The sentence is the decision. The path beside it is the file. Open that file only when you need more than the sentence. Do not open every decision file. Do not log a decision for something the vision already says, or for a task you are about to do. Do not edit AGENTS.md or CLAUDE.md by hand.
```

Also change the `ax log decision` bullet so it says `## Choice` is one sentence.

In `.ax/capabilities/record-decision.md`, if it does not already say "Choice is one sentence", add this after the first paragraph:

```
Log a decision only when a later chat would do the wrong thing without it. Skip it when the vision already says it, or when it is a task you are about to do. Every decision adds a line to AGENTS.md, and every chat loads that file.
```

On the `ax log decision` step, state that `## Choice` is one sentence, at most 200 characters, and that this sentence plus the file path is the line in `AGENTS.md`. ax rejects an empty choice, a second sentence, or a longer line.

Replace "The root files get one line. The full writeup stays under `.ax/decisions/`." with "Put detail in Context, Options, and Why. The root files get the Choice sentence only."

Leave any other project-specific lines in those two files alone.
