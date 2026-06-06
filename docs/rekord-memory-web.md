---
title: Rekord Memory
description: Persistent shared memory, snapshots, and resume context for coding agents.
version: 0.3.0
status: upcoming
---

# Rekord Memory

Rekord Memory is a user-local shared memory layer for humans and coding agents.

It lets agents remember what happened, what changed, what failed, and where work should continue, even after a terminal closes or a different agent takes over.

## The Problem

Coding agents lose context across sessions.

Claude, Codex, Cursor, OpenCode, Goose, Aider, and other agents each keep their own short-lived view of work. When users switch tools, close terminals, or hand off from one agent to another, useful context disappears.

Git remembers code history. Rekord remembers work history.

## What Rekord Memory Adds

Rekord `0.3.0` adds persistent project memory on top of terminal recording:

- Durable project memories
- Git-aware snapshots with full patches
- Agent-to-agent handoff
- Resume context for interrupted work
- Named session linkage
- MCP tools so agents can read and write memory directly

Memory is stored locally by default under:

```text
~/.rekord/projects/<project-hash>/
```

Rekord does not write memory files into your repository by default.

## Core Workflow

Store something important:

```bash
rekord remember "Parser refactor stopped at the failing unicode fixture"
```

Capture a stopping point:

```bash
rekord snapshot "Implemented parser refactor; tests still failing"
```

Search project memory:

```bash
rekord recall parser
```

Resume work later:

```bash
rekord resume
```

## Agent-To-Agent Handoff

Rekord Memory can scope context by agent.

If Claude started the work:

```bash
rekord snapshot --agent=claude "Stopped at failing refresh-token test"
```

Codex, OpenCode, or another agent can continue from Claude's context:

```bash
rekord resume --from-agent=claude --to-agent=codex
```

The output includes the latest snapshot, relevant memories, changed files, patch files, blockers, and continuation context.

## Named Sessions

Agents can name Rekord sessions and tell users how to resume them.

Example session name:

```text
memory-mvp-claude
```

Store memory linked to that session:

```bash
rekord remember --agent=claude --session=memory-mvp-claude \
  "Auth middleware refactor is incomplete"
```

Create a snapshot linked to that session:

```bash
rekord snapshot --agent=claude --session=memory-mvp-claude \
  "Stopped after debugging token expiry"
```

Resume from that exact session:

```bash
rekord resume --session=memory-mvp-claude
```

Recommended agent message:

```text
I started a Rekord session named `memory-mvp-claude`.

If this terminal closes, another agent can continue with:

rekord resume --session memory-mvp-claude
```

## Git-Aware Snapshots

`rekord snapshot` captures project state:

- Current branch
- Current HEAD
- Dirty status
- Changed files
- Full unstaged patch
- Full staged patch

Patch files are written locally under:

```text
~/.rekord/projects/<project-hash>/patches/
```

This makes snapshots useful for review, recovery, and handoff.

## CLI Reference

```bash
rekord remember <text>
rekord recall [query]
rekord resume
rekord snapshot [note]
```

Full memory management:

```bash
rekord memory add <text>
rekord memory list
rekord memory search <query>
rekord memory show <id>
rekord memory resolve <id>
```

Useful filters:

```bash
rekord resume --agent=claude
rekord resume --from-agent=claude --to-agent=codex
rekord resume --session=memory-mvp-claude
rekord recall --agent=opencode memory
rekord snapshot --agent=claude --session=auth-refresh-claude "Stopped at failing test"
```

## MCP Tools

Rekord exposes memory to agents through MCP:

```text
memory_write
memory_search
memory_list
memory_get
memory_resolve
snapshot_create
resume_context
```

The most important tool is `resume_context`.

Agents can call it at startup to answer:

- What was happening in this project?
- What did the previous agent do?
- What files changed?
- What patches exist?
- What blockers are still open?
- Where should work continue?

## Positioning

Rekord is still a terminal workflow recorder.

Memory makes Rekord the continuity layer for agentic development:

```text
Record what happened.
Snapshot where work stopped.
Remember what matters.
Resume with the next agent.
```

## One-Liner

Rekord Memory is persistent shared memory for coding agents, built into Rekord.
