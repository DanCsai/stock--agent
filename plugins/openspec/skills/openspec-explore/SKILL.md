---
name: openspec-explore
description: Enter explore mode for OpenSpec work. Use when the user wants to investigate problems, compare options, or clarify requirements before or during a change.
license: MIT
compatibility: Requires openspec CLI.
metadata:
  author: openspec
  version: "1.0"
  generatedBy: "migrated-for-codex"
---

Enter explore mode. Think deeply. Visualize freely. Follow the conversation wherever it goes.

IMPORTANT: Explore mode is for thinking, not implementing. You may read files, search code, and investigate the codebase, but you must not write production code as part of this mode. If the user asks to implement something, move them to a proposal or apply workflow first. You may create OpenSpec artifacts if the user explicitly wants to capture the discussion.

This is a stance, not a rigid workflow.

## Approach

- Be curious, not prescriptive.
- Surface multiple directions instead of forcing one path.
- Use ASCII diagrams when they clarify structure or tradeoffs.
- Ground the discussion in the actual codebase when relevant.
- Call out risks, assumptions, and unknowns.

## OpenSpec Awareness

At the start, inspect current context:

```bash
openspec list --json
```

If there is an active change relevant to the discussion, read the existing artifacts under `openspec/changes/<name>/` and reference them naturally.

When decisions solidify, offer to capture them in the appropriate artifact:

- Scope or motivation: `proposal.md`
- Design decisions: `design.md`
- Requirements or deltas: `specs/<capability>/spec.md`
- Work items: `tasks.md`

Keep the interaction exploratory unless the user explicitly asks to formalize or implement.
