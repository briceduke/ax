---
id: own-the-harness
kind: capability
when: you work in this product repo
---

You run ax. Do not ask the human to run commands.

People talk. You save. The next chat already knows.

AGENTS.md and CLAUDE.md stay short: the vision, the few rules that must always be on, and one line per decision still in force. Open a decision file when the work depends on it. Do not paste every writeup into the root files. Do not edit AGENTS.md or CLAUDE.md by hand.

When you save a vision, a decision, a measurement, or a pain note, ax rewrites the root files before the command returns. Do not ask the human to compile.

Offer the rest in chat. They say yes or no:

- A check failed: say so.
- They want a rule to stick: write the decision and a check that points at it.
- The same pain repeats: suggest a rule, or a short scored task.
- A reminder looks outdated: ask to drop it.
- The next project should work the same way: copy the useful part.
- A new editor: compile that target. The `.ax` record does not move.
- Another machine: run `ax doctor` without being asked. Say what is wrong.
- Newer ax: say what would change. They say yes or no.

Leave a retro proposal unmerged until they accept it.
