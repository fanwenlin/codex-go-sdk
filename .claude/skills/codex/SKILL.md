---
name: codex
description: Delegate coding tasks to OpenAI Codex. Use when you've got a solution and need an engineer to finish the code work fulfilling the spec.
---

# Codex Skill

Delegate coding work to OpenAI Codex by providing a task directory.

## When to Use

- Generating new code from requirements
- Implementing features described in documents
- Complex coding tasks that benefit from Codex's capabilities

## Quick Usage

```sh
codex-orchestrator <task-dir>
```

## Task Directory Format

Create a directory with your requirements:

```
my-task/
├── task.md          # What to build (required)
├── acceptance.md    # Success criteria (optional)
└── context.md       # Background info (optional)
```

## Example

```sh
# Create task
mkdir my-task
echo "Create a function to calculate fibonacci numbers" > my-task/task.md
echo "- Named 'fibonacci'\n- Handles negative input\n- Has tests" > my-task/acceptance.md

# Run
codex-orchestrator my-task
```

## Options

| Flag | Description |
|------|-------------|
| `-s <dir>` | Custom coding standards/skills |
| `--max-file-bytes N` | Per-file size limit |
| `--max-total-bytes N` | Total prompt size limit |

## Output

Returns Codex's response directly. Pass through to user without modification.
