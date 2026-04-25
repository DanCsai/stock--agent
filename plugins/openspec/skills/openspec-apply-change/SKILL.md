---
name: openspec-apply-change
description: Implement tasks from an OpenSpec change. Use when the user wants to start or continue execution against an existing change.
license: MIT
compatibility: Requires openspec CLI.
metadata:
  author: openspec
  version: "1.0"
  generatedBy: "migrated-for-codex"
---

Implement tasks from an OpenSpec change.

## Workflow

1. Select the target change.
   - Use the provided name if present.
   - Otherwise infer from context when safe.
   - If ambiguous, run `openspec list --json` and ask the user to choose.

2. Inspect current state:

```bash
openspec status --change "<name>" --json
openspec instructions apply --change "<name>" --json
```

3. Read all context files returned by the apply instructions.

4. Show current progress, then implement pending tasks one by one.

5. After finishing each task, update the tasks file from `- [ ]` to `- [x]`.

6. Stop and ask for guidance if:
   - A task is unclear
   - Implementation exposes a design issue
   - A blocker prevents safe progress

7. On completion or pause, summarize completed work and the remaining task count.

## Guardrails

- Keep edits focused on the current task.
- Respect the schema and the context files returned by the CLI.
- If the change is fully complete, suggest archiving it.
