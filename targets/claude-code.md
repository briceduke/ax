---
id: claude-code
instructions: CLAUDE.md
invocable: .claude/skills/{id}/SKILL.md
isolated: .claude/agents/{id}.md
hooks: .claude/settings.json
hooks_format: claude-settings
mcp: .mcp.json
---

# Claude Code

Where each core piece lands. The assembler uses only these paths.

- **Instructions** (intent + always-on rules): `CLAUDE.md` at the project root.
  Docs: https://code.claude.com/docs/en/memory
- **User-invocable capabilities**: `.claude/skills/{id}/SKILL.md`. Skills and slash commands both create `/{id}`; prefer skills.
  Docs: https://code.claude.com/docs/en/slash-commands
- **Isolated work**: `.claude/agents/{id}.md`
  Docs: https://code.claude.com/docs/en/sub-agents
- **Fast-tier checks**: PostToolUse on Write|Edit in `.claude/settings.json` (committed project settings; personal overrides go in `settings.local.json`).
  Docs: https://code.claude.com/docs/en/hooks
- **Tools**: `.mcp.json`
  Docs: https://code.claude.com/docs/en/mcp
