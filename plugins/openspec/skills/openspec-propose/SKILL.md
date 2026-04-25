---
name: openspec-propose
description: Create an OpenSpec change proposal with implementation-ready artifacts. Use when the user wants to define what to build and generate proposal, design, and task documents.
license: MIT
compatibility: Requires openspec CLI.
metadata:
  author: openspec
  version: "1.0"
  generatedBy: "migrated-for-codex"
---

Propose a new change and generate the artifacts needed before implementation.

When ready to implement, continue with `openspec-apply-change`.

## Input

The request should include either:

- A change name in kebab-case, or
- A clear description of what should be built or fixed

If the request is unclear, ask the user what they want to build before proceeding.

## Workflow

1. Create the change:

```bash
openspec new change "<name>"
```

2. Inspect artifact status and build order:

```bash
openspec status --change "<name>" --json
```

3. For each ready artifact, fetch instructions:

```bash
openspec instructions <artifact-id> --change "<name>" --json
```

4. Read any dependency artifacts, then create the target artifact using the returned template and guidance.

5. Re-check status until all apply-required artifacts are done.

6. Show final status:

```bash
openspec status --change "<name>"
```

## Guardrails

- Do not copy CLI `context` or `rules` blocks into output files.
- Read dependency artifacts before creating downstream artifacts.
- If context is critically unclear, ask the user; otherwise keep momentum.
- If the change already exists, ask whether to continue it or create a new one.
