---
id: record-decision
kind: capability
when: a choice between real alternatives gets made
hints: [user-invocable]
---

When you make a real choice, write it down in this repo.

1. Run `ax log decision "<short title>"`. Fill Context, Options, Choice, and Why. The command rewrites AGENTS.md and CLAUDE.md before it returns. Do not ask for a compile step.
2. Do not edit an old decision. Write a new one. If it replaces an older choice, pass `--supersedes <id>`.
3. Ask whether a machine can test the choice. If yes, add a check in `.ax/checks.yaml` in the same change and set `enforces:` to the new decision file.

Keep the entry short. The root files get one line. The full writeup stays under `.ax/decisions/`.
