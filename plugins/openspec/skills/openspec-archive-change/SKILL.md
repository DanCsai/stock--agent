---
name: openspec-archive-change
description: Archive a completed OpenSpec change. Use when the user wants to finalize a change after implementation and artifact work are done.
license: MIT
compatibility: Requires openspec CLI.
metadata:
  author: openspec
  version: "1.0"
  generatedBy: "migrated-for-codex"
---

Archive a completed OpenSpec change.

## Workflow

1. Determine the change name.
   - Use the provided one when available.
   - Otherwise run `openspec list --json` and ask the user to choose.

2. Inspect artifact completion:

```bash
openspec status --change "<name>" --json
```

3. Check the tasks file for incomplete items. Warn if tasks remain unfinished.

4. If delta specs exist under `openspec/changes/<name>/specs/`, compare them with `openspec/specs/` and explain what would sync.

5. Archive the change under the date-based archive path:

```bash
mkdir -p openspec/changes/archive
mv openspec/changes/<name> openspec/changes/archive/YYYY-MM-DD-<name>
```

6. Summarize the archive result, including warnings or skipped sync.

## Guardrails

- Do not silently guess the target change when multiple active changes exist.
- Warn before archiving incomplete artifacts or tasks.
- Preserve the full change directory, including `.openspec.yaml`.
