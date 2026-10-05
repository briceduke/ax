---
id: cursor
instructions: AGENTS.md
invocable: .cursor/skills/{id}/SKILL.md
isolated: .cursor/agents/{id}.md
hooks: .cursor/hooks.json
hooks_format: cursor-hooks
mcp: .cursor/mcp.json
---

# Cursor

Where each core piece lands. The assembler uses only these paths.

- **Instructions** (intent + always-on rules): `AGENTS.md` at the project root (plain markdown; `.cursor/rules` is the structured alternative, not used here).
  Docs: https://cursor.com/docs/rules
- **User-invocable capabilities**: `.cursor/skills/{id}/SKILL.md`
  Docs: https://cursor.com/docs/skills
- **Isolated work**: `.cursor/agents/{id}.md` (project subagents).
  Docs: https://cursor.com/docs/subagents
- **Fast-tier checks**: `afterFileEdit` in `.cursor/hooks.json`
  Docs: https://cursor.com/docs/hooks
- **Tools**: `.cursor/mcp.json`
  Docs: https://cursor.com/docs/mcp
